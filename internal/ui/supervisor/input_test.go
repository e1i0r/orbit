package supervisor

// The line you answer in, when it is more than one line.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// TestOnlyTheFirstLineCarriesThePrompt.
//
// A message of three lines drew three ❯, which reads as three prompts — three
// separate things somebody is about to send — when it is one. The mark says
// "this is where you are writing", and it is only true once.
func TestOnlyTheFirstLineCarriesThePrompt(t *testing.T) {
	s, e := open(t)
	s.input = "first\nsecond\nthird"

	rows := s.inputLines(90, e)
	if len(rows) < 3 {
		t.Fatalf("three lines drew %d rows", len(rows))
	}

	if n := strings.Count(ansi.Strip(strings.Join(rows, "\n")), "❯"); n != 1 {
		t.Errorf("the typing.Field drew %d prompts for one message:\n%s", n, ansi.Strip(strings.Join(rows, "\n")))
	}
}

// TestTheLinesAfterTheFirstLineUpUnderIt, so the message reads as a block
// rather than stepping back to the margin.
func TestTheLinesAfterTheFirstLineUpUnderIt(t *testing.T) {
	s, e := open(t)
	s.input = "first\nsecond"

	rows := s.inputLines(90, e)

	// Columns and not bytes: the prompt glyph is three bytes wide and one
	// cell, which is the whole reason the terminal is measured this way.
	first, second := ansi.Strip(rows[0]), ansi.Strip(rows[1])

	at := func(row, word string) int {
		before, _, found := strings.Cut(row, word)
		if !found {
			t.Fatalf("%q does not hold %q", row, word)
		}

		return lipgloss.Width(before)
	}

	if at(first, "first") != at(second, "second") {
		t.Errorf("the second line starts at a different column:\n%q\n%q", first, second)
	}
}
