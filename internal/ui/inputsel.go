package ui

// What is selected in a field, which is the other half of a caret. A reader
// who can put the caret anywhere expects to be able to take a stretch of
// text with it: to type over it, to delete it whole, to copy it out.
//
// A selection is not a third thing to keep in step with the value and the
// caret. It is one more offset — the anchor, where the gesture started —
// and what is selected is whatever lies between it and the caret. When the
// two are the same nothing is selected, and that is what every plain
// movement leaves behind: moveTo carries the anchor along with it, and only
// extend leaves it where it was.

// selection is what lies between the anchor and the caret, in the order the
// value is written in rather than the order it was dragged in.
func (in input) selection() (from, to int) {
	n := len(in.runes())

	from, to = clamp(in.anchor, 0, n), clamp(in.at, 0, n)
	if from > to {
		from, to = to, from
	}

	return from, to
}

// hasSelection is whether anything is selected at all.
func (in input) hasSelection() bool {
	from, to := in.selection()

	return to > from
}

// selected is the text between the two ends: what a copy puts on the
// clipboard, and what a cut takes away.
func (in input) selected() string {
	from, to := in.selection()

	return string(in.runes()[from:to])
}

// selectAll is the whole value, with the caret at the end of it.
func (in *input) selectAll() {
	in.anchor = 0
	in.at = len(in.runes())
}

// extend runs a movement without carrying the anchor along, which is what
// shift held with a movement means: the caret ends where the movement left
// it, and everything it crossed is now selected.
//
// It takes the movement rather than repeating any of it, so a shifted arrow
// selects across exactly what the unshifted one walks over — the word jump,
// the ends of a line, the lines of the box.
func (in *input) extend(move func(*input)) {
	was := clamp(in.anchor, 0, len(in.runes()))

	move(in)

	in.anchor = clamp(was, 0, len(in.runes()))
}

// cutSelection removes what is selected and leaves the caret where it
// started. Typing over a selection replaces it and backspace takes it
// whole, which is why the field's own edits come through here first.
func (in *input) cutSelection() {
	from, to := in.selection()
	if to <= from {
		return
	}

	rs := in.runes()

	in.val = string(rs[:from]) + string(rs[to:])
	in.at = from
	in.anchor = from
}
