package quota

// Package quota queries the Anthropic / proxy quota endpoint (GET /quota)
// to discover remaining limits and reset intervals.
//
// Absence of the proxy is not a failure: when ANTHROPIC_BASE_URL is unset,
// or unreachable, no quota is reported and no error is raised.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// CacheDuration is how long a good quota read is cached.
const CacheDuration = 30 * time.Second

// firstBackoff and maxBackoff are how long a failed read is not retried for,
// doubling from one to the other.
//
// The status bar asks for the quota on every frame it draws, which is several
// times a second while anything is moving, so a proxy that is down used to
// mean a connection attempt per frame for as long as the window stayed open:
// nothing was written down after a failure, so the next look found the cache
// cold and sent another. The cap is a minute rather than longer because this
// is a number a person is watching — a proxy that comes back is worth seeing
// again inside a minute — and a request a minute is not a loop.
const (
	firstBackoff = 5 * time.Second
	maxBackoff   = time.Minute
)

// MaxKeyLen is the maximum length of an account key preserved for display.
const MaxKeyLen = 12

// Window is one quota window: percentage and countdown clock.
type Window struct {
	Key      string
	Label    string
	Pct      float64
	ResetsIn time.Duration
}

// Client fetches and caches quota windows from the proxy.
type Client struct {
	baseURL string
	client  *http.Client

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

// New builds a Client for the given base URL. An empty base URL yields nil.
func New(baseURL string) *Client {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 1 * time.Second},
	}
}

// FromEnv builds a Client from ANTHROPIC_BASE_URL.
func FromEnv() *Client {
	return New(os.Getenv("ANTHROPIC_BASE_URL"))
}

// Fetch requests GET /quota from the base URL synchronously.
func (c *Client) Fetch() ([]Window, error) {
	if c == nil || c.baseURL == "" {
		return nil, nil
	}
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/quota", nil)
	if err != nil {
		return nil, fmt.Errorf("build quota request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch quota: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("quota HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read quota body: %w", err)
	}
	return parseWindows(body)
}

// Quota returns the cached quota windows immediately. If the cache is cold,
// a background goroutine is spawned to refresh it unless wait is true (used
// for single-frame CLI commands like --once).
//
// Either way it asks at most as often as keep allows. What it used to do
// with a proxy that was not answering was ask again on the next frame, and
// the one after that, because a failure left no trace: the cache stayed cold,
// so every look at the status bar started another request.
func (c *Client) Quota(wait bool) []Window {
	if c == nil || c.baseURL == "" {
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

type wireWindow struct {
	Key       string  `json:"key"`
	Label     string  `json:"label"`
	Pct       float64 `json:"pct"`
	ResetsInS int64   `json:"resets_in_s"`
	ResetsIn  int64   `json:"resets_in"`
}

func parseWindows(body []byte) ([]Window, error) {
	var list []wireWindow
	if err := json.Unmarshal(body, &list); err == nil {
		return toWindows(list), nil
	}
	var wrapper struct {
		Windows []wireWindow `json:"windows"`
		Quota   []wireWindow `json:"quota"`
	}
	if err := json.Unmarshal(body, &wrapper); err == nil {
		if len(wrapper.Windows) > 0 {
			return toWindows(wrapper.Windows), nil
		}
		if len(wrapper.Quota) > 0 {
			return toWindows(wrapper.Quota), nil
		}
	}
	var single wireWindow
	if err := json.Unmarshal(body, &single); err == nil && (single.Label != "" || single.Pct > 0) {
		return toWindows([]wireWindow{single}), nil
	}
	return nil, errors.New("unrecognised quota response format")
}

func toWindows(list []wireWindow) []Window {
	out := make([]Window, 0, len(list))
	for _, w := range list {
		key := w.Key
		if len(key) > MaxKeyLen {
			key = key[:MaxKeyLen]
		}
		res := w.ResetsInS
		if res == 0 {
			res = w.ResetsIn
		}
		out = append(out, Window{
			Key:      key,
			Label:    w.Label,
			Pct:      w.Pct,
			ResetsIn: time.Duration(res) * time.Second,
		})
	}
	return out
}
