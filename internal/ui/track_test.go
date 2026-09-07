package ui

// The scroll bar down the side of a pane: where the thumb sits, and how much
// of the rail it covers.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
)

// thumbRun is where the thumb sits on a track and how much of it it covers.
func thumbRun(col []string) (first, last, rows int) {
	first, last = -1, -1

	for i, c := range col {
		if !strings.Contains(c, cells.Thumb) {
			continue
		}

		if first < 0 {
			first = i
		}

		last, rows = i, rows+1
	}

	return first, last, rows
}

// TestAPaneThatFitsHasNoBar. A rail that cannot move says there is somewhere
// to go, and spends a column of every pane saying it.
func TestAPaneThatFitsHasNoBar(t *testing.T) {
	for _, c := range []struct{ rows, total int }{{10, 10}, {10, 3}, {0, 40}} {
		if got := cells.Track(c.rows, c.total, 0); got != nil {
			t.Errorf("cells.Track(%d rows, %d lines) drew %d rows of bar", c.rows, c.total, len(got))
		}
	}
}

// TestTheBarIsOneCellWide. The rail is drawn in the pane's last column over
// a line filled to the column before it, so a two-cell rail wraps every row
// of the pane onto a second line.
func TestTheBarIsOneCellWide(t *testing.T) {
	for _, c := range cells.Track(12, 90, 30) {
		if got := lipgloss.Width(c); got != 1 {
			t.Errorf("a row of the bar is %d cells wide, want 1", got)
		}
	}
}

// TestTheThumbSaysWhereTheReaderIs. Top of the text, top of the rail; end of
// the text, floor of the rail. Dividing down leaves the thumb a cell short
// of the floor on the last screen, which reads as more to come.
func TestTheThumbSaysWhereTheReaderIs(t *testing.T) {
	const rows, total = 10, 90

	first, _, _ := thumbRun(cells.Track(rows, total, 0))
	if first != 0 {
		t.Errorf("at the top of the text the thumb starts on row %d, want 0", first)
	}

	_, last, _ := thumbRun(cells.Track(rows, total, total-rows))
	if last != rows-1 {
		t.Errorf("at the end of the text the thumb ends on row %d, want %d", last, rows-1)
	}
}

// TestTheThumbIsTheShareThatShows. How tall the thumb is, is how much of the
// text is on the screen: a tenth of it is a tenth of the rail, and never
// nothing at all.
func TestTheThumbIsTheShareThatShows(t *testing.T) {
	for _, c := range []struct{ rows, total, want int }{
		{10, 20, 5},
		{10, 100, 1},
		{10, 40000, 1},
		{20, 30, 13},
	} {
		if _, _, got := thumbRun(cells.Track(c.rows, c.total, 0)); got != c.want {
			t.Errorf("%d rows over %d lines: thumb is %d rows, want %d", c.rows, c.total, got, c.want)
		}
	}
}
