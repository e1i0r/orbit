package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// The shapes a pane is built out of.
//
// tokens.go says what paper and what ink exist; this says what they are
// assembled into — a card, a grid of fields, a badge. A pane asks for a part
// and gets the same one every other pane gets, which is the difference
// between a window with a look and eleven tabs that each invented one.
//
// Every part returns lines rather than one string, because a pane is a list
// of lines that the frame scrolls, and a part that returned a block would
// have to be taken apart again by every caller.

// cardChrome is what a card spends on being a card: two columns of border
// and two of padding. lipgloss counts both inside the width it is given, so a
// card is exactly as wide as it was asked to be and its body gets this much
// less — which is what keeps a line from wrapping around its own border.
const cardChrome = 4

// cardFloor is the narrowest a card is drawn. Below it the chrome is most of
// the card, so a strip that cannot afford this asks for something else.
const cardFloor = 12

// card lays a block on raised paper inside a rounded border, with a quiet
// uppercase title on the first line the way a detail page names the block
// before stating it.
//
// A title is optional: the strip of figures at the top of a pane is four
// cards whose titles are the figures' own labels, and a card around a code
// listing has nothing to add above it.
func card(title string, body []string, width int) []string {
	outer := max(cardFloor, width)
	inner := outer - cardChrome

	box := Surface(Raised).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Rule()).
		Padding(0, 1).
		Width(outer)

	lines := make([]string, 0, len(body)+1)
	if title != "" {
		lines = append(lines, Text(Tertiary).Render(fit(strings.ToUpper(title), inner)))
	}

	for _, l := range body {
		lines = append(lines, fit(l, inner))
	}

	return strings.Split(box.Render(strings.Join(lines, "\n")), "\n")
}

// field is one thing a card states about its subject: what it is called, and
// what it says.
type field struct {
	label string
	value string
}

// fields sets label-over-value pairs in columns: the label small, quiet and
// upper-cased above, the value at full contrast under it, and a blank line
// between rows.
//
// Two lines per pair rather than a label column and a value column, because
// a terminal has one type size — a label beside its value can only be told
// from it by colour, and above it, it is told by position as well.
func fields(pairs []field, columns, width int) []string {
	if columns < 1 || len(pairs) == 0 {
		return nil
	}

	cell := max(10, width/columns)

	var out []string

	for row := 0; row < len(pairs); row += columns {
		labels, values := "", ""

		for col := 0; col < columns && row+col < len(pairs); col++ {
			p := pairs[row+col]
			labels += pad(Text(Tertiary).Render(strings.ToUpper(p.label)), cell, false)
			values += pad(p.value, cell, false)
		}

		if row > 0 {
			out = append(out, "")
		}

		out = append(out, strings.TrimRight(labels, " "), strings.TrimRight(values, " "))
	}

	return out
}

// badge is a soft pill: the role's own hue, on paper tinted with it. It is
// what the saturated blocks become — legible at a glance without being the
// loudest thing on a pane that has ten other things to say.
func badge(text string, r Role) string {
	return Tint(r).Render(text)
}

// tabGap is the space between two chips in a strip. Two cells rather than
// one: the chips carry no brackets, so the gap is the only thing saying
// where one tab ends.
const tabGap = "  "

// tabChip is one tab of the strip: its key, then its name, and an underline
// under both when it is the tab being read.
//
// It returns what it drew and the plain text of it, because the strip is
// clickable and a hit test on rendered text would be counting escape codes.
func tabChip(key, text string, active bool) (plain, rendered string) {
	// A tab too narrow to be named keeps the brackets the named ones drop:
	// a bare digit in a row of digits does not say it is a key to press.
	if text == "" {
		return "[" + key + "]", Text(Tertiary).Render("[") +
			Paint(Accent).Bold(true).Render(key) + Text(Tertiary).Render("]")
	}

	plain = key + " " + text

	if active {
		return plain, Paint(Accent).Bold(true).Underline(true).Render(key) +
			Text(Primary).Bold(true).Underline(true).Render(" "+text)
	}

	return plain, Paint(Accent).Bold(true).Render(key) + Text(Tertiary).Render(" "+text)
}

// The two cells a scroll bar is drawn with: a rail the height of the pane
// and a thumb over the part of it the reader is looking at.
const (
	scrollRail  = "│"
	scrollThumb = "┃"
)

// scrollTrack is the bar down the right edge of a pane holding more than it
// can show. It answers what the line at the foot of the pane cannot: where
// in the text the reader is, and how much of it is left.
//
// It returns nil when everything fits, so a pane that does not scroll does
// not grow a rail that never moves.
func scrollTrack(rows, total, offset int) []string {
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
			col[i] = Text(Tertiary).Render(scrollRail)
			continue
		}

		col[i] = Text(Secondary).Render(scrollThumb)
	}

	return col
}
