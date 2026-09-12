package panes

// The history pane: the conversation out of the record, in the order it
// happened.
//
// A person's words and each engine's, whichever program they were typed
// in — the same reading the file behind `orbit history` makes, so a
// reader looking at this pane and an engine handed that file are looking
// at one conversation.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// TestHistorySaysWhoSaidWhat. Notes, directives and answers, each under
// the name of whoever said them.
func TestHistorySaysWhoSaidWhat(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "task.noted", Text: "cents, not floats", By: "operator"},
		{Kind: "task.dialogue", Text: "use redis", By: "operator"},
	})

	rows, _ := History(e)
	joined := strings.Join(rows, "\n")

	for _, want := range []string{"cents, not floats", "use redis"} {
		if !strings.Contains(joined, want) {
			t.Errorf("history does not carry %q:\n%s", want, joined)
		}
	}
}

// TestHistoryIsEmptyWhenNothingWasSaid. No turns is a sentence saying
// so, not a blank pane.
func TestHistoryIsEmptyWhenNothingWasSaid(t *testing.T) {
	rows, _ := History(world(t, nil))

	if !strings.Contains(strings.Join(rows, "\n"), "nothing has been said") {
		t.Errorf("an empty history reads:\n%s", strings.Join(rows, "\n"))
	}
}

// TestHistorySaysWhenReadingBroke. A failure reads as itself, in front
// of everything the record might have said.
func TestHistorySaysWhenReadingBroke(t *testing.T) {
	e := world(t, nil)
	e.Failed = "the record would not open"

	rows, _ := History(e)

	if !strings.Contains(strings.Join(rows, "\n"), "would not open") {
		t.Errorf("a broken history reads:\n%s", strings.Join(rows, "\n"))
	}
}
