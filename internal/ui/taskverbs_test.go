package ui

// The verbs about a task, from that task's own screen.

import "testing"

// onScreenOf is the window on id's screen, with the board's cursor left on
// some other task: what a verb acts on has to be the task being read.
func onScreenOf(t *testing.T, id string) Model {
	t.Helper()

	m, _ := testModel(t, 100, 30)

	for _, r := range m.rows() {
		if !r.head && !r.blank && r.task.ID != id {
			m = onRow(t, m, r.task.ID)

			break
		}
	}

	task, ok := m.task(id)
	if !ok {
		t.Fatalf("%s is not on the board", id)
	}

	m, _ = m.openDetail(task)

	return m
}

// TestPauseFromATasksMenuPausesThatTask. p is the pull request on a task's
// screen, so the menu's pause, which is sent as p, opened the delivery
// instead, and about the task the board's cursor was on.
func TestPauseFromATasksMenuPausesThatTask(t *testing.T) {
	m := onScreenOf(t, "ACME-2705").openMenu("ACME-2705")

	_, cmd := pointAt(t, m, "p").chooseMenu()
	if cmd == nil {
		t.Fatal("choosing pause on a task's screen did nothing")
	}

	got, ok := cmd().(controlMsg)
	if !ok || got.ID != "ACME-2705" || got.Word != "pause" {
		t.Errorf("choosing pause raised %+v, want pause about ACME-2705", cmd())
	}
}

// TestTheVerbsThisScreenLeftAreAnsweredOnIt. b and x ask before they act,
// and the question is about the task being read.
func TestTheVerbsThisScreenLeftAreAnsweredOnIt(t *testing.T) {
	for k, want := range map[string]confirm{"b": confirmRequeue, "x": confirmCancel} {
		m := onScreenOf(t, "ACME-2705")

		m, _ = pressed(t, m, k)
		if m.confirm != want || m.confirmID != "ACME-2705" {
			t.Errorf("%s on ACME-2705's screen asked %v about %q, want %v about it",
				k, m.confirm, m.confirmID, want)
		}
	}
}
