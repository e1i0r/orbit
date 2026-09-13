package panes

// The timeline pane: every entry in the order it happened, with seams
// between attempts and headings where a row folds.
//
// Attempts are what the eye follows: a run that went round twice draws
// the seam between its tries, and a row that folds carries its heading
// with it.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// TestTimelineDrawsSeamsBetweenAttempts. Every attempt opens with a
// seam naming it, so a run that went round twice reads as two tries.
func TestTimelineDrawsSeamsBetweenAttempts(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "task.started", Attempt: 1},
		{Kind: "phase.finished", Phase: "implement", Attempt: 1},
		{Kind: "task.started", Attempt: 2},
		{Kind: "phase.failed", Phase: "implement", Attempt: 2},
	})

	drawn := Timeline(e)

	if len(drawn.Rows) == 0 {
		t.Fatal("the timeline drew nothing")
	}

	if len(drawn.Seams) != 2 || drawn.Seams[0] != 1 || drawn.Seams[3] != 2 {
		t.Errorf("the timeline seamed %v, want one seam per attempt", drawn.Seams)
	}
}

// TestTimelineIsEmptyWhenNothingHappened. No entries is a sentence
// saying so, not a blank pane.
func TestTimelineIsEmptyWhenNothingHappened(t *testing.T) {
	drawn := Timeline(world(t, nil))

	if !strings.Contains(strings.Join(drawn.Rows, "\n"), "nothing has been recorded") {
		t.Errorf("an empty timeline reads:\n%s", strings.Join(drawn.Rows, "\n"))
	}
}

// TestTimelineSaysWhenReadingBroke. A failure reads as itself, in front
// of everything the record might have said.
func TestTimelineSaysWhenReadingBroke(t *testing.T) {
	e := world(t, nil)
	e.Failed = "the record would not open"

	drawn := Timeline(e)

	if !strings.Contains(strings.Join(drawn.Rows, "\n"), "would not open") {
		t.Errorf("a broken timeline reads:\n%s", strings.Join(drawn.Rows, "\n"))
	}
}
