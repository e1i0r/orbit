package known

// Choosing one of a row's options from a list, the way the diff's file
// selector already does it.
//
// A row of pills is right for two or three answers and wrong for fourteen. A
// project's Makefile has as many targets as it has, and laid out side by side
// they wrapped to four lines that pushed the rest of the form off the screen
// — so the many are behind one keystroke, in a box with a cursor and a rail,
// and the row itself shows only what it is holding.

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// pickRows is how many of the list are on screen at once, and pickFrom how
// many options a row needs before its answers are put behind one.
//
// Two. Every row that offers a choice at all offers it the same way, and
// that is the whole of why: the ones with two used to lay them out side by
// side, and a lit word beside a grey word does not read as a choice — it
// reads as a value with something after it, and the reader has to work out
// which. A row that always says "▾ 2 to choose from" needs working out once.
const (
	pickRows = 7
	pickFrom = 2
)

// picked is the row whose list is open, and nothing when none is.
func (s State) picked(e Env) (aRow, bool) {
	if !s.picking {
		return aRow{}, false
	}

	for _, r := range s.rows2(e) {
		if r.which == s.field {
			return r, true
		}
	}

	return aRow{}, false
}

// openPicker puts the list up with the cursor on what the row is holding.
func (s State) openPicker(r aRow) State {
	s.picking, s.pick = true, 0

	for i, o := range r.options {
		if o.value == s.held(r) {
			s.pick = i
		}
	}

	return s
}

// pickKey is every key while the list is open. It is a list with one row
// chosen and nothing else, so it answers what every such list in this window
// answers and nothing more.
func (s State) pickKey(code rune, r aRow) (State, bool) {
	switch code {
	case 'k':
		s.pick = max(s.pick-1, 0)
	case 'j':
		s.pick = min(s.pick+1, len(r.options)-1)
	default:
		return s, false
	}

	return s, true
}

// takePick puts what the cursor is on into the row, and shuts the list.
func (s State) takePick(r aRow) State {
	if s.pick >= 0 && s.pick < len(r.options) {
		s = s.walkTo(r, r.options[s.pick])
	}

	s.picking = false

	return s
}

// pickBox is the list, drawn where the form's rows would be.
func (s State) pickBox(r aRow, cw int, e Env) []string {
	inner := max(cw-2, 20)
	line := theme.Paint(theme.Dim)

	out := []string{edge("┌", "┐", theme.Paint(theme.Accent).Bold(true).Render(r.label), cw)}

	top := 0
	if s.pick >= pickRows {
		top = s.pick - pickRows + 1
	}

	rail := cells.Track(pickRows, len(r.options), top)

	// The names in a column of their own, so the reasons beside them line
	// up and the eye can run down either one.
	wide := 0
	for _, o := range r.options {
		wide = max(wide, lipgloss.Width(o.label))
	}

	for i := top; i < len(r.options) && i < top+pickRows; i++ {
		one := r.options[i]

		mark, ink := "   ", theme.Text(theme.Primary)
		if i == s.pick {
			mark, ink = theme.Paint(theme.Live).Bold(true).Render(" ▸ "), theme.Paint(theme.Live).Bold(true)
		}

		held := "  "
		if one.value == s.held(r) {
			held = theme.Paint(theme.OK).Render("● ")
		}

		named := ink.Render(cells.Pad(one.label, wide, false))
		if one.note != "" {
			named += "  " + theme.Paint(theme.Dim).Render(one.note)
		}

		text := ansi.Truncate(mark+held+named, inner, "…")

		right := line.Render("│")
		if rail != nil {
			right = rail[i-top]
		}

		out = append(out, line.Render("│")+cells.PadRight(text, inner)+right)
	}

	return append(out, edge("└", "┘", theme.Paint(theme.Dim).Render(e.Words.T("knowledge.pick_ways",
		"[↑↓] move · [↵] choose it · [esc] leave it")), cw))
}

// pickAt is which option of the open list is on that line of the body, and
// -1 for a cell that is the box's own chrome or outside it.
func (s State) pickAt(line int, r aRow, cw int, e Env) int {
	// The head above the box, then its top border.
	first := len(s.formAbove(cw, e)) + 1

	at := line - first
	if at < 0 || at >= pickRows {
		return -1
	}

	top := 0
	if s.pick >= pickRows {
		top = s.pick - pickRows + 1
	}

	if top+at >= len(r.options) {
		return -1
	}

	return top + at
}

// pickHint is what a closed row shows at the end of its value: that there is
// a list behind it, and how long it is.
func (s State) pickHint(r aRow, e Env) string {
	if len(r.options) < pickFrom {
		return ""
	}

	return theme.Paint(theme.Dim).Render(e.Words.T("knowledge.pick_open",
		"▾ {n} to choose from", about("n", itoa(len(r.options)))))
}

// itoa is a count in the one place this package needs one inline.
func itoa(n int) string {
	var out []string

	for n > 9 {
		out = append([]string{string(rune('0' + n%10))}, out...)
		n /= 10
	}

	return strings.Join(append([]string{string(rune('0' + n))}, out...), "")
}

// edge is a border with a word set into it: "┌─ what this is ─────┐". The
// title goes on the top and the keys on the bottom, so neither costs a row
// of a terminal that has none to spare, and the box stays a rectangle.
func edge(left, right, set string, cw int) string {
	line := theme.Paint(theme.Dim)
	if set == "" {
		return line.Render(left + strings.Repeat("─", max(cw-2, 0)) + right)
	}

	set = ansi.Truncate(set, max(cw-8, 4), "…")
	rule := max(cw-5-lipgloss.Width(set), 0)

	return line.Render(left+"─ ") + set + line.Render(" "+strings.Repeat("─", rule)+right)
}
