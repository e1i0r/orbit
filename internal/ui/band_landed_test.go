package ui

// Did it finish? The band says so, for as long as it is worth saying.

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// TestTheBandSaysAVerbCameBack. The band said who was working and then went
// quiet, and quiet is the same picture as a key that did nothing: a reader
// could not tell a pull request that had been opened from one that never
// was.
func TestTheBandSaysAVerbCameBack(t *testing.T) {
	m, _ := deliverWindow(t)
	m.screen = screenDetail
	m.entries = []view.Entry{
		{Kind: "deliver.asked", Verb: "CREATE PR", By: "supervisor", At: ago(2 * time.Minute)},
		{
			Kind: "deliver.answered",
			Verb: "CREATE PR",
			Text: "opened pull request 12\nagainst main",
			At:   ago(30 * time.Second),
		},
	}

	got := ansi.Strip(m.bandLeft())
	for _, want := range []string{"CREATE PR", "came back", "opened pull request 12", "30s ago"} {
		if !strings.Contains(got, want) {
			t.Errorf("bandLeft = %q, want %q in it", got, want)
		}
	}

	// One line of the answer, not the whole of it: the band is one row.
	if strings.Contains(got, "against main") {
		t.Errorf("bandLeft = %q, want only the first line of the answer", got)
	}
}

// TestAVerbThatBrokeSaysSo, because "came back" over a failure is the same
// lie in the other direction.
func TestAVerbThatBrokeSaysSo(t *testing.T) {
	m, _ := deliverWindow(t)
	m.screen = screenDetail
	m.entries = []view.Entry{
		{Kind: "deliver.asked", Verb: "CREATE PR", By: "supervisor", At: ago(2 * time.Minute)},
		{
			Kind:  "deliver.answered",
			Verb:  "CREATE PR",
			Cause: "no remote called origin",
			At:    ago(20 * time.Second),
		},
	}

	got := ansi.Strip(m.bandLeft())
	for _, want := range []string{"CREATE PR", "broken", "no remote called origin"} {
		if !strings.Contains(got, want) {
			t.Errorf("bandLeft = %q, want %q in it", got, want)
		}
	}
}

// TestAnOldAnswerIsHistory. A verb answered an hour ago is on the tree, not
// on the band: the band is for the reader who is still looking at the
// screen, and a line that never goes away is a line nobody reads.
func TestAnOldAnswerIsHistory(t *testing.T) {
	m, _ := deliverWindow(t)
	m.screen = screenDetail
	m.entries = []view.Entry{
		{Kind: "deliver.asked", Verb: "CREATE PR", By: "supervisor", At: ago(2 * time.Hour)},
		{
			Kind: "deliver.answered",
			Verb: "CREATE PR",
			Text: "opened pull request 12",
			At:   ago(time.Hour),
		},
	}

	if got := ansi.Strip(m.bandLeft()); strings.Contains(got, "came back") {
		t.Errorf("bandLeft = %q, want an hour-old answer left to the tree", got)
	}
}
