package ui

// r on the task view: the key the needs-you banner names, on the screen it
// names it from.

import "testing"

// TestResumeIsAnsweredOnTheTaskViewAndNotOnlyOnTheBoard.
//
// The key was in the board's map and in no other. A reader who opened the
// task the banner is drawn on and pressed r fell through to the pane's
// scroll: the run stayed parked, nothing was written and nothing was said.
func TestResumeIsAnsweredOnTheTaskViewAndNotOnlyOnTheBoard(t *testing.T) {
	m, got := parkedModel(t)
	m.screen, m.detail = screenDetail, "ACME-2705"

	_, cmd := advance(t, m, press("r"))
	wantControl(t, cmd, got, "ACME-2705", "resume")
}

// TestTheTaskViewSaysWhyAResumeIsRefused is the other half of the same
// promise: a key that is answered says something even when the answer is no.
func TestTheTaskViewSaysWhyAResumeIsRefused(t *testing.T) {
	m, got := testModel(t, 100, 30)
	m.screen, m.detail = screenDetail, "ACME-2710"

	after, cmd := advance(t, m, press("r"))
	if cmd != nil {
		t.Fatal("r on a task nothing is running produced a command")
	}

	if got.word != "" {
		t.Errorf("the control port was asked to write %q on a task with no run", got.word)
	}

	if after.message == "" {
		t.Error("r on a task that cannot be resumed said nothing at all")
	}
}

// TestTheDiffTabKeepsItsOwnR is the case the new route is matched under: on
// the diff tab r is the rationale, and a reader reading a change is not
// resuming anything with it.
func TestTheDiffTabKeepsItsOwnR(t *testing.T) {
	m, got := parkedModel(t)
	m.screen, m.detail, m.tab = screenDetail, "ACME-2705", tabDiff

	after, _ := advance(t, m, press("r"))
	if !after.hideDiffRationale {
		t.Error("r on the diff tab no longer hides the rationale")
	}

	if got.word != "" {
		t.Errorf("r on the diff tab wrote the control word %q", got.word)
	}
}
