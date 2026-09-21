package ui

// The box a note is typed into, drawn at every width there is room for.
//
// A reader meets this at one terminal size. The bug it hides is a box whose
// top border is one cell wider than its sides, or one that runs off the
// right of a narrow window — and either is invisible until somebody resizes
// the terminal they have used for a year.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestTheNoteBoxIsTheSameWidthOnEveryLine.
func TestTheNoteBoxIsTheSameWidthOnEveryLine(t *testing.T) {
	for _, said := range []struct {
		why  string
		text string
	}{
		{"nothing typed yet", ""},
		{"a short note", "check the linter first"},
		{"one long enough to wrap", strings.Repeat("a sentence that keeps going ", 8)},
		{"one with its own line breaks", "first\n\nthird"},
		{"one that is not English", "revisá el árbol de decisión ñ"},
	} {
		for w := 1; w <= 140; w++ {
			m, _ := testModel(t, 100, 30)
			m.note.open = true
			m.note.text = said.text

			rows := m.noteRows(15, w)
			if len(rows) == 0 {
				t.Errorf("%s at %d columns drew nothing", said.why, w)
				continue
			}

			// Nothing runs off the right of the window: a line wider than
			// the terminal wraps where the terminal decides, which puts the
			// bottom border a row lower than everything else expects.
			for i, line := range rows {
				if got := lipgloss.Width(line); got > w {
					t.Errorf("%s at %d columns: line %d is %d cells wide", said.why, w, i, got)
				}
			}
		}
	}
}

// TestTheNoteBoxSurvivesAWindowWithNoRoomInIt, which is what a terminal
// being dragged smaller is for a moment: a size nobody would choose and
// every window passes through.
func TestTheNoteBoxSurvivesAWindowWithNoRoomInIt(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.note.open = true
	m.note.text = "check the linter first"

	for _, one := range []struct{ h, w int }{
		{0, 80}, {15, 0}, {0, 0}, {-1, 80}, {15, -1}, {1, 1},
	} {
		rows := m.noteRows(one.h, one.w)

		if one.h <= 0 || one.w <= 0 {
			if rows != nil {
				t.Errorf("a window of %dx%d drew %d rows", one.w, one.h, len(rows))
			}

			continue
		}

		for i, line := range rows {
			if got := lipgloss.Width(line); got > one.w {
				t.Errorf("a window of %dx%d: line %d is %d cells wide", one.w, one.h, i, got)
			}
		}
	}
}

// TestANoteAlwaysHasRoomToType.
//
// Three lines, whatever was typed: a box that shrank to fit one word would
// jump every time the reader pressed a key, and one that grew without a
// floor would have nowhere to put the cursor on an empty note.
func TestANoteAlwaysHasRoomToType(t *testing.T) {
	for _, text := range []string{"", "one", "first\nsecond", "first\nsecond\nthird\nfourth"} {
		m, _ := testModel(t, 100, 30)
		m.note.open = true
		m.note.text = text

		rows := m.noteRows(20, 80)

		// The two borders and the prompt line are the furniture; what is
		// left is what the note is typed into.
		if len(rows) < 5 {
			t.Errorf("a note of %q drew %d rows, which is not a box with room in it", text, len(rows))
		}
	}
}
