package cells

// The marks the window draws in the margins: the cursor's arrow, the bar
// down the side of a list that is longer than its box, and the padding that
// squares a column off.
//
// They are here rather than beside what draws them because every screen
// draws them, and a screen that is its own package still has to be able to
// say "this list has more in it".

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/theme"
)

// Thumb is the run of the scroll bar that shows where the eye is; the rest
// of the track is drawn with the box's own edge.
const Thumb = "┃"

// Rail is the rest of that bar: the part with nothing on it.
const Rail = "│"

// Gutter is the three cells to the left of every row, where the cursor's
// mark is drawn. It is subtracted before a row's columns are planned,
// because the plan is about the row and the mark is not part of one.
const Gutter = 3

// Pad cuts one field to its budget and fills what is left, measuring in
// cells throughout: an accented word is one column narrower than its length
// in bytes, and a column planned in bytes looks crooked exactly where
// nobody tests.
func Pad(text string, cells int, right bool) string {
	if cells <= 0 {
		return ""
	}

	text = ansi.Truncate(text, cells, "…")

	space := strings.Repeat(" ", max(cells-lipgloss.Width(text), 0))
	if right {
		return space + text
	}

	return text + space
}

// Mark is what the cursor's row is marked with, in the gutter.
//
// It is a chevron rather than the triangle a fold is drawn with. The two sit
// side by side on any screen whose rows fold — the gutter mark, then the
// section's own arrow — and drawn with the same glyph they read as one
// stutter rather than as two different facts about the row.
const Mark = "❯"

// Track is the bar down the right edge of a pane holding more than it
// can show. It answers what the line at the foot of the pane cannot: where
// in the text the reader is, and how much of it is left.
//
// It returns nil when everything fits, so a pane that does not scroll does
// not grow a rail that never moves.
func Track(rows, total, offset int) []string {
	if rows <= 0 || total <= rows {
		return nil
	}

	thumb := max(1, rows*rows/total)
	top := min(max(offset, 0)*rows/total, rows-thumb)

	// The last scroll position ends on the last row. Dividing down leaves
	// the thumb a cell short of the floor, which reads as more to come on a
	// pane that has nothing left.
	if offset >= total-rows {
		top = rows - thumb
	}

	col := make([]string, rows)

	for i := range col {
		if i < top || i >= top+thumb {
			col[i] = theme.Text(theme.Tertiary).Render(Rail)
			continue
		}

		col[i] = theme.Text(theme.Secondary).Render(Thumb)
	}

	return col
}

// Dot separates two facts on one line. It is a middle dot and not a comma
// or a pipe: a comma reads as a list of one thing and a pipe as a column
// rule, and this is two things a reader takes in at once.
const Dot = " · "

// The arrows a section is opened and closed with. They are the whole of the
// affordance: a head with no mark beside it is read as a label, and a reader
// who cannot see that a block folds never folds one.
const (
	FoldOpen = "▾ "
	FoldShut = "▸ "
)

// Fold is the arrow for a section in that state.
func Fold(open bool) string {
	if open {
		return FoldOpen
	}

	return FoldShut
}

// another
