package ui

// The caret of the form: what a key does to the field the reader is in, and
// how the box the task is written in is measured.
//
// The measurement lives here rather than in the drawing because three
// things need the same answer: what is drawn, which line the caret goes up
// or down to, and where a click inside the box lands. A box wrapped at one
// width and pointed at as if it were another puts the caret somewhere the
// reader did not point.

// composeBoxWidth is how wide the box a task is written in is drawn, and
// composeInnerWidth is how much of that the text itself gets.
func composeBoxWidth(w int) int {
	return clamp(w-8, 24, 84)
}

func composeInnerWidth(w int) int {
	return composeBoxWidth(w) - 4
}

// composeTextRows is how many lines of the box are drawn at once.
const composeTextRows = 6

// composeEdit is a key that changes what a field holds. What it writes can
// be a URL, so the form is given the chance to recognise one.
func (m Model) composeEdit(f func(*input)) Model {
	in := m.compose.active()
	if in == nil {
		return m
	}

	f(in)
	m.onComposeChanged()

	return m
}

// composeCaret is a key that only moves. Nothing was written, so nothing is
// re-read.
func (m Model) composeCaret(f func(*input)) Model {
	if in := m.compose.active(); in != nil {
		f(in)
	}

	return m
}

// composeUp is the up and down arrows, which mean two things on this form.
//
// Inside the box a task is written in they are the lines of the task: a
// reader who wrote four of them is walking their own text. On the first
// line going up, or the last going down, there is no line left and the
// arrow means what it means everywhere else on the form — the field above
// or the field below.
func (m Model) composeUp(d int) Model {
	in := m.compose.active()
	if in == nil || m.compose.tab != composeTabManual || m.compose.field != composeText {
		return m.composeMove(d)
	}

	rs := in.runes()
	spans := wrapSpans(rs, composeInnerWidth(m.frame.Body.W))

	row := spanRow(spans, in.at)
	col := in.at - spans[row].from

	next := row + d
	if next < 0 || next >= len(spans) {
		return m.composeMove(d)
	}

	in.moveTo(spanOffset(spans, next, col))

	return m
}

// composePoint is a click inside the box: the row and column of the cell
// that was pointed at, as a place in the text.
func (m Model) composePoint(row, col int) Model {
	in := m.compose.active()
	if in == nil {
		return m
	}

	spans := wrapSpans(in.runes(), composeInnerWidth(m.frame.Body.W))
	top := spanWindow(len(spans), composeTextRows, spanRow(spans, in.at))

	in.moveTo(spanOffset(spans, top+row, col))

	return m
}
