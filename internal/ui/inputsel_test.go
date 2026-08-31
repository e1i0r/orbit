package ui

// What a field does with a stretch of text somebody has selected. The
// complaint this comes from is short: "no puedo seleccionar el texto".

import "testing"

func TestASelectionIsWhatLiesBetweenTheAnchorAndTheCaret(t *testing.T) {
	in := newInput("una tarea larga")
	in.moveTo(4)
	in.extend(func(in *input) { in.moveTo(9) })

	if got := in.selected(); got != "tarea" {
		t.Errorf("what is selected reads %q, want %q", got, "tarea")
	}

	// Dragged the other way it is the same stretch: the value is read in
	// the order it was written, not in the order it was pointed at.
	in.moveTo(9)
	in.extend(func(in *input) { in.moveTo(4) })

	if got := in.selected(); got != "tarea" {
		t.Errorf("selected backwards reads %q, want %q", got, "tarea")
	}
}

func TestAMovementOnItsOwnSelectsNothing(t *testing.T) {
	in := newInput("una tarea")
	in.moveTo(2)
	in.extend(func(in *input) { in.moveTo(5) })

	in.moveBy(1)

	if in.hasSelection() {
		t.Errorf("a plain arrow left %q selected, want the selection dropped", in.selected())
	}
}

func TestTypingOverASelectionReplacesIt(t *testing.T) {
	in := newInput("una tarea larga")
	in.moveTo(4)
	in.extend(func(in *input) { in.moveTo(9) })

	in.insert("nota")

	if got := in.String(); got != "una nota larga" {
		t.Errorf("typing over the selection left %q", got)
	}

	if in.at != 8 || in.hasSelection() {
		t.Errorf("the caret is at %d with %q selected, want 8 and nothing", in.at, in.selected())
	}
}

// Backspace takes the whole selection rather than the character behind the
// caret, which is what makes selecting a word worth doing at all.
func TestBackspaceAndDeleteTakeTheWholeSelection(t *testing.T) {
	in := newInput("una tarea larga")
	in.moveTo(3)
	in.extend(func(in *input) { in.moveTo(9) })

	in.backspace()

	if got := in.String(); got != "una larga" {
		t.Errorf("backspace over the selection left %q", got)
	}

	in.moveTo(0)
	in.extend(func(in *input) { in.lineEnd() })
	in.deleteForward()

	if got := in.String(); got != "" {
		t.Errorf("delete over everything selected left %q, want an empty field", got)
	}
}

func TestSelectAllIsTheWholeValue(t *testing.T) {
	in := newInput("uno\ndos")
	in.moveTo(1)
	in.selectAll()

	if got := in.selected(); got != "uno\ndos" {
		t.Errorf("select all took %q", got)
	}
}

// The word jump and the ends of a line are movements like any other, so
// shift held with them selects across exactly what they walk over.
func TestExtendingSelectsAcrossWhateverTheMovementWalks(t *testing.T) {
	in := newInput("una tarea larga")
	in.moveTo(4)
	in.extend((*input).wordRight)

	if got := in.selected(); got != "tarea " {
		t.Errorf("shift with the word jump took %q, want %q", got, "tarea ")
	}

	in.extend((*input).lineEnd)

	if got := in.selected(); got != "tarea larga" {
		t.Errorf("shift with end took %q, want the rest of the line", got)
	}
}
