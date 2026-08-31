package ui

import "strings"

// How one drawn line of a field is painted when its cells do not all mean
// the same thing: a stretch of it is selected, one cell of it is where the
// caret stands, and the rest is drawn the way the field draws itself.
//
// The painting is split from the drawing because the two fields that use it
// paint differently — the box paints nothing, a one-line field paints its
// value as the accent — and neither of them should have to know how a
// selection looks.

// What one cell of a line is.
const (
	cellPlain = iota
	cellSel
	cellCaret
)

// paintCells draws one line, given the stretch of it that is selected and
// the column the caret is on, both in the line's own columns. A caret of -1
// is a line the caret is not on; a caret one past the last character is a
// block drawn after the text, which is where a reader typing at the end of
// a line is standing.
//
// Cells of a kind are painted in one go rather than one at a time, so a
// line comes out carrying a handful of escapes rather than one per
// character.
func paintCells(line string, from, to, caret int, paint func(string) string) string {
	rs := []rune(line)
	if caret == len(rs) {
		rs = append(rs, ' ')
	}

	kind := func(at int) int {
		switch {
		case at == caret:
			return cellCaret
		case at >= from && at < to:
			return cellSel
		}

		return cellPlain
	}

	var b strings.Builder

	run := 0

	for at := 1; at <= len(rs); at++ {
		if at < len(rs) && kind(at) == kind(run) {
			continue
		}

		b.WriteString(paintRun(kind(run), string(rs[run:at]), paint))
		run = at
	}

	return b.String()
}

// paintRun paints one stretch of a line. The caret is the selection's own
// colours made bold, so that a caret sitting at the edge of a selection is
// still something the eye can find.
func paintRun(kind int, s string, paint func(string) string) string {
	switch kind {
	case cellCaret:
		return Paint(Sel).Bold(true).Render(s)
	case cellSel:
		return Paint(Sel).Render(s)
	}

	return paint(s)
}
