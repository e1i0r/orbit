package ui

import tea "charm.land/bubbletea/v2"

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

// composeExtend is a movement made with shift held: the caret goes where
// the movement takes it and the anchor stays where it was, so what was
// crossed comes out selected.
//
// It takes the whole movement, up to and including the ones that leave the
// field — up on the first line of the box is the field above it, and there
// is nothing selected between two fields, so the anchor is only put back
// when the reader ended up where they started.
func (m Model) composeExtend(move func(Model) Model) Model {
	in := m.compose.active()
	if in == nil {
		return move(m)
	}

	was := in.anchor
	field := m.compose.field

	next := move(m)
	if next.compose.field != field {
		return next
	}

	if out := next.compose.active(); out != nil {
		out.anchor = clamp(was, 0, len(out.runes()))
	}

	return next
}

// composeAim is a cell of the form as a place inside a field: which field
// was pointed at, and where in what it holds.
func (m Model) composeAim(t Target) Model {
	m.compose.field = t.Pane

	if t.Pane == composeText && m.compose.tab == composeTabManual {
		return m.composePoint(t.Phase, t.Caret)
	}

	return m.composeCaret(func(in *input) { in.moveTo(t.Caret) })
}

// dragCaret is the pointer moving with the button still down: the caret
// follows it while the anchor stays on the cell the button went down on,
// which is a selection being dragged out of the text.
//
// A pointer that has wandered out of the field it started in is ignored
// rather than clamped to its edge. The drag is over that field's text, and
// a cell somewhere else on the form is not a place in it.
func (m Model) dragCaret(e tea.Mouse) Model {
	t := m.hit(e.X, e.Y)
	if t.Kind != TargetComposeCaret || t.Pane != m.held.target.Pane {
		return m
	}

	return m.composeExtend(func(mm Model) Model { return mm.composeAim(t) })
}

// composeCopy puts what is selected on the system clipboard, and takes it
// out of the field when it was a cut rather than a copy.
//
// Nothing selected is nothing to copy, and a clipboard that refused what it
// was handed leaves the text where it is: a cut that emptied the field
// after the clipboard dropped what was in it is text nobody can get back.
func (m Model) composeCopy(cut bool) Model {
	in := m.compose.active()
	if in == nil || !in.hasSelection() {
		return m
	}

	if !writeClipboard(in.selected()) {
		return m
	}

	if cut {
		in.cutSelection()
		m.onComposeChanged()
	}

	return m
}
