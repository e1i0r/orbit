package menu

import "testing"

// The menu itself: whether it is up, and what it is about.

// TestUpSaysWhetherTheMenuIsShowing.
func TestUpSaysWhetherTheMenuIsShowing(t *testing.T) {
	if (State{}).Up() {
		t.Error("a menu that never opened reads as up")
	}

	if !Open("", world(t)).Up() {
		t.Error("an opened menu reads as down")
	}
}

// TestTaskNamesWhatTheMenuIsAbout. Empty on the board's own menu, which
// is the menu of no row in particular.
func TestTaskNamesWhatTheMenuIsAbout(t *testing.T) {
	if got := Open("", world(t)).Task(); got != "" {
		t.Errorf("the board's menu is about %q", got)
	}

	if got := Open(theTask, world(t)).Task(); got != theTask {
		t.Errorf("the task's menu is about %q, want %q", got, theTask)
	}
}
