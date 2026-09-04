// Package quota is how much of an engine's allowance is left, and what a
// number about that engine's use means at all.
//
// Two questions, and they are one question. An engine paid per token is
// spoken about in money; an engine paid by subscription has no money to
// speak of — what it has is a share of a window and the hour that window
// comes back — and $0.42 shown to somebody on a fixed subscription is a
// figure nobody is charging them. A header, a budget screen and a stats
// screen that each decided this for themselves would disagree with each
// other in front of one reader, so the decision is made here, once, and they
// inherit it: see Mode.
//
// Where the number comes from differs per engine, and the shape it arrives
// in does not. A proxy that answers GET /quota is one source; an engine with
// no source at all is an answer of its own, and this package says so rather
// than reporting zero — "nobody can read codex's quota" and "codex has none
// left" are not the same sentence, and only one of them is true.
package quota

import (
	"sync"
	"time"
)

// CacheDuration is how long a good quota read is cached.
const CacheDuration = 30 * time.Second

// firstBackoff and maxBackoff are how long a failed read is not retried for,
// doubling from one to the other.
//
// The status bar asks for the quota on every frame it draws, which is several
// times a second while anything is moving. Unless a failure is written down,
// the next look finds the cache cold and sends another request, and a proxy
// that is down becomes a connection attempt per frame for as long as the
// window stays open. The cap is a minute rather than longer because this
// is a number a person is watching — a proxy that comes back is worth seeing
// again inside a minute — and a request a minute is not a loop.
const (
	firstBackoff = 5 * time.Second
	maxBackoff   = time.Minute
)

// Window is one quota window: percentage and countdown clock.
type Window struct {
	Key      string
	Label    string
	Pct      float64
	ResetsIn time.Duration
}

// Client is one source with the cache and the backoff in front of it.
type Client struct {
	src Source

	mu     sync.Mutex
	cached []Window
	// nextTry is when a fetch may go out again, and it is set after every
	// one of them: a good read holds it for CacheDuration, a failed read
	// for the backoff. fetching is the same idea over a shorter window —
	// one request in flight at a time — and neither is enough on its own.
	nextTry  time.Time
	backoff  time.Duration
	fetching bool
}

// New builds a Client over the proxy at the given base URL. An empty base
// URL yields nil, which is this package's way of saying there is nowhere to
// look: a nil Client answers nothing to everything, and the engine it was
// built for is reported as having no source rather than no quota left.
func New(baseURL string) *Client {
	src := newProxy(baseURL)
	if src == nil {
		return nil
	}

	return &Client{src: src}
}

// over puts the cache and the backoff in front of a source that is not a
// proxy. A nil source yields nil, which is this package's way of saying
// there is nowhere to look.
func over(src Source) *Client {
	if src == nil {
		return nil
	}

	return &Client{src: src}
}

// backed is a source with another one behind it, for an engine that can be
// read two ways.
type backed struct{ first, then Source }

// behind puts a second source behind a first: the first is asked, and the
// second answers when the first had nothing to say.
//
// A proxy that is up and a proxy that answers for this engine are two
// different things — a base URL naming one that has no idea what this engine
// is returns a 404 per look, and the engine reads as unreadable while the
// answer sits in a file on the same machine. Either half being nil is the
// other half alone, which is what makes a machine with no proxy and a
// machine with no rollouts both work out to one source rather than a case.
func behind(first, then Source) Source {
	switch {
	case first == nil:
		return then
	case then == nil:
		return first
	default:
		return backed{first: first, then: then}
	}
}

// Read is what the first source said, and what the second says when the
// first said nothing. A failure is nothing: the point of the second source
// is the times the first cannot answer.
func (b backed) Read() ([]Window, error) {
	if windows, err := b.first.Read(); err == nil && len(windows) > 0 {
		return windows, nil
	}

	return b.then.Read()
}

// Fetch asks the source once, synchronously, past the cache.
func (c *Client) Fetch() ([]Window, error) {
	if c == nil || c.src == nil {
		return nil, nil
	}

	return c.src.Read()
}

// Quota returns the cached quota windows immediately. If the cache is cold,
// a background goroutine is spawned to refresh it unless wait is true (used
// for single-frame CLI commands like --once).
//
// Either way it asks at most as often as keep allows. A failure leaves a
// trace for the same reason: without one the cache stays cold, and every
// look at the status bar starts another request against a proxy that is not
// answering.
func (c *Client) Quota(wait bool) []Window {
	if c == nil || c.src == nil {
		return nil
	}

	c.mu.Lock()
	if time.Now().Before(c.nextTry) {
		cached := c.cached
		c.mu.Unlock()

		return cached
	}

	if wait {
		c.mu.Unlock()
		windows, err := c.Fetch()
		c.mu.Lock()
		c.keep(windows, err)
		c.mu.Unlock()

		if err != nil {
			return nil
		}

		return windows
	}

	if !c.fetching {
		c.fetching = true

		c.mu.Unlock()
		go func() {
			windows, err := c.Fetch()
			c.mu.Lock()
			defer c.mu.Unlock()

			c.fetching = false
			c.keep(windows, err)
		}()

		c.mu.Lock()
	}

	cached := c.cached
	c.mu.Unlock()

	return cached
}

// keep files the outcome of one fetch and, with it, when the next one may go
// out. The lock is held by the caller.
//
// A failed read keeps whatever was cached before it. A quota reading that is
// a minute old is worth more to a reader than an empty corner of the status
// bar, and the alternative — clearing it — would make a proxy that hiccups
// look like a proxy that is gone.
func (c *Client) keep(windows []Window, err error) {
	if err != nil || windows == nil {
		c.backoff = min(max(c.backoff*2, firstBackoff), maxBackoff)
		c.nextTry = time.Now().Add(c.backoff)

		return
	}

	c.cached, c.backoff = windows, 0
	c.nextTry = time.Now().Add(CacheDuration)
}
