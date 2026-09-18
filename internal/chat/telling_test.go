package chat

// The other direction: what Orbit says without being asked.
//
// Everything in serve_test.go happens because somebody wrote a message.
// Nothing here does, and that is the whole difference between a chat you
// have to remember to open and one that reaches you — so the half that read
// almost nothing is the half that is the point.

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// aWatchingChannel is a chat that holds its line open until the test stops
// it, so the desk's outward half has somewhere to push into.
type aWatchingChannel struct {
	aChannel

	// typing counts how often the reader was shown that an answer is
	// coming, which is what a Working channel is for.
	mu     sync.Mutex
	typing int
	// stop ends Listen, because a desk that is serving holds it open.
	stop chan struct{}
}

func (c *aWatchingChannel) Listen(ctx context.Context, said func(Message)) error {
	for _, m := range c.hears {
		said(m)
	}

	select {
	case <-ctx.Done():
	case <-c.stop:
	}

	return nil
}

func (c *aWatchingChannel) Working(context.Context, string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.typing++

	return nil
}

func (c *aWatchingChannel) shown() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.typing
}

// deskOn is a desk over that channel, with an Env the caller has shaped.
func deskOn(t *testing.T, c Channel, shape func(*Env)) *Desk {
	t.Helper()

	e := Env{
		Words:   words.For("en"),
		Allowed: anyone,
		World: func() (verb.World, func(), error) {
			return nil, func() {}, errNoWorld
		},
	}

	shape(&e)

	return Open(e, c)
}

// TestNewsReachesTheConversationItIsToldTo. A poll and not a hook, because a
// run is its own process: it can be killed, and a notification that depended
// on the dying process to send it is the one that never arrives for the task
// that most needed it.
func TestNewsReachesTheConversationItIsToldTo(t *testing.T) {
	c := &aWatchingChannel{stop: make(chan struct{})}

	var (
		elsewhere []string
		mu        sync.Mutex
	)

	happened := []Happening{{
		Task:  "ACME-1",
		Event: record.Event{Kind: record.TaskFinished, At: time.Now()},
	}}

	var once sync.Once

	d := deskOn(t, c, func(e *Env) {
		e.Tells = "here"
		e.Also = func(said string) {
			mu.Lock()
			defer mu.Unlock()

			elsewhere = append(elsewhere, said)
		}
		e.Watch = func(context.Context) ([]Happening, error) {
			var out []Happening

			once.Do(func() { out = happened })

			return out, nil
		}
	})

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	go d.telling(ctx)

	// The tick is seconds, because nobody reaching for a phone can tell
	// four seconds from one.
	until(t, func() bool { return len(c.heard()) > 0 })

	said := c.heard()
	if !strings.Contains(said[0], "ACME-1") {
		t.Errorf("it said %q, want the task it is about", said[0])
	}

	mu.Lock()
	also := len(elsewhere)
	mu.Unlock()

	if also == 0 {
		t.Error("the same news reached the conversation and nowhere else")
	}
}

// TestAChatToldNothingSaysNothing. The switch is one for the lot, and a desk
// with nothing behind it must not sit in a loop asking a record it has no
// handle on.
func TestAChatToldNothingSaysNothing(t *testing.T) {
	cases := []struct {
		name  string
		shape func(*Env)
	}{
		{"nothing to watch", func(e *Env) { e.Tells = "here" }},
		{
			"nowhere to say it",
			func(e *Env) {
				e.Watch = func(context.Context) ([]Happening, error) {
					t.Error("a desk with nowhere to say it asked the record")

					return nil, nil
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ch := &aWatchingChannel{stop: make(chan struct{})}
			d := deskOn(t, ch, c.shape)

			ctx, stop := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer stop()

			d.telling(ctx)

			if len(ch.heard()) != 0 {
				t.Errorf("it said %v", ch.heard())
			}
		})
	}
}

// TestARecordThatCannotBeReadDoesNotEndTheTelling. Orbit is a program
// somebody leaves running: a read that failed once is a line in the log, and
// a loop that ended over it is a chat silently off for the afternoon.
func TestARecordThatCannotBeReadDoesNotEndTheTelling(t *testing.T) {
	c := &aWatchingChannel{stop: make(chan struct{})}

	var asked int

	var mu sync.Mutex

	d := deskOn(t, c, func(e *Env) {
		e.Tells = "here"
		e.Watch = func(context.Context) ([]Happening, error) {
			mu.Lock()
			defer mu.Unlock()

			asked++

			return nil, errNoWorld
		}
	})

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	go d.telling(ctx)

	until(t, func() bool {
		mu.Lock()
		defer mu.Unlock()

		return asked >= 2
	})
}

// TestTheReaderIsShownAnAnswerIsComing. Started for every message and not
// only the slow ones, because which ones are slow is not knowable here: a
// verb that reads the board is instant and one that walks a repository's
// history is not, and guessing wrong in the second direction is the silence
// this exists to remove.
func TestTheReaderIsShownAnAnswerIsComing(t *testing.T) {
	c := &aWatchingChannel{stop: make(chan struct{})}
	c.hears = []Message{{Where: "here", Who: "elio", Text: "/board"}}

	d := deskOn(t, c, func(*Env) {})

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	done := make(chan error, 1)
	go func() { done <- d.Serve(ctx) }()

	until(t, func() bool { return len(c.heard()) > 0 })

	if c.shown() == 0 {
		t.Error("the reader was left with silence while the verb ran")
	}

	close(c.stop)

	if err := <-done; err != nil {
		t.Errorf("serving ended with %v", err)
	}
}

// TestAChannelThatCannotShowTypingIsStillServed. The indicator is optional —
// a terminal has no menu to put one in either — and a desk that refused to
// answer without one would be a chat that stopped over a nicety.
func TestAChannelThatCannotShowTypingIsStillServed(t *testing.T) {
	said := served(t, anyone, "/board")
	if len(said) != 1 {
		t.Errorf("a channel with no typing indicator answered %v", said)
	}
}

// until spins until the condition holds or the test has waited long enough
// to say the thing never happened.
//
// Long enough is generous on purpose: the tick under test is four seconds,
// and a suite running fifty packages at once is not the place to measure a
// timer to the millisecond.
func until(t *testing.T, ok func() bool) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("it never happened")
}
