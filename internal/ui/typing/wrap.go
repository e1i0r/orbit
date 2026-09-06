package typing

// Where a value breaks when it is drawn in a box, and where a position in
// what was drawn is in the value.
//
// The window's own splitIntoLines answers the first half, and is what every
// other screen wraps with, but it rejoins words with a single space and drops the rest,
// so nothing that comes out of it can be counted back to an offset. A caret
// and a click both need that count: the reader points at a cell of a box
// and means a place in the text.

// Span is one drawn line of a value: where it starts and ends in the runes
// of that value, the end being one past the last rune on the line.
type Span struct {
	From int
	To   int
}

// Wrap breaks a value into the lines it is drawn as, keeping every
// rune: the lines of one value are back to back, so the offset of any cell
// on any of them is the line's start plus the column.
//
// A break takes the space with the line it ends, which is what puts the
// caret at the head of the next line when somebody types past the edge. A
// word longer than the box is broken where the box ends, because the
// alternative is a line that overflows it.
func Wrap(rs []rune, w int) []Span {
	var out []Span

	line := 0

	for at := 0; at <= len(rs); at++ {
		if at < len(rs) && rs[at] != '\n' {
			continue
		}

		out = append(out, wrapOne(rs, line, at, w)...)
		line = at + 1
	}

	return out
}

// wrapOne breaks one line of the value, between two newlines.
func wrapOne(rs []rune, from, to, w int) []Span {
	if w <= 0 || to-from <= w {
		return []Span{{From: from, To: to}}
	}

	var out []Span

	for from < to {
		if to-from <= w {
			out = append(out, Span{From: from, To: to})

			break
		}

		brk := from + w
		for i := from + w; i > from; i-- {
			if rs[i-1] == ' ' {
				brk = i

				break
			}
		}

		out = append(out, Span{From: from, To: brk})
		from = brk
	}

	return out
}

// SpanRow is the line an offset is drawn on.
//
// An offset that is both the end of one line and the start of the next is
// on the next one: a caret that has just been carried over the edge belongs
// where the next character it types will go.
func SpanRow(spans []Span, at int) int {
	row := 0

	for i, s := range spans {
		if at >= s.From {
			row = i
		}
	}

	return row
}

// SpanOffset is the place in the value a row and a column of the box point
// at. A column past the end of a line is that line's end, so a click in the
// Empty half of a line lands after its last character rather than nowhere.
func SpanOffset(spans []Span, row, col int) int {
	if len(spans) == 0 {
		return 0
	}

	s := spans[clamp(row, 0, len(spans)-1)]

	return clamp(s.From+col, s.From, s.To)
}

// SpanText is one drawn line of the value.
func SpanText(rs []rune, s Span) string {
	return string(rs[clamp(s.From, 0, len(rs)):clamp(s.To, 0, len(rs))])
}

// SpanWindow is which lines of a box are drawn when there are more of them
// than it is tall: the last ones, unless the caret is above them, in which
// case the ones the caret is on the bottom of. It answers the first line
// drawn.
func SpanWindow(lines, height, caretRow int) int {
	if lines <= height {
		return 0
	}

	top := lines - height
	if caretRow < top {
		top = caretRow
	}

	return top
}
