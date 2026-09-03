package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// stuckBoard is a board whose last n runs all ended stuck, newest last.
func stuckBoard(n int) []view.Task {
	var tasks []view.Task

	for i := range n {
		tasks = append(tasks, view.Task{
			Repo: "payments", ID: "ACME-" + string(rune('1'+i)), Title: "a task that will not pass its gate",
			Band: view.NeedsYou, Since: ago(time.Duration(n-i) * time.Minute),
			Reason: view.Reason{Key: view.ReasonStuck, Args: []view.Arg{arg("attempts", "3")}},
		})
	}

	return tasks
}

// TestThreeStuckInARowTakeAutopilotOff. When every task the queue picks up
// comes back stuck, the problem is the repository or the plan and not the
// task — and an autopilot that keeps picking up the next one is spending
// money to prove that three times an hour.
func TestThreeStuckInARowTakeAutopilotOff(t *testing.T) {
	m, rec := testModel(t, 100, 30)
	m.seen = true

	var said string

	m.opts.RecordSupervisor = func(_, _, message string) error {
		said = message
		return nil
	}

	next, _ := m.applyBoard(boardMsg{Board: fixtureBoard(stuckBoard(3), 1)})
	got := asModel(t, next)

	if m.opts.Settings.Autopilot() {
		t.Error("three stuck runs in a row left autopilot on")
	}

	if !strings.Contains(said, "3") || !strings.Contains(strings.ToLower(said), "stuck") {
		t.Errorf("the supervisor thread was told %q, want a line naming the three stuck runs", said)
	}

	if !strings.Contains(strings.ToLower(got.message), "autopilot") {
		t.Errorf("the band says %q, want it to say autopilot went off", got.message)
	}

	if rec.flow != "" {
		t.Error("autopilot started a task on the board that tripped the breaker")
	}
}

// TestTwoStuckInARowLeaveAutopilotAlone. The breaker is for a board where
// nothing is working, and two is not that: a task that got stuck twice is a
// task, and switching the queue off for it would be Orbit deciding a run is
// not worth trying.
func TestTwoStuckInARowLeaveAutopilotAlone(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.seen = true

	next, _ := m.applyBoard(boardMsg{Board: fixtureBoard(stuckBoard(2), 1)})
	if !asModel(t, next).opts.Settings.Autopilot() {
		t.Error("two stuck runs took autopilot off")
	}
}

// TestTheBreakerDoesNotTripTwiceOverTheSameBoard. The poll is twice a
// second and the board does not change on its own: a breaker that says so
// on every read owns the band and writes the supervisor a line a second.
func TestTheBreakerDoesNotTripTwiceOverTheSameBoard(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.seen = true

	lines := 0
	m.opts.RecordSupervisor = func(_, _, _ string) error {
		lines++
		return nil
	}

	b := fixtureBoard(stuckBoard(3), 1)

	next, _ := m.applyBoard(boardMsg{Board: b})
	again, _ := asModel(t, next).applyBoard(boardMsg{Board: b})

	_ = asModel(t, again)

	if lines != 1 {
		t.Errorf("the breaker wrote %d supervisor lines over one board, want one", lines)
	}
}

// TestABreakerThatCannotFlipTheSwitchSaysSo. The settings file has a lock,
// and a second orbit holding it makes the write refuse. A breaker that
// swallowed that would leave the queue picking up tasks while the band said
// nothing at all.
func TestABreakerThatCannotFlipTheSwitchSaysSo(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.seen = true

	held, ok := m.opts.Settings.(*settings)
	if !ok {
		t.Fatalf("the window's settings port is %T, want the fixture's", m.opts.Settings)
	}

	held.fail = errors.New("the settings file is held by another orbit")

	next, _ := m.applyBoard(boardMsg{Board: fixtureBoard(stuckBoard(3), 1)})
	if got := asModel(t, next).message; !strings.Contains(got, "held by another orbit") {
		t.Errorf("the band says %q, want the refusal the settings file gave", got)
	}
}
