package panes

// The rule that separates one attempt from the next, drawn at every width.
//
// It is a horizontal line with a word at each end, and its length is what
// is left of the window after those words. A sum that is one out leaves a
// line a cell short of the edge, or one cell past it — and past it wraps,
// which puts every row of the attempt below where the reader expects it.

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/view"
)

// TestTheSeamBetweenTwoAttemptsFillsTheWidth.
func TestTheSeamBetweenTwoAttemptsFillsTheWidth(t *testing.T) {
	e := world(t, nil)

	// The narrowest body a window can have: layout refuses to draw below
	// MinWidth, so anything under this is a size the timeline never sees.
	narrowest, err := layout.Fit(layout.MinWidth, 30)
	if err != nil {
		t.Fatalf("the narrowest window layout allows will not fit: %v", err)
	}

	for _, entry := range []view.Entry{
		{Attempt: 1, At: now},
		{Attempt: 2, At: ago(90 * time.Minute)},
		{Attempt: 12, At: time.Time{}},
		{Attempt: 0, At: now},
	} {
		// The rule reaches the edge and stops there. A seam a cell short
		// reads as a box that did not close; one a cell past wraps, which
		// puts every row of the attempt below where the reader expects it.
		for w := narrowest.Body.W; w <= 160; w++ {
			if drawn := lipgloss.Width(e.seam(entry, w)); drawn != w {
				t.Errorf("attempt %d at %d columns drew %d cells, want the whole width",
					entry.Attempt, w, drawn)
			}
		}
	}
}

// TestTheSeamSaysWhichAttemptItOpens, which is the one thing a reader
// comparing two runs of the same task asks first — and the rule either side
// of it is furniture.
func TestTheSeamSaysWhichAttemptItOpens(t *testing.T) {
	e := world(t, nil)

	got := e.seam(view.Entry{Attempt: 3, At: ago(2 * time.Hour)}, 100)
	if !strings.Contains(got, "3") {
		t.Errorf("the seam of attempt 3 does not name it: %q", got)
	}

	// A record with no clock on the entry says nothing about when rather
	// than naming a time nothing measured.
	bare := e.seam(view.Entry{Attempt: 3}, 100)
	if lipgloss.Width(bare) != 100 {
		t.Errorf("a seam with no time on it drew %d cells", lipgloss.Width(bare))
	}
}
