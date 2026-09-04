package ui

// The word the gate has always understood, and what it takes to write it.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestSkipIsAskedAboutBeforeItIsWritten.
//
// A phase that did not run cannot be un-run, so s asks the way x and b do.
// The word reaches the port only after the reader says yes.
func TestSkipIsAskedAboutBeforeItIsWritten(t *testing.T) {
	m, got := parkedModel(t)
	m = onRow(t, m, "ACME-2705")

	asked, cmd := advance(t, m, press("s"))
	if cmd != nil || got.word != "" {
		t.Fatalf("s wrote %q before the reader answered", got.word)
	}

	if asked.confirm != confirmSkip || asked.confirmID != "ACME-2705" {
		t.Fatalf("confirm = %v on %q, want the question about the task under the cursor", asked.confirm, asked.confirmID)
	}

	if band := ansi.Strip(asked.bandLine(100)); !strings.Contains(band, "skip the phase") {
		t.Errorf("the band asks %q, want the question the key raised", band)
	}

	_, cmd = advance(t, asked, press("y"))
	wantControl(t, cmd, got, "ACME-2705", "skip")
}

// TestSkipIsDroppedWhenTheAnswerIsNotYes: anything that is not the
// confirming key leaves the run waiting where it was.
func TestSkipIsDroppedWhenTheAnswerIsNotYes(t *testing.T) {
	m, got := parkedModel(t)
	m = onRow(t, m, "ACME-2705")

	asked, _ := advance(t, m, press("s"))

	after, cmd := advance(t, asked, press("n"))
	if cmd != nil || got.word != "" {
		t.Errorf("a no wrote %q", got.word)
	}

	if after.confirm != confirmNone {
		t.Error("the question is still up after it was answered")
	}
}

// TestSkipIsAnsweredOnTheTaskViewToo: it is offered from the same screen the
// needs-you banner is drawn on.
func TestSkipIsAnsweredOnTheTaskViewToo(t *testing.T) {
	m, _ := parkedModel(t)
	m.screen, m.detail = screenDetail, "ACME-2705"

	asked, _ := advance(t, m, press("s"))
	if asked.confirm != confirmSkip || asked.confirmID != "ACME-2705" {
		t.Fatalf("confirm = %v on %q, want the question about the task the view is on", asked.confirm, asked.confirmID)
	}
}

// TestSkipSaysWhyOnATaskThatIsNotWaiting: the refusal is said, as every
// other refused verb's is.
func TestSkipSaysWhyOnATaskThatIsNotWaiting(t *testing.T) {
	m, got := testModel(t, 100, 30)
	m = onRow(t, m, "ACME-2706")

	after, cmd := advance(t, m, press("s"))
	if cmd != nil || got.word != "" {
		t.Fatalf("s wrote %q on a phase that is under way", got.word)
	}

	if after.confirm != confirmNone {
		t.Fatal("s on a phase that is under way asked the question anyway")
	}

	if after.message == "" {
		t.Error("s on a phase that is under way said nothing")
	}
}
