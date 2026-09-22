package ui

// The cancel a reader actually presses.
//
// `x` on the board asks first, and the answer to that question went down a
// line of its own that wrote the control word — which a run reads at its
// next phase boundary. So the signalled cancel was built, tested, merged,
// and never reached by the key it was built for: Elio pressed x on a phase
// a minute old, watched the row go on saying "implement" and the task go
// on spending, and had no way to tell whether anything had been taken.
//
// These press the key.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// pressed sends one keystroke to a window and gives back what it became.
func pressed(t *testing.T, m Model, k string) (Model, tea.Cmd) {
	t.Helper()

	next, cmd := m.Update(tea.KeyPressMsg{Code: rune(k[0]), Text: k})

	return asModel(t, next), cmd
}

// TestConfirmingACancelSignalsTheRun.
func TestConfirmingACancelSignalsTheRun(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	var (
		signalled string
		worded    string
	)

	m.opts.Stop = func(task view.Task) error { signalled = task.ID; return nil }
	m.opts.Control = func(task view.Task, word string) error {
		worded = task.ID + ":" + word

		return nil
	}

	// Onto a running task, then x and yes.
	m = onto(t, m, runningTask(t, m))

	m, _ = pressed(t, m, "x")
	if m.confirm != confirmCancel {
		t.Fatalf("x left confirm %v, want the cancel question", m.confirm)
	}

	m, cmd := pressed(t, m, confirmYes)
	if cmd == nil {
		t.Fatal("confirming the cancel ran nothing")
	}

	cmd()

	if signalled == "" {
		t.Error("the run was not signalled: the cancel went to the control file, " +
			"which is read at the next phase boundary")
	}

	if worded != "" {
		t.Errorf("the cancel wrote %q instead of signalling", worded)
	}
}

// TestTheRowSaysItIsCancellingTheMomentYouConfirm. Elio: this has to be
// immediate, and if it is not there is no point.
func TestTheRowSaysItIsCancellingTheMomentYouConfirm(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.opts.Stop = func(view.Task) error { return nil }

	id := runningTask(t, m)
	m = onto(t, m, id)

	m, _ = pressed(t, m, "x")
	m, _ = pressed(t, m, confirmYes)

	if m.await.id != id || m.await.verb != gestureCancel {
		t.Fatalf("after confirming, the window is awaiting %+v", m.await)
	}

	word, out := m.awaitedWord(id)
	if !out || !strings.Contains(word, "cancel") {
		t.Errorf("the status cell says %q, want it to say it is cancelling", word)
	}

	if !strings.Contains(m.message, "cancel") {
		t.Errorf("the band says %q, want it to say it is cancelling", m.message)
	}
}

// TestAWindowWithNoStopPortStillCancels: a window built without one falls
// back to the word rather than doing nothing.
func TestAWindowWithNoStopPortStillCancels(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	worded := ""
	m.opts.Stop = nil
	m.opts.Control = func(task view.Task, word string) error {
		worded = task.ID + ":" + word

		return nil
	}

	m = onto(t, m, runningTask(t, m))
	m, _ = pressed(t, m, "x")

	_, cmd := pressed(t, m, confirmYes)
	if cmd == nil {
		t.Fatal("confirming ran nothing")
	}

	cmd()

	if !strings.HasSuffix(worded, ":cancel") {
		t.Errorf("with no stop port the cancel wrote %q", worded)
	}
}

// runningTask is the id of a task on the fixture board that a cancel is
// allowed on.
func runningTask(t *testing.T, m Model) string {
	t.Helper()

	for _, task := range m.board.Tasks {
		if view.BandOf(task) == view.Running {
			return task.ID
		}
	}

	t.Fatal("no running task on the fixture board to cancel")

	return ""
}
