package compose

import (
	"github.com/e1i0r/orbit/internal/ui/clip"
	"github.com/e1i0r/orbit/internal/ui/typing"
)

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
//
// They are the model's because the paste button stands to the right of the
// box and the words on it are translated: a box measured without asking how
// much room that takes would push the button off the edge in whichever
// language spends the most cells on it.
func (s State) composeBoxWidth(w int, e Env) int {
	return min(max(w-composeLabelStart-2-composePasteRoom(e.Words), 24), 84)
}

func (s State) composeInnerWidth(w int, e Env) int {
	return s.composeBoxWidth(w, e) - 4
}

// composeTextRows is how many lines of the box are drawn at once.
const composeTextRows = 6

// composeInBox is whether the field the cursor is in is the one this tab
// draws as a box: the task on one, the URL on the other.
//
// What it decides is what a line means. In a box the lines are the value
// wrapped, so up and down walk them and a click lands at a row and a
// column; everywhere else on the form a line is a field and the same keys
// leave it.
func (s State) composeInBox() bool {
	if s.tab == composeTabURL {
		return s.field == composeURL
	}

	return s.field == composeText
}

// composeEdit is a key that changes what a field holds. What it writes can
// be a URL, so the form is given the chance to recognise one.
func (s State) composeEdit(f func(*typing.Field)) State {
	in := s.active()
	if in == nil {
		return s
	}

	f(in)
	s.onComposeChanged()

	return s
}

// composeCaret is a key that only moves. Nothing was written, so nothing is
// re-read.
func (s State) composeCaret(f func(*typing.Field)) State {
	if in := s.active(); in != nil {
		f(in)
	}

	return s
}

// composeUp is the up and down arrows, which mean two things on this form.
//
// Inside the box a task is written in they are the lines of the task: a
// reader who wrote four of them is walking their own text. On the first
// line going up, or the last going down, there is no line left and the
// arrow means what it means everywhere else on the form — the field above
// or the field below.
func (s State) composeUp(d int, e Env) State {
	in := s.active()
	if in == nil || !s.composeInBox() {
		return s.composeMove(d)
	}

	rs := in.Runes()
	spans := typing.Wrap(rs, s.composeInnerWidth(e.Frame.Body.W, e))

	row := typing.SpanRow(spans, in.At)
	col := in.At - spans[row].From

	next := row + d
	if next < 0 || next >= len(spans) {
		return s.composeMove(d)
	}

	in.MoveTo(typing.SpanOffset(spans, next, col))

	return s
}

// composePoint is a click inside the box: the row and column of the cell
// that was pointed at, as a place in the text.
func (s State) composePoint(row, col int, e Env) State {
	in := s.active()
	if in == nil {
		return s
	}

	spans := typing.Wrap(in.Runes(), s.composeInnerWidth(e.Frame.Body.W, e))
	top := typing.SpanWindow(len(spans), composeTextRows, typing.SpanRow(spans, in.At))

	in.MoveTo(typing.SpanOffset(spans, top+row, col))

	return s
}

// composeExtend is a movement made with shift held: the caret goes where
// the movement takes it and the anchor stays where it was, so what was
// crossed comes out selected.
//
// It takes the whole movement, up to and including the ones that leave the
// field — up on the first line of the box is the field above it, and there
// is nothing selected between two fields, so the anchor is only put back
// when the reader ended up where they started.
func (s State) composeExtend(move func(State) State) State {
	in := s.active()
	if in == nil {
		return move(s)
	}

	was := in.Anchor
	field := s.field

	next := move(s)
	if next.field != field {
		return next
	}

	if out := next.active(); out != nil {
		out.Anchor = min(max(was, 0), len(out.Runes()))
	}

	return next
}

// composeCopy puts what is selected on the system clipboard, and takes it
// out of the field when it was a cut rather than a copy.
//
// Nothing selected is nothing to copy, and a clipboard that refused what it
// was handed leaves the text where it is: a cut that emptied the field
// after the clipboard dropped what was in it is text nobody can get back.
func (s State) composeCopy(cut bool) State {
	in := s.active()
	if in == nil || !in.HasSelection() {
		return s
	}

	if !clip.Write(in.Selected()) {
		return s
	}

	if cut {
		in.CutSelection()
		s.onComposeChanged()
	}

	return s
}
