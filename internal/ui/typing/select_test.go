package typing

// What a field does with a stretch of text somebody has selected. The
// complaint this comes from is short: "no puedo seleccionar el texto".

import "testing"

func TestASelectionIsWhatLiesBetweenTheAnchorAndTheCaret(t *testing.T) {
	in := New("una tarea larga")
	in.MoveTo(4)
	in.Extend(func(in *Field) { in.MoveTo(9) })

	if got := in.Selected(); got != "tarea" {
		t.Errorf("what is selected reads %q, want %q", got, "tarea")
	}

	// Dragged the other way it is the same stretch: the value is read in
	// the order it was written, not in the order it was pointed at.
	in.MoveTo(9)
	in.Extend(func(in *Field) { in.MoveTo(4) })

	if got := in.Selected(); got != "tarea" {
		t.Errorf("selected backwards reads %q, want %q", got, "tarea")
	}
}

func TestAMovementOnItsOwnSelectsNothing(t *testing.T) {
	in := New("una tarea")
	in.MoveTo(2)
	in.Extend(func(in *Field) { in.MoveTo(5) })

	in.MoveBy(1)

	if in.HasSelection() {
		t.Errorf("a plain arrow left %q selected, want the selection dropped", in.Selected())
	}
}

func TestTypingOverASelectionReplacesIt(t *testing.T) {
	in := New("una tarea larga")
	in.MoveTo(4)
	in.Extend(func(in *Field) { in.MoveTo(9) })

	in.Insert("nota")

	if got := in.String(); got != "una nota larga" {
		t.Errorf("typing over the selection left %q", got)
	}

	if in.At != 8 || in.HasSelection() {
		t.Errorf("the caret is at %d with %q selected, want 8 and nothing", in.At, in.Selected())
	}
}

// Backspace takes the whole selection rather than the character behind the
// caret, which is what makes selecting a word worth doing at all.
func TestBackspaceAndDeleteTakeTheWholeSelection(t *testing.T) {
	in := New("una tarea larga")
	in.MoveTo(3)
	in.Extend(func(in *Field) { in.MoveTo(9) })

	in.Backspace()

	if got := in.String(); got != "una larga" {
		t.Errorf("backspace over the selection left %q", got)
	}

	in.MoveTo(0)
	in.Extend(func(in *Field) { in.LineEnd() })
	in.DeleteForward()

	if got := in.String(); got != "" {
		t.Errorf("delete over everything selected left %q, want an empty field", got)
	}
}

func TestSelectAllIsTheWholeValue(t *testing.T) {
	in := New("uno\ndos")
	in.MoveTo(1)
	in.SelectAll()

	if got := in.Selected(); got != "uno\ndos" {
		t.Errorf("select all took %q", got)
	}
}

// The word jump and the ends of a line are movements like any other, so
// shift held with them selects across exactly what they walk over.
func TestExtendingSelectsAcrossWhateverTheMovementWalks(t *testing.T) {
	in := New("una tarea larga")
	in.MoveTo(4)
	in.Extend((*Field).WordRight)

	if got := in.Selected(); got != "tarea " {
		t.Errorf("shift with the word jump took %q, want %q", got, "tarea ")
	}

	in.Extend((*Field).LineEnd)

	if got := in.Selected(); got != "tarea larga" {
		t.Errorf("shift with end took %q, want the rest of the line", got)
	}
}
