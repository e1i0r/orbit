package typing

// What is selected in a field, which is the other half of a caret. A reader
// who can put the caret anywhere expects to be able to take a stretch of
// text with it: to type over it, to delete it whole, to copy it out.
//
// A selection is not a third thing to keep in step with the value and the
// caret. It is one more offset — the anchor, where the gesture started —
// and what is selected is whatever lies between it and the caret. When the
// two are the same nothing is selected, and that is what every plain
// movement leaves behind: MoveTo carries the anchor along with it, and only
// Extend leaves it where it was.

// Selection is what lies between the anchor and the caret, in the order the
// value is written in rather than the order it was dragged in.
func (in Field) Selection() (from, to int) {
	n := len(in.Runes())

	from, to = clamp(in.Anchor, 0, n), clamp(in.At, 0, n)
	if from > to {
		from, to = to, from
	}

	return from, to
}

// HasSelection is whether anything is selected at all.
func (in Field) HasSelection() bool {
	from, to := in.Selection()

	return to > from
}

// Selected is the text between the two ends: what a copy puts on the
// clipboard, and what a cut takes away.
func (in Field) Selected() string {
	from, to := in.Selection()

	return string(in.Runes()[from:to])
}

// SelectAll is the whole value, with the caret at the end of it.
func (in *Field) SelectAll() {
	in.Anchor = 0
	in.At = len(in.Runes())
}

// Extend runs a movement without carrying the anchor along, which is what
// shift held with a movement means: the caret ends where the movement left
// it, and everything it crossed is now selected.
//
// It takes the movement rather than repeating any of it, so a shifted arrow
// selects across exactly what the unshifted one walks over — the word jump,
// the ends of a line, the lines of the box.
func (in *Field) Extend(move func(*Field)) {
	was := clamp(in.Anchor, 0, len(in.Runes()))

	move(in)

	in.Anchor = clamp(was, 0, len(in.Runes()))
}

// CutSelection removes what is selected and leaves the caret where it
// started. Typing over a selection replaces it and backspace takes it
// whole, which is why the field's own edits come through here first.
func (in *Field) CutSelection() {
	from, to := in.Selection()
	if to <= from {
		return
	}

	rs := in.Runes()

	in.Val = string(rs[:from]) + string(rs[to:])
	in.At = from
	in.Anchor = from
}
