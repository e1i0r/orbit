package typing

// Where a value breaks when it is drawn, counted so that it can be counted
// back. splitIntoLines cannot do this — it rejoins words with one space —
// and a caret that cannot be counted back is a caret a click cannot move.

import (
	"strings"
	"testing"
)

// The lines are back to back and hold every rune, but for the newlines,
// which are where a line ends rather than something drawn on one. If they
// were not, the offset a click answers with would drift by the number of
// breaks above it.
func TestTheDrawnLinesOfAValueAccountForEveryCharacter(t *testing.T) {
	text := "una tarea larga que no entra en la caja\ny otra línea"
	rs := []rune(text)

	spans := Wrap(rs, 12)

	var b strings.Builder

	at := 0

	for _, s := range spans {
		for ; at < s.From; at++ {
			if rs[at] != '\n' {
				t.Fatalf("the character at %d is on no line at all: %q", at, string(rs[at]))
			}
		}

		b.WriteString(SpanText(rs, s))

		at = s.To
	}

	// What is missing between the lines is the newlines, which are not
	// drawn: everything else is there, in order.
	if got := b.String(); got != strings.ReplaceAll(text, "\n", "") {
		t.Errorf("the lines put together read %q", got)
	}
}

// TestWhereAValueBreaks, line by line. The count a click and a caret are
// both read back through is the one the breaks make, so the break itself is
// the thing to state: at the last space that fits, at the box's edge when
// there is no space to take, and at the newline whatever the width is.
func TestWhereAValueBreaks(t *testing.T) {
	for _, c := range []struct {
		text  string
		width int
		want  []Span
	}{
		// The break takes the space with the line it ends, and the line
		// after it starts on the word.
		{"abcd ef ghij", 6, []Span{{0, 5}, {5, 8}, {8, 12}}},

		// A last chunk exactly as wide as the box is that chunk, not a
		// chunk broken at the space inside it.
		{"abc de f", 4, []Span{{0, 4}, {4, 8}}},

		// A word longer than the box is broken where the box ends,
		// because the alternative is a line that overflows it.
		{"palabralarguisima", 6, []Span{{0, 6}, {6, 12}, {12, 17}}},

		// A newline ends a line whatever is left of the width, and it
		// is not drawn on either side of the break.
		{"hola\nmundo", 8, []Span{{0, 4}, {5, 10}}},

		// One column is a column: the value comes out a character to
		// the line rather than whole.
		{"hola", 1, []Span{{0, 1}, {1, 2}, {2, 3}, {3, 4}}},

		// The blank line between two paragraphs is a line: dropped, it
		// takes a row off the box and the caret with it.
		{"a\n\nb", 4, []Span{{0, 1}, {2, 2}, {3, 4}}},

		// A box with no columns at all is one every window passes
		// through while somebody drags its corner. It draws the line
		// whole rather than taking it a column at a time and never
		// reaching the end of it.
		{"hola", 0, []Span{{0, 4}}},
	} {
		got := Wrap([]rune(c.text), c.width)
		if len(got) != len(c.want) {
			t.Errorf("%q in a box of %d broke into %v, want %v", c.text, c.width, got, c.want)

			continue
		}

		for i, s := range got {
			if s != c.want[i] {
				t.Errorf("%q in a box of %d broke into %v, want %v", c.text, c.width, got, c.want)

				break
			}
		}
	}
}

// TestPointingPastTheLastLineLandsOnIt. A click is a row and a column of a
// box, and a box is taller than the value in it whenever the value is short
// — so the row pointed at is routinely one the value has no line for.
func TestPointingPastTheLastLineLandsOnIt(t *testing.T) {
	rs := []rune("hola\nmundo")
	spans := Wrap(rs, 8)

	if got := SpanOffset(spans, len(spans), 0); got != spans[len(spans)-1].From {
		t.Errorf("a click a row below the last line answered %d, want the head of the last line", got)
	}

	if got := SpanOffset(spans, 99, 99); got != spans[len(spans)-1].To {
		t.Errorf("a click past every line and every column answered %d, want the end of the value", got)
	}

	if got := SpanOffset(nil, 0, 0); got != 0 {
		t.Errorf("a click on a box with nothing in it answered %d", got)
	}
}

func TestALineNeverComesOutWiderThanTheBox(t *testing.T) {
	rs := []rune("palabras cortas y una palabralarguisimaquenoentra al final")

	for _, s := range Wrap(rs, 10) {
		if s.To-s.From > 10 {
			t.Errorf("a line of %d characters was drawn in a box of 10: %q", s.To-s.From, SpanText(rs, s))
		}
	}
}

// A caret carried over the edge is at the head of the next line, not at the
// tail of the one it left: what it types next goes on the new line.
func TestACaretOnABreakIsOnTheLineItIsAboutToWriteOn(t *testing.T) {
	rs := []rune("aaa bbb ccc")

	spans := Wrap(rs, 4)
	if len(spans) != 3 {
		t.Fatalf("the value was drawn as %d lines, want 3", len(spans))
	}

	if row := SpanRow(spans, spans[1].From); row != 1 {
		t.Errorf("the start of the second line is on row %d, want 1", row)
	}

	if row := SpanRow(spans, len(rs)); row != 2 {
		t.Errorf("the end of the value is on row %d, want the last line", row)
	}
}

// Which is what a click has to answer: a column of the third line is that
// far into the third line, not that far into the value.
func TestPointingAtALineAnswersWithThePlaceInTheValue(t *testing.T) {
	rs := []rune("aaa bbb ccc")
	spans := Wrap(rs, 4)

	if got := SpanOffset(spans, 2, 1); got != 9 {
		t.Errorf("the second cell of the third line is offset %d, want 9", got)
	}

	// Past the end of a line is the end of that line. The half of a row
	// that has no text on it still belongs to the row.
	if got := SpanOffset(spans, 0, 40); got != spans[0].To {
		t.Errorf("a cell past the first line is offset %d, want its end %d", got, spans[0].To)
	}
}

// The box shows six lines of a task that may have more. Which six is
// decided by where the reader is: the last ones while they are typing at
// the end, and the caret's own once they have walked back up into it.
func TestTheBoxScrollsToWhereTheCaretIs(t *testing.T) {
	if got := SpanWindow(3, 6, 2); got != 0 {
		t.Errorf("a value shorter than the box starts at line %d, want 0", got)
	}

	if got := SpanWindow(10, 6, 9); got != 4 {
		t.Errorf("a caret on the last of ten lines shows from %d, want 4", got)
	}

	if got := SpanWindow(10, 6, 1); got != 1 {
		t.Errorf("a caret on the second of ten lines shows from %d, want 1", got)
	}
}
