package flows

// The box a paragraph is typed into: how wide it is, what it shows when it
// is empty, and which end of a long one is on the screen.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// TestTheBoxClosesInTheSameColumnOnEveryRow. A top and a bottom that do not
// agree with the sides between them is not a box, it is three kinds of line
// near each other — and the reader is typing inside it.
func TestTheBoxClosesInTheSameColumnOnEveryRow(t *testing.T) {
	s, e := designing(t, 110, 45)
	_ = e

	for w := 60; w <= 160; w++ {
		for _, content := range []string{
			"",
			"short",
			strings.Repeat("a paragraph that keeps going and going ", 6),
		} {
			rows := s.textBox(flowFieldPrompt, content, "(say something here)", 4, w)
			if len(rows) < 2 {
				t.Fatalf("at %d columns the box is %d rows", w, len(rows))
			}

			want := lipgloss.Width(rows[0].text)

			for i, r := range rows {
				if got := lipgloss.Width(r.text); got != want {
					t.Fatalf("at %d columns row %d of the box is %d cells and the top is %d: %q",
						w, i, got, want, ansi.Strip(r.text))
				}
			}

			if want > w {
				t.Fatalf("at %d columns the box is %d cells wide", w, want)
			}
		}
	}
}

// TestTheBoxIsAsWideAsTheWindowAllows, up to a line length that is still
// comfortable to read back: what is typed here is a paragraph, and eighty
// columns of it on a hundred-and-thirty column terminal was a third of the
// screen left empty beside the field being written in.
func TestTheBoxIsAsWideAsTheWindowAllows(t *testing.T) {
	s, _ := designing(t, 110, 45)

	for _, c := range []struct {
		w, want int
	}{
		{60, 58},   // the window less the margins
		{100, 98},  //
		{110, 108}, //
		// Past the measure it stops growing: a hundred and four columns
		// of box, and the rest of the terminal left alone.
		{140, 108},
		{200, 108},
	} {
		rows := s.textBox(flowFieldPrompt, "algo", "", 4, c.w)
		if got := lipgloss.Width(rows[0].text); got != c.want {
			t.Errorf("in a window of %d the box is %d cells wide, want %d", c.w, got, c.want)
		}
	}
}

// TestAnEmptyBoxSaysWhatToWriteInIt, and one with something in it does not.
func TestAnEmptyBoxSaysWhatToWriteInIt(t *testing.T) {
	s, _ := designing(t, 110, 45)

	const ghost = "(what is this flow for?)"

	empty := ansi.Strip(joinRows(s.textBox(flowFieldDescription, "", ghost, 4, 100)))
	if !strings.Contains(empty, ghost) {
		t.Errorf("an empty box does not say what to write in it:\n%s", empty)
	}

	full := ansi.Strip(joinRows(s.textBox(flowFieldDescription, "already written", ghost, 4, 100)))
	if strings.Contains(full, ghost) {
		t.Errorf("a box with something in it still says what to write:\n%s", full)
	}

	if !strings.Contains(full, "already written") {
		t.Errorf("the box does not show what is in it:\n%s", full)
	}
}

// TestTheCaretIsInTheFieldBeingTypedInto, and in no other.
func TestTheCaretIsInTheFieldBeingTypedInto(t *testing.T) {
	s, _ := designing(t, 110, 45)
	s.field = flowFieldPrompt

	on := ansi.Strip(joinRows(s.textBox(flowFieldPrompt, "algo", "", 4, 100)))
	if !strings.Contains(on, "algo_") {
		t.Errorf("the box being typed into has no caret:\n%s", on)
	}

	off := ansi.Strip(joinRows(s.textBox(flowFieldDescription, "algo", "", 4, 100)))
	if strings.Contains(off, "algo_") {
		t.Errorf("a box that is not being typed into carries a caret:\n%s", off)
	}
}

// TestALongParagraphShowsItsEnd. The caret is at the end of what has been
// typed, and a box that showed the first lines would scroll away from the
// reader as they wrote.
func TestALongParagraphShowsItsEnd(t *testing.T) {
	s, _ := designing(t, 110, 45)
	s.field = flowFieldPrompt

	var b strings.Builder
	for i := range 12 {
		b.WriteString("line" + string(rune('a'+i)) + " of the paragraph, long enough to wrap on its own. ")
	}

	drawn := ansi.Strip(joinRows(s.textBox(flowFieldPrompt, b.String(), "", 3, 100)))

	if !strings.Contains(drawn, "_") {
		t.Errorf("the end of a long paragraph is not on the screen:\n%s", drawn)
	}

	if strings.Contains(drawn, "linea of the paragraph") {
		t.Errorf("a long paragraph shows its head rather than its tail:\n%s", drawn)
	}

	// A paragraph that fits keeps its first line: the cut is for what is
	// over the rows, not for anything that is close to them.
	short := ansi.Strip(joinRows(s.textBox(flowFieldPrompt, "one\ntwo\nthree", "", 3, 100)))
	for _, want := range []string{"one", "two", "three"} {
		if !strings.Contains(short, want) {
			t.Errorf("a paragraph of exactly three rows lost %q:\n%s", want, short)
		}
	}
}

// joinRows is the rows of a box as one string.
func joinRows(rows []builderLine) string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.text)
	}

	return strings.Join(out, "\n")
}
