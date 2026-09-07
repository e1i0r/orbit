package cells

// The marks the window draws in the margins, and the two measurements every
// screen makes of a row: how wide it is, and where the two halves of it go.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// TestAFieldIsPaddedInCellsAndNotInBytes. An accented word is one column
// narrower than its length in bytes, and a column planned in bytes looks
// crooked exactly where nobody tests.
func TestAFieldIsPaddedInCellsAndNotInBytes(t *testing.T) {
	for _, c := range []struct {
		what  string
		text  string
		cells int
		right bool
		want  string
	}{
		{"short of the budget", "ab", 5, false, "ab   "},
		{"against the right edge", "ab", 5, true, "   ab"},
		{"exactly the budget", "abcde", 5, false, "abcde"},
		{"an accented word", "café", 6, false, "café  "},
		{"no room at all", "abc", 0, false, ""},
	} {
		if got := Pad(c.text, c.cells, c.right); got != c.want {
			t.Errorf("%s: Pad(%q, %d, %v) = %q, want %q", c.what, c.text, c.cells, c.right, got, c.want)
		}
	}

	// Past the budget it is cut, and the ellipsis says so.
	if got := Pad("abcdefgh", 5, false); lipgloss.Width(got) != 5 || !strings.Contains(got, "…") {
		t.Errorf("Pad past the budget = %q, want five cells ending in an ellipsis", got)
	}
}

// TestASectionSaysWhetherItIsOpen. The arrow is the whole of the affordance:
// a head drawn without one is read as a label.
func TestASectionSaysWhetherItIsOpen(t *testing.T) {
	if Fold(true) != FoldOpen || Fold(false) != FoldShut {
		t.Errorf("Fold answers %q open and %q shut", Fold(true), Fold(false))
	}

	if FoldOpen == FoldShut {
		t.Error("the two states of a fold are drawn with the same mark")
	}
}

// TestThePaneWithNothingHiddenGrowsNoRail. A bar that never moves is a bar
// that says something untrue about a pane that fits.
func TestThePaneWithNothingHiddenGrowsNoRail(t *testing.T) {
	if got := Track(10, 10, 0); got != nil {
		t.Errorf("a pane holding exactly what it shows drew a bar of %d rows", len(got))
	}

	if got := Track(0, 100, 0); got != nil {
		t.Error("a pane of no rows drew a bar")
	}
}

// TestTheThumbSaysWhereInTheTextTheReaderIs, and reaches the floor on the
// last row: dividing down leaves it a cell short, which reads as more to
// come on a pane that has nothing left.
func TestTheThumbSaysWhereInTheTextTheReaderIs(t *testing.T) {
	const rows, total = 10, 100

	top := Track(rows, total, 0)
	if len(top) != rows {
		t.Fatalf("the bar is %d rows, want %d", len(top), rows)
	}

	if !strings.Contains(top[0], Thumb) {
		t.Errorf("at the top of the text the thumb is not on the first row: %q", ansi.Strip(strings.Join(top, "")))
	}

	end := Track(rows, total, total-rows)
	if !strings.Contains(end[rows-1], Thumb) {
		t.Errorf("at the end of the text the thumb is not on the last row: %q", ansi.Strip(strings.Join(end, "")))
	}

	// And in the middle it is neither: the bar is where the eye is.
	mid := Track(rows, total, total/2)
	if strings.Contains(mid[0], Thumb) || strings.Contains(mid[rows-1], Thumb) {
		t.Error("halfway through the text the thumb is at one of the ends")
	}
}

// TestOneThingLeftAndOneThingRight, and the right-hand one given up whole
// when they will not both fit: a hint truncated to "unread cap reac…" costs
// the reader the number, which was the only part of it worth the cells.
func TestOneThingLeftAndOneThingRight(t *testing.T) {
	if got := Spread("left", "right", 20); got != "left           right" {
		t.Errorf("Spread with room = %q", got)
	}

	if got := Spread("left", "", 10); got != Fit("left", 10) {
		t.Errorf("Spread with nothing on the right = %q, want the left alone", got)
	}

	tight := Spread("the left half", "the right half", 20)
	if strings.Contains(tight, "right") {
		t.Errorf("Spread with no room = %q, want the right-hand one given up", tight)
	}

	if lipgloss.Width(tight) > 20 {
		t.Errorf("Spread over the width drew %d cells", lipgloss.Width(tight))
	}
}
