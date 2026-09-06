package typing

// What a field does to what is typed into it. Every case here is one the
// form got wrong while it held plain strings: text landed at the end, and
// Backspace ate the end, wherever the reader was standing.

import "testing"

func TestTypingGoesInAtTheCaretAndNotAtTheEnd(t *testing.T) {
	in := New("hola mundo")
	in.MoveTo(4)
	in.Insert(" gran")

	if got := in.String(); got != "hola gran mundo" {
		t.Errorf("typing at 4 left %q, want the words in the order they were typed in", got)
	}

	if in.At != 9 {
		t.Errorf("the caret is at %d, want 9 — after what was just written", in.At)
	}
}

func TestBackspaceAndDeleteTakeTheCharacterOnEitherSideOfTheCaret(t *testing.T) {
	in := New("orbita")
	in.MoveTo(3)

	in.Backspace()

	if got := in.String(); got != "orita" {
		t.Errorf("backspace at 3 left %q, want the character behind the caret gone", got)
	}

	in.DeleteForward()

	if got := in.String(); got != "orta" {
		t.Errorf("delete at 2 left %q, want the character in front of the caret gone", got)
	}

	if in.At != 2 {
		t.Errorf("the caret moved to %d, want it to stay at 2", in.At)
	}
}

// Both ends refuse rather than wrap. A caret that fell off the start would
// delete the last character of the value, which is the bug this whole file
// is about, arrived at from the other side.
func TestTheEndsOfAFieldHoldTheCaret(t *testing.T) {
	in := New("ab")

	in.MoveTo(0)
	in.Backspace()

	if got := in.String(); got != "ab" || in.At != 0 {
		t.Errorf("backspace at the start left %q at %d, want %q at 0", got, in.At, "ab")
	}

	in.MoveTo(2)
	in.DeleteForward()
	in.MoveBy(1)

	if got := in.String(); got != "ab" || in.At != 2 {
		t.Errorf("delete at the end left %q at %d, want %q at 2", got, in.At, "ab")
	}
}

// A caret counted in bytes lands in the middle of a character, and what is
// drawn from there is not text.
func TestTheCaretCountsCharactersAndNotBytes(t *testing.T) {
	in := New("café ñu")
	in.MoveTo(4)
	in.Insert("!")

	if got := in.String(); got != "café! ñu" {
		t.Errorf("typing after the fourth character left %q", got)
	}

	in.MoveTo(len(in.Runes()))
	in.Backspace()

	if got := in.String(); got != "café! ñ" {
		t.Errorf("backspace at the end left %q, want a whole character gone", got)
	}
}

func TestHomeAndEndAreTheLineTheCaretIsOnAndNotTheWholeValue(t *testing.T) {
	in := New("uno\ndos\ntres")
	in.MoveTo(5) // inside "dos"

	in.LineStart()

	if in.At != 4 {
		t.Errorf("home left the caret at %d, want 4 — the start of the second line", in.At)
	}

	in.LineEnd()

	if in.At != 7 {
		t.Errorf("end left the caret at %d, want 7 — the end of the second line", in.At)
	}
}

func TestTheWordJumpCrossesTheBlanksAndThenTheWord(t *testing.T) {
	in := New("una tarea  larga")

	in.WordLeft()

	if in.At != 11 {
		t.Errorf("the first jump back left the caret at %d, want 11 — the head of the last word", in.At)
	}

	in.WordLeft()

	if in.At != 4 {
		t.Errorf("the second jump back left the caret at %d, want 4", in.At)
	}

	in.WordRight()

	if in.At != 11 {
		t.Errorf("the jump forward left the caret at %d, want 11 — past the word and its blanks", in.At)
	}
}

// A value handed to the field from somewhere else — a fetched issue, the
// clipboard — is somebody else's, and the reader has never been anywhere
// inside it. The caret goes where they would carry on typing.
func TestAValueSetFromOutsideLeavesTheCaretAfterIt(t *testing.T) {
	in := New("")
	in.SetValue("ORBIT-42")

	if in.At != 8 {
		t.Errorf("the caret is at %d after being handed a value, want 8", in.At)
	}
}
