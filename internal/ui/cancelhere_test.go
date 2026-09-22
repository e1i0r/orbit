package ui

// The cancel pressed on a task's own screen.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// TestXOnATasksScreenCancelsThatTask. The menu's cancel is the x key, and
// on a task's screen x had no case of its own: it fell through to scroll,
// which has no use for it. Elio chose cancel from the menu of a task
// waiting at review and nothing happened, while the same entry on the
// board asked and cancelled.
func TestXOnATasksScreenCancelsThatTask(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	var signalled string

	m.opts.Stop = func(task view.Task) error { signalled = task.ID; return nil }

	id := runningTask(t, m)
	m = onto(t, m, id)

	task, ok := m.task(id)
	if !ok {
		t.Fatalf("%s is not on the board", id)
	}

	m, _ = m.openDetail(task)

	m, _ = pressed(t, m, "x")
	if m.confirm != confirmCancel || m.confirmID != task.ID {
		t.Fatalf("x on %s's screen left confirm %v about %q, want the cancel question about it",
			task.ID, m.confirm, m.confirmID)
	}

	m, cmd := pressed(t, m, confirmYes)
	if cmd == nil {
		t.Fatal("confirming the cancel ran nothing")
	}

	cmd()

	if signalled != task.ID {
		t.Errorf("the cancel signalled %q, want %s", signalled, task.ID)
	}
}
