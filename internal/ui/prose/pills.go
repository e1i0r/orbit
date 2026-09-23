package prose

// A row of choices, every one of them visible.
//
// The compose form has drawn its flows this way since it was written: all
// of them on one line, the chosen one marked, each of them a thing you can
// click. The start dialog drew only the one it was on and cycled to the
// next on a keypress, so choosing `gated` out of eight meant pressing f
// until it appeared — with no way to see how many were left, or that you
// had just gone past it.
//
// So the row lives here rather than inside either screen. Two drawings of
// one row is two rows that drift, and the second one was already a worse
// version of the first.

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/theme"
)

// Choice is one option on the row.
type Choice struct {
	// Label is the word a reader picks by.
	Label string
	// Glyph goes in front of it, and is empty for a row that has none.
	Glyph string
}

// Placed is where one choice was drawn, counted in cells from the left
// edge of the row.
//
// It is answered by the same function that draws, because a zone worked
// out a second time from the same widths is a zone that disagrees with the
// drawing the first time somebody changes a glyph.
type Placed struct {
	At    int
	Cells int
	// Line is which row of a wrapped Row this one landed on, counted from
	// zero. A hit test that ignored it would answer the choice sitting at
	// the same column two lines down.
	Line int
}

// pillGap is what the columns are separated by, and what Row steps over.
const pillGap = "  "

// Row lays the choices out as a grid: every cell the same width, every
// column lined up down the block, the one at `chosen` marked.
//
// from is the column the grid starts at, so a caller that drew a label
// first can hand over where it stopped and get zones measured against the
// terminal rather than against the row. within is the width there is.
//
// A grid and not a line of pills. Laid out as they came, the second row
// began wherever the first happened to end and the words under each other
// were half a pill apart — eight flows read as two ragged lines rather
// than as a list of eight things. Elio asked for a table, and a table is
// what makes a column of names scannable.
//
// Every choice is drawn. The whole reason this exists is that the start
// dialog showed one out of eight, and a grid that cuts the last three off
// at a hundred columns is the same failure with a tidier first line.
func Row(choices []Choice, chosen, from, within int) ([]string, []Placed) {
	if len(choices) == 0 {
		return []string{""}, nil
	}

	cell := cellWidth(choices)
	gap := lipgloss.Width(pillGap)

	// How many fit across.
	//
	// A width of nought is a caller that does not want the grid wrapped at
	// all — the compose form, whose caret arithmetic is per line and whose
	// box below is placed from a fixed plan. Never fewer than one: a
	// terminal too narrow for a single cell gets one per line and
	// overruns, which is visible, rather than no lines at all.
	across := len(choices)

	if within > from {
		across = 1
		if n := (within - from + gap) / (cell + gap); n > 1 {
			across = n
		}
	}

	var (
		lines  []string
		drawn  []string
		at     = make([]Placed, 0, len(choices))
		indent = strings.Repeat(" ", from)
	)

	for i, c := range choices {
		col := len(drawn)
		if col == across {
			lines = append(lines, strings.Join(drawn, pillGap))
			drawn, col = nil, 0
		}

		drawn = append(drawn, pill(c, i == chosen, cell))
		at = append(at, Placed{
			At:    from + col*(cell+gap),
			Cells: cell,
			Line:  len(lines),
		})
	}

	lines = append(lines, strings.Join(drawn, pillGap))
	for i := 1; i < len(lines); i++ {
		lines[i] = indent + lines[i]
	}

	return lines, at
}

// cellWidth is how wide a rendered cell is: the widest of them, drawn.
//
// Drawn and measured rather than counted up from the words, twice over.
// The glyphs are emoji and an emoji is not one cell, and theme.Pill puts
// its own space either side of whatever it is given — so a number added up
// from the labels was two cells short of what the terminal showed, and the
// grid ran past the edge it had just been fitted to.
func cellWidth(choices []Choice) int {
	wide := 0

	for i, c := range choices {
		if w := lipgloss.Width(pill(c, i == 0, 0)); w > wide {
			wide = w
		}
	}

	return wide
}

// plain is one choice's glyph and word, with nothing around them.
func plain(c Choice) string {
	if c.Glyph == "" {
		return c.Label
	}

	return c.Glyph + " " + c.Label
}

// pill is one choice, lit when it is the one in use.
func pill(c Choice, chosen bool, cell int) string {
	// The marker's room is kept on every cell, so that a name does not
	// shift two columns left the moment it stops being the one in use.
	marker := "  "
	if chosen {
		marker = "● "
	}

	drawn := paper(marker+plain(c), chosen)

	// Widened to the cell by measuring what came back, because theme.Pill
	// adds space of its own and the glyphs are emoji: both are things this
	// cannot count and both are things it can measure.
	if pad := cell - lipgloss.Width(drawn); pad > 0 {
		drawn = paper(marker+plain(c)+strings.Repeat(" ", pad), chosen)
	}

	return drawn
}

// paper is one cell's words on the colour that says whether it is the one
// in use.
func paper(text string, chosen bool) string {
	if chosen {
		return theme.Pill(text, theme.PillInkLit, theme.PillChosen)
	}

	return theme.Pill(text, theme.PillInkRest, theme.PillRest)
}

// At is which choice a cell is over, and whether it is over one at all.
//
// line is which row of the wrapped row the pointer is on, counted from the
// first. Both are needed: the same column on two lines is two choices.
func At(placed []Placed, x, line int) (int, bool) {
	for i, p := range placed {
		if p.Line == line && x >= p.At && x < p.At+p.Cells {
			return i, true
		}
	}

	return 0, false
}

// FlowGlyph is the mark a flow is recognised by at a glance.
//
// Here rather than in either screen for the reason the row is: both draw
// the same flows, and a shield on one screen and a bolt on the other is
// two flows as far as a reader is concerned.
func FlowGlyph(name string) string {
	switch name {
	case "quick":
		return "🚀"
	case "careful":
		return "🛡️"
	}

	return "⚡"
}
