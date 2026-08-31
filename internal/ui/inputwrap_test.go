package ui

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

	spans := wrapSpans(rs, 12)

	var b strings.Builder

	at := 0

	for _, s := range spans {
		for ; at < s.from; at++ {
			if rs[at] != '\n' {
				t.Fatalf("the character at %d is on no line at all: %q", at, string(rs[at]))
			}
		}

		b.WriteString(spanText(rs, s))

		at = s.to
	}

	// What is missing between the lines is the newlines, which are not
	// drawn: everything else is there, in order.
	if got := b.String(); got != strings.ReplaceAll(text, "\n", "") {
		t.Errorf("the lines put together read %q", got)
	}
}

func TestALineNeverComesOutWiderThanTheBox(t *testing.T) {
	rs := []rune("palabras cortas y una palabralarguisimaquenoentra al final")

	for _, s := range wrapSpans(rs, 10) {
		if s.to-s.from > 10 {
			t.Errorf("a line of %d characters was drawn in a box of 10: %q", s.to-s.from, spanText(rs, s))
		}
	}
}

// A caret carried over the edge is at the head of the next line, not at the
// tail of the one it left: what it types next goes on the new line.
func TestACaretOnABreakIsOnTheLineItIsAboutToWriteOn(t *testing.T) {
	rs := []rune("aaa bbb ccc")

	spans := wrapSpans(rs, 4)
	if len(spans) != 3 {
		t.Fatalf("the value was drawn as %d lines, want 3", len(spans))
	}

	if row := spanRow(spans, spans[1].from); row != 1 {
		t.Errorf("the start of the second line is on row %d, want 1", row)
	}

	if row := spanRow(spans, len(rs)); row != 2 {
		t.Errorf("the end of the value is on row %d, want the last line", row)
	}
}

// Which is what a click has to answer: a column of the third line is that
// far into the third line, not that far into the value.
func TestPointingAtALineAnswersWithThePlaceInTheValue(t *testing.T) {
	rs := []rune("aaa bbb ccc")
	spans := wrapSpans(rs, 4)

	if got := spanOffset(spans, 2, 1); got != 9 {
		t.Errorf("the second cell of the third line is offset %d, want 9", got)
	}

	// Past the end of a line is the end of that line. The half of a row
	// that has no text on it still belongs to the row.
	if got := spanOffset(spans, 0, 40); got != spans[0].to {
		t.Errorf("a cell past the first line is offset %d, want its end %d", got, spans[0].to)
	}
}

// The box shows six lines of a task that may have more. Which six is
// decided by where the reader is: the last ones while they are typing at
// the end, and the caret's own once they have walked back up into it.
func TestTheBoxScrollsToWhereTheCaretIs(t *testing.T) {
	if got := spanWindow(3, 6, 2); got != 0 {
		t.Errorf("a value shorter than the box starts at line %d, want 0", got)
	}

	if got := spanWindow(10, 6, 9); got != 4 {
		t.Errorf("a caret on the last of ten lines shows from %d, want 4", got)
	}

	if got := spanWindow(10, 6, 1); got != 1 {
		t.Errorf("a caret on the second of ten lines shows from %d, want 1", got)
	}
}
