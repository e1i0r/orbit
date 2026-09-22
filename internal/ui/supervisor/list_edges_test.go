package supervisor

// The offers over the line, the box the line is typed in, and the two doors
// that write something down.
//
// The list shows six at a time out of however many there are, and the sums
// that decide which six — and the count of the ones left over — are read by
// a reader deciding which of them to take.

import (
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/spoken"
)

// TestTheListShowsSixAndSaysHowManyMoreThereAre. The window onto the offers
// moves with the pick, and the line under it counts what is outside it: a
// count that is out says a list has more in it than it has, and a reader
// walks the key down looking for something that is not there.
func TestTheListShowsSixAndSaysHowManyMoreThereAre(t *testing.T) {
	s, e := halfWritten(t, "/")

	offers := s.completions(e)
	if len(offers) <= completionRows {
		t.Skipf("the gestures are %d, and this needs more than the %d drawn at once",
			len(offers), completionRows)
	}

	for _, pick := range []int{0, 1, completionRows - 1, completionRows, len(offers) - 1} {
		s.pick = pick

		from := 0
		if pick >= completionRows {
			from = pick - completionRows + 1
		}

		drawn := s.drawCompletions(60, e)

		// Six offers, the count of what is left over, and the blank row
		// that sets the list off from the line.
		if want := completionRows + 2; len(drawn) != want {
			t.Fatalf("with %d picked the list drew %d rows, want %d", pick, len(drawn), want)
		}

		marked := 0

		for row := range completionRows {
			line := ansi.Strip(drawn[row])
			if !strings.Contains(line, offers[from+row].Text) {
				t.Errorf("with %d picked, row %d says %q, want offer %d",
					pick, row, line, from+row)
			}

			if strings.Contains(line, "▸") {
				marked++

				if from+row != pick {
					t.Errorf("with %d picked, the mark is on offer %d", pick, from+row)
				}
			}
		}

		if marked != 1 {
			t.Errorf("with %d picked, %d offers are marked, want one", pick, marked)
		}

		// The count is of what is not on the list, not of what is.
		rest := len(offers) - completionRows
		if got := ansi.Strip(drawn[completionRows]); !strings.Contains(got, strconv.Itoa(rest)) {
			t.Errorf("with %d picked, the list says %q, want %d more", pick, got, rest)
		}
	}
}

// TestAListThatFitsSaysNothingAboutMore. "0 more" under a list with nothing
// left over is a line that says the reader has not seen everything when
// they have.
func TestAListThatFitsSaysNothingAboutMore(t *testing.T) {
	s, e := halfWritten(t, "@ACME-")

	offers := s.completions(e)
	if len(offers) == 0 || len(offers) > completionRows {
		t.Skipf("the mentions are %d, and this needs between one and %d", len(offers), completionRows)
	}

	for _, row := range s.drawCompletions(60, e) {
		if strings.Contains(ansi.Strip(row), "more") {
			t.Errorf("a list of %d in a box that holds %d says %q", len(offers), completionRows, row)
		}
	}
}

// TestAPickPastTheEndTakesTheLastOffer. The list is narrowed by every
// character typed into the word under it, so the offer the keyboard walked
// to can be past the end of the list by the time ⏎ arrives.
func TestAPickPastTheEndTakesTheLastOffer(t *testing.T) {
	s, e := halfWritten(t, "/")

	offers := s.completions(e)
	if len(offers) < 2 {
		t.Fatalf("a slash offered %d gestures", len(offers))
	}

	s.pick = len(offers) + 4

	if got := s.takeCompletion(e).input; got != offers[len(offers)-1].Text+" " {
		t.Errorf("a pick past the end took %q, want the last offer", got)
	}

	// And walking the list comes back round from either end to the other.
	s.pick = len(offers) - 1
	if down := s.moveCompletion(1, e); down.pick != 0 {
		t.Errorf("down past the end left the pick at %d, want the first", down.pick)
	}

	s.pick = 0
	if up := s.moveCompletion(-1, e); up.pick != len(offers)-1 {
		t.Errorf("up past the start left the pick at %d, want the last of %d", up.pick, len(offers))
	}
}

// TestTheLineIsTwoRowsAndStaysInsideItsBox. The box does not resize under
// the first character, and what is typed into it is wrapped against the
// width the box is drawn at less the mark in front of it — a wrap against
// the full width puts the tail of every long line under the mark, where the
// next row of the box is.
func TestTheLineIsTwoRowsAndStaysInsideItsBox(t *testing.T) {
	s, e := open(t)

	if got := len(s.inputLines(60, e)); got != 2 {
		t.Errorf("an empty line is %d rows, want two", got)
	}

	s.input = "corto"

	if got := len(s.inputLines(60, e)); got != 2 {
		t.Errorf("a line with one word in it is %d rows, want two", got)
	}

	s.input = strings.Repeat("una frase que sigue y sigue ", 12)

	for cw := 20; cw <= 120; cw++ {
		rows := s.inputLines(cw, e)
		if len(rows) < 2 {
			t.Fatalf("a long line in a box of %d is %d rows", cw, len(rows))
		}

		for i, r := range rows {
			if got := lipgloss.Width(r); got > cw {
				t.Errorf("row %d of a line in a box of %d is %d cells wide", i, cw, got)
			}
		}
	}
}

// TestARuleStopsAndANoteDoesNot. The two are one gesture apart and they are
// not the same power: one sends work back and the other is told to whoever
// is working. Which of them was written is the flag this door carries.
func TestARuleStopsAndANoteDoesNot(t *testing.T) {
	for _, c := range []struct {
		kind  spoken.Kind
		stops bool
	}{
		{spoken.Rule, true},
		{spoken.Aware, false},
	} {
		var got []bool

		s, e := opened(t, &held{})
		e.Learn = func(stops bool, _, _, _ string) error {
			got = append(got, stops)

			return nil
		}

		line := spoken.Line{Kind: c.kind, Phrase: "never log a card number"}

		next, out := s.learn(line, e)

		if len(got) != 1 || got[0] != c.stops {
			t.Fatalf("%v was written down as stops=%v, want %v", c.kind, got, c.stops)
		}

		if out.Said == "" {
			t.Errorf("%v was written down and the screen said nothing", c.kind)
		}

		if next.input != s.input {
			t.Errorf("%v left the line as %q", c.kind, next.input)
		}
	}
}

// TestAThreadCacheThatIsNotThereIsNotThrownAway. The cache is a pointer the
// window may not have made yet, and every read of the record throws it away
// — so the one that runs before the first frame is a read against nothing.
func TestAThreadCacheThatIsNotThereIsNotThrownAway(t *testing.T) {
	var c *threadCache

	c.invalidate()
}
