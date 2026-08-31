package ui

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

// input is a value and the caret in it, in runes from the start.
type input struct {
	val string
	at  int
}

// newInput is a field already holding something, with the caret after it —
// where somebody who was handed a value would carry on typing.
func newInput(s string) input {
	return input{val: s, at: len([]rune(s))}
}

func (in input) String() string { return in.val }

func (in input) runes() []rune { return []rune(in.val) }

// empty is a field with nothing in it, whatever the caret says.
func (in input) empty() bool { return in.val == "" }

// setValue replaces what the field holds, and puts the caret at the end:
// the value came from somewhere else — an issue that was fetched, a
// clipboard — so there is no earlier position in it the reader meant.
func (in *input) setValue(s string) {
	in.val = s
	in.at = len([]rune(s))
}

// insert writes at the caret and leaves it after what was written.
func (in *input) insert(s string) {
	rs := in.runes()
	at := clamp(in.at, 0, len(rs))

	var b strings.Builder

	b.WriteString(string(rs[:at]))
	b.WriteString(s)
	b.WriteString(string(rs[at:]))

	in.val = b.String()
	in.at = at + len([]rune(s))
}

// backspace removes what is behind the caret. At the start of the value
// there is nothing behind it and nothing happens.
func (in *input) backspace() {
	rs := in.runes()

	at := clamp(in.at, 0, len(rs))
	if at == 0 {
		return
	}

	in.val = string(rs[:at-1]) + string(rs[at:])
	in.at = at - 1
}

// deleteForward removes what is in front of the caret, which is the other
// half of a field a reader can stand in the middle of.
func (in *input) deleteForward() {
	rs := in.runes()

	at := clamp(in.at, 0, len(rs))
	if at >= len(rs) {
		return
	}

	in.val = string(rs[:at]) + string(rs[at+1:])
	in.at = at
}

// moveTo puts the caret where it was pointed at, inside the value.
func (in *input) moveTo(at int) {
	in.at = clamp(at, 0, len(in.runes()))
}

// moveBy walks the caret one position at a time, in either direction.
func (in *input) moveBy(d int) {
	in.moveTo(in.at + d)
}

// lineStart and lineEnd are the ends of the line the caret is on rather
// than the ends of the value: a task written in four lines has four of
// each, and Home on the third of them is the third line's own start.
func (in *input) lineStart() {
	rs := in.runes()

	at := clamp(in.at, 0, len(rs))
	for at > 0 && rs[at-1] != '\n' {
		at--
	}

	in.at = at
}

func (in *input) lineEnd() {
	rs := in.runes()

	at := clamp(in.at, 0, len(rs))
	for at < len(rs) && rs[at] != '\n' {
		at++
	}

	in.at = at
}

// wordLeft and wordRight are the jump a reader expects from the option key:
// over the whitespace, then over the word behind or in front of it.
func (in *input) wordLeft() {
	rs := in.runes()
	at := clamp(in.at, 0, len(rs))

	for at > 0 && isBlank(rs[at-1]) {
		at--
	}

	for at > 0 && !isBlank(rs[at-1]) {
		at--
	}

	in.at = at
}

func (in *input) wordRight() {
	rs := in.runes()
	at := clamp(in.at, 0, len(rs))

	for at < len(rs) && !isBlank(rs[at]) {
		at++
	}

	for at < len(rs) && isBlank(rs[at]) {
		at++
	}

	in.at = at
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
