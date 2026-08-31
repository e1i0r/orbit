package ui

// Where a value breaks when it is drawn in a box, and where a position in
// what was drawn is in the value.
//
// splitIntoLines answers the first half and is what every other screen
// wraps with, but it rejoins words with a single space and drops the rest,
// so nothing that comes out of it can be counted back to an offset. A caret
// and a click both need that count: the reader points at a cell of a box
// and means a place in the text.

// span is one drawn line of a value: where it starts and ends in the runes
// of that value, the end being one past the last rune on the line.
type span struct {
	from int
	to   int
}

// wrapSpans breaks a value into the lines it is drawn as, keeping every
// rune: the lines of one value are back to back, so the offset of any cell
// on any of them is the line's start plus the column.
//
// A break takes the space with the line it ends, which is what puts the
// caret at the head of the next line when somebody types past the edge. A
// word longer than the box is broken where the box ends, because the
// alternative is a line that overflows it.
func wrapSpans(rs []rune, w int) []span {
	var out []span

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
func wrapOne(rs []rune, from, to, w int) []span {
	if w <= 0 || to-from <= w {
		return []span{{from: from, to: to}}
	}

	var out []span

	for from < to {
		if to-from <= w {
			out = append(out, span{from: from, to: to})

			break
		}

		brk := from + w
		for i := from + w; i > from; i-- {
			if rs[i-1] == ' ' {
				brk = i

				break
			}
		}

		out = append(out, span{from: from, to: brk})
		from = brk
	}

	return out
}

// spanRow is the line an offset is drawn on.
//
// An offset that is both the end of one line and the start of the next is
// on the next one: a caret that has just been carried over the edge belongs
// where the next character it types will go.
func spanRow(spans []span, at int) int {
	row := 0

	for i, s := range spans {
		if at >= s.from {
			row = i
		}
	}

	return row
}

// spanOffset is the place in the value a row and a column of the box point
// at. A column past the end of a line is that line's end, so a click in the
// empty half of a line lands after its last character rather than nowhere.
func spanOffset(spans []span, row, col int) int {
	if len(spans) == 0 {
		return 0
	}

	s := spans[clamp(row, 0, len(spans)-1)]

	return clamp(s.from+col, s.from, s.to)
}

// spanText is one drawn line of the value.
func spanText(rs []rune, s span) string {
	return string(rs[clamp(s.from, 0, len(rs)):clamp(s.to, 0, len(rs))])
}

// spanWindow is which lines of a box are drawn when there are more of them
// than it is tall: the last ones, unless the caret is above them, in which
// case the ones the caret is on the bottom of. It answers the first line
// drawn.
func spanWindow(lines, height, caretRow int) int {
	if lines <= height {
		return 0
	}

	top := lines - height
	if caretRow < top {
		top = caretRow
	}

	return top
}
