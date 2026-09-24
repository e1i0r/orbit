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

// TestOnTheDiffTabRResumesAndHShowsTheReasons. The diff tab took r for
// the engine's reasons, so a reader parked on it could not let the run go
// with the key that does it everywhere else. r is resume here too, and the
// reasons are H.
func TestOnTheDiffTabRResumesAndHShowsTheReasons(t *testing.T) {
	m, got := parkedModel(t)
	m.screen, m.detail, m.tab = screenDetail, "ACME-2705", tabDiff

	after, cmd := advance(t, m, press("r"))
	if after.hideDiffRationale {
		t.Error("r on the diff tab hid the reasons")
	}

	wantControl(t, cmd, got, "ACME-2705", "resume")

	after, _ = advance(t, m, press("H"))
	if !after.hideDiffRationale {
		t.Error("H on the diff tab left the reasons shown")
	}

	// And n starts a run here as it does on the board, with the next change
	// of the diff on }.
	// This run is parked, so the dialog may refuse; either way it answers.
	if after, _ = advance(t, m, press("n")); after.screen != screenStart && after.message == "" {
		t.Errorf("n on the diff tab left screen %v and said nothing, want the start dialog's answer",
			after.screen)
	}
}
