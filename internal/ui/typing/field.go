package typing

import "strings"

// A field somebody types into: what it holds and where the caret is.
//
// The form used to hold three plain strings, and every key appended to the
// end of one of them: what was typed landed after everything else no matter
// where the reader was looking, and backspace ate the last character rather
// than the one behind the caret. There was nothing for a click inside the
// box to move, either, which is what made the form feel unlike every other
// text field on the machine.
//
// The caret is counted in runes and not in bytes, because it is a position
// in what the reader sees. A value is never held apart from its caret, so a
// caret that outlives the text it pointed into is not a state this can be
// in.

// Field is a value, the caret in it, and the other end of what is
// selected, both counted in runes from the start.
//
// The three are exported because what draws a field is not this package:
// the window paints it, and a caret nobody outside can read is a caret
// nobody can draw. They are written through the methods below rather than
// by hand — every one of them keeps the three in step.
type Field struct {
	Val string
	At  int
	// Anchor is where a selection started. It is equal to At whenever
	// nothing is selected, which is what every plain movement leaves
	// behind, so there is no third state to keep in step with the other
	// two: see select.go.
	Anchor int
}

// New is a field already holding something, with the caret after it —
// where somebody who was handed a value would carry on typing.
func New(s string) Field {
	at := len([]rune(s))

	return Field{Val: s, At: at, Anchor: at}
}

func (in Field) String() string { return in.Val }

// Runes is the value as the caret counts it, which is the only way anything
// here measures a position.
func (in Field) Runes() []rune { return []rune(in.Val) }

// Empty is a field with nothing in it, whatever the caret says.
func (in Field) Empty() bool { return in.Val == "" }

// SetValue replaces what the field holds, and puts the caret at the end:
// the value came from somewhere else — an issue that was fetched, a
// clipboard — so there is no earlier position in it the reader meant.
func (in *Field) SetValue(s string) {
	in.Val = s
	in.At = len([]rune(s))
	in.Anchor = in.At
}

// Insert writes at the caret and leaves it after what was written. What was
// selected goes first: typing over a selection replaces it, the way it does
// in every other field on the machine.
func (in *Field) Insert(s string) {
	in.CutSelection()

	rs := in.Runes()
	at := clamp(in.At, 0, len(rs))

	var b strings.Builder

	b.WriteString(string(rs[:at]))
	b.WriteString(s)
	b.WriteString(string(rs[at:]))

	in.Val = b.String()
	in.At = at + len([]rune(s))
	in.Anchor = in.At
}

// Backspace removes what is selected, or what is behind the caret when
// nothing is. At the start of the value there is nothing behind it and
// nothing happens.
func (in *Field) Backspace() {
	if in.HasSelection() {
		in.CutSelection()

		return
	}

	rs := in.Runes()

	at := clamp(in.At, 0, len(rs))
	if at == 0 {
		return
	}

	in.Val = string(rs[:at-1]) + string(rs[at:])
	in.At = at - 1
	in.Anchor = in.At
}

// DeleteForward removes what is in front of the caret, which is the other
// half of a field a reader can stand in the middle of.
func (in *Field) DeleteForward() {
	if in.HasSelection() {
		in.CutSelection()

		return
	}

	rs := in.Runes()

	at := clamp(in.At, 0, len(rs))
	if at >= len(rs) {
		return
	}

	in.Val = string(rs[:at]) + string(rs[at+1:])
	in.At = at
	in.Anchor = in.At
}

// MoveTo puts the caret where it was pointed at, inside the value, and
// carries the anchor along with it: a movement on its own selects nothing,
// and only extend leaves the anchor behind.
func (in *Field) MoveTo(at int) {
	in.At = clamp(at, 0, len(in.Runes()))
	in.Anchor = in.At
}

// MoveBy walks the caret one position at a time, in either direction.
func (in *Field) MoveBy(d int) {
	in.MoveTo(in.At + d)
}

// LineStart and LineEnd are the ends of the line the caret is on rather
// than the ends of the value: a task written in four lines has four of
// each, and Home on the third of them is the third line's own start.
func (in *Field) LineStart() {
	rs := in.Runes()

	at := clamp(in.At, 0, len(rs))
	for at > 0 && rs[at-1] != '\n' {
		at--
	}

	in.MoveTo(at)
}

// LineEnd is the other end of that line: see LineStart.
func (in *Field) LineEnd() {
	rs := in.Runes()

	at := clamp(in.At, 0, len(rs))
	for at < len(rs) && rs[at] != '\n' {
		at++
	}

	in.MoveTo(at)
}

// WordLeft and WordRight are the jump a reader expects from the option key:
// over the whitespace, then over the word behind or in front of it.
func (in *Field) WordLeft() {
	rs := in.Runes()
	at := clamp(in.At, 0, len(rs))

	for at > 0 && isBlank(rs[at-1]) {
		at--
	}

	for at > 0 && !isBlank(rs[at-1]) {
		at--
	}

	in.MoveTo(at)
}

// WordRight is that jump the other way: see WordLeft.
func (in *Field) WordRight() {
	rs := in.Runes()
	at := clamp(in.At, 0, len(rs))

	for at < len(rs) && !isBlank(rs[at]) {
		at++
	}

	for at < len(rs) && isBlank(rs[at]) {
		at++
	}

	in.MoveTo(at)
}

func isBlank(r rune) bool { return r == ' ' || r == '\t' || r == '\n' }

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}
