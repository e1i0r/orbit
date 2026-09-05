package ui

// The one line that says what the window is waiting on.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// TestEveryWaitSaysTheSameFourThings: that it is moving, what it is, how
// long it has been, and the key that ends the wait when there is one.
func TestEveryWaitSaysTheSameFourThings(t *testing.T) {
	m, _ := testModel(t, 140, 30)
	m.now = time.Now()

	cases := []struct {
		name string
		set  func(Model) Model
		says string
		stop bool
	}{
		{
			name: "a draft",
			set: func(m Model) Model {
				m.flows.saying, m.flows.sayAt = true, m.now.Add(-42*time.Second)
				return m
			},
			says: "42s",
			stop: true,
		},
		{
			name: "the supervisor",
			set: func(m Model) Model {
				m.supervisorBusy, m.supervisorAt = true, m.now.Add(-9*time.Second)
				return m
			},
			says: "9s",
		},
		{
			name: "a delivery",
			set: func(m Model) Model {
				m.delivering = deliverPending{task: view.Task{ID: "X-1"}, verb: "merge", at: m.now.Add(-3 * time.Second)}
				return m
			},
			says: "X-1",
		},
	}

	for _, c := range cases {
		next := c.set(m)

		line := next.waitingLine()
		if line == "" {
			t.Errorf("%s: the band says nothing", c.name)
			continue
		}

		if !strings.Contains(line, c.says) {
			t.Errorf("%s: the line does not say %q:\n%s", c.name, c.says, line)
		}

		// The spinner is what says it is moving rather than wedged.
		if !strings.Contains(line, next.spin()) {
			t.Errorf("%s: the line does not turn:\n%s", c.name, line)
		}

		if held := strings.Contains(line, "esc"); held != c.stop {
			t.Errorf("%s: offers a way to stop = %v, want %v", c.name, held, c.stop)
		}

		// And the band draws it over anything that merely happened.
		next.message, next.messageAt = "something finished", next.now
		if got := next.bandLeft(); !strings.Contains(got, c.says) {
			t.Errorf("%s: a stale message covered the wait:\n%s", c.name, got)
		}
	}
}

// TestTwoWaitsAtOnceSayHowManyThereAre, rather than one of them quietly
// standing for both.
func TestTwoWaitsAtOnceSayHowManyThereAre(t *testing.T) {
	m, _ := testModel(t, 140, 30)
	m.now = time.Now()
	m.flows.saying, m.flows.sayAt = true, m.now.Add(-time.Second)
	m.supervisorBusy, m.supervisorAt = true, m.now

	if got := m.waitingLine(); !strings.Contains(got, "1") {
		t.Errorf("two waits drew %q, want the count of the rest in it", got)
	}
}

// TestNothingWaitingSaysNothing, so the band goes back to the board.
func TestNothingWaitingSaysNothing(t *testing.T) {
	m, _ := testModel(t, 140, 30)

	if got := m.waitingLine(); got != "" {
		t.Errorf("an idle window is waiting on %q", got)
	}
}
