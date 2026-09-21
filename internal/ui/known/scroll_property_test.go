package known

// The arithmetic of a window on a list, swept rather than sampled.
//
// Every one of these is an off-by-one waiting to happen: a row half on
// screen, a head that scrolled away, a list parked one line past its end. A
// reader meets them at one particular height with one particular list, which
// is why picking three sizes by hand finds none of them — so these walk the
// whole small space instead.

import (
	"strconv"
	"strings"
	"testing"
)

// consecutive is whether these lines are one unbroken stretch of a body.
func consecutive(shown []string) bool {
	for i := 1; i < len(shown); i++ {
		was, err := strconv.Atoi(strings.TrimPrefix(shown[i-1], "line "))
		if err != nil {
			return false
		}

		now, err := strconv.Atoi(strings.TrimPrefix(shown[i], "line "))
		if err != nil || now != was+1 {
			return false
		}
	}

	return true
}

// lines is a body of n lines, each saying which one it is.
func lines(n int) []string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, "line "+strconv.Itoa(i))
	}

	return out
}

// TestAFramedScreenKeepsItsHeadAndItsFoot.
//
// The head says what is being looked at and the foot what the keys do, and
// both are needed most by the reader who has scrolled furthest — which is
// exactly the reader who would have lost them.
func TestAFramedScreenKeepsItsHeadAndItsFoot(t *testing.T) {
	head, foot := []string{"HEAD"}, []string{"FOOT"}

	for h := range 12 {
		for n := range 12 {
			for off := -2; off <= n+2; off++ {
				got := framed(head, lines(n), foot, h, off)

				if len(got) == 0 {
					t.Fatalf("h=%d n=%d off=%d drew nothing", h, n, off)
				}

				if got[0] != "HEAD" {
					t.Errorf("h=%d n=%d off=%d: the screen opens with %q", h, n, off, got[0])
				}

				if last := got[len(got)-1]; last != "FOOT" {
					t.Errorf("h=%d n=%d off=%d: the screen ends with %q", h, n, off, last)
				}

				// A screen tall enough for all three parts is exactly as
				// tall as it was asked to be. One line more and it writes
				// over whatever is under it; one less and there is a gap.
				if h >= len(head)+len(foot)+1 && len(got) != h {
					t.Errorf("h=%d n=%d off=%d drew %d lines", h, n, off, len(got))
				}

				// What is between them came out of the body, in the order
				// the body has them and with none skipped: a window that
				// jumped a line is a rule a reader never sees.
				var shown []string

				for _, line := range got[1 : len(got)-1] {
					if line != "" {
						shown = append(shown, line)
					}
				}

				if !consecutive(shown) {
					t.Fatalf("h=%d n=%d off=%d showed %v, which is not one stretch", h, n, off, shown)
				}

				if n == 0 {
					continue
				}

				// A screen with room for a body line always shows one: a
				// list that is there and a window showing none of it is the
				// screen of blank rows this scrolling exists to prevent.
				if h >= len(head)+len(foot)+1 && len(shown) == 0 {
					t.Fatalf("h=%d n=%d off=%d showed none of a list of %d", h, n, off, n)
				}

				if len(shown) == 0 {
					continue
				}

				// A reader at the top sees the first line, and one who
				// scrolled past the end sees the last. Between them the
				// window never runs out of list.
				if off <= 0 && shown[0] != "line 0" {
					t.Errorf("h=%d n=%d off=%d opens at %q, not the top of the list", h, n, off, shown[0])
				}

				if off >= n && shown[len(shown)-1] != "line "+strconv.Itoa(n-1) {
					t.Errorf("h=%d n=%d off=%d ends at %q, not the bottom of the list",
						h, n, off, shown[len(shown)-1])
				}
			}
		}
	}
}

// TestAWindowNeverParksPastTheEndOfItsList.
//
// The list gets shorter on its own: a sentence kept leaves the tray, a rule
// turned off shortens no row but a repository emptied loses a heading. A
// view parked past the end of a list that shrank is a screen of blank rows
// and no way to tell why.
func TestAWindowNeverParksPastTheEndOfItsList(t *testing.T) {
	for lines := range 12 {
		for view := 1; view <= 12; view++ {
			for off := -3; off <= lines+3; off++ {
				got := State{offset: off}.window(lines, view)

				if got < 0 {
					t.Errorf("lines=%d view=%d off=%d: the window opens at %d", lines, view, off, got)
				}

				if most := max(0, lines-view); got > most {
					t.Errorf("lines=%d view=%d off=%d: the window opens at %d, past %d",
						lines, view, off, got, most)
				}

				// A list that fits is shown from its top: there is nothing
				// to scroll to.
				if lines <= view && got != 0 {
					t.Errorf("lines=%d view=%d off=%d: a list that fits opens at %d",
						lines, view, off, got)
				}
			}
		}
	}
}

// TestBringingAStretchOnScreenMovesAsLittleAsItCan.
//
// It is what keeps the row being typed into visible as a form is walked, and
// what the arrow keys move on a rule's own screen. What it must never do is
// answer a window that does not hold the stretch it was asked about, when
// there was room for it.
func TestBringingAStretchOnScreenMovesAsLittleAsItCan(t *testing.T) {
	for total := range 10 {
		for room := 1; room <= 6; room++ {
			for from := range total + 1 {
				for rows := 1; rows <= 4; rows++ {
					for off := -2; off <= total+room+2; off++ {
						got := deepEnough(off, from, rows, room, total)

						if got < 0 || got > max(0, total-room) {
							t.Fatalf("total=%d room=%d from=%d rows=%d off=%d answered %d",
								total, room, from, rows, off, got)
						}

						// A stretch that fits, and sits inside the list, is
						// on screen afterwards.
						if rows <= room && from+rows <= total {
							if from < got || from+rows > got+room {
								t.Errorf("total=%d room=%d from=%d rows=%d off=%d: %d..%d is not inside %d..%d",
									total, room, from, rows, off, from, from+rows, got, got+room)
							}
						}
					}
				}
			}
		}
	}
}

// TestARowHoldsItsOwnLinesAndNobodyElses.
//
// A rule is as tall as its sentence wraps, so neither the scrolling nor a
// click can assume one row is one line — and a row that claimed the line
// under it would put the cursor on its neighbour when somebody clicked the
// gap.
func TestARowHoldsItsOwnLinesAndNobodyElses(t *testing.T) {
	for from := range 8 {
		for rows := range 5 {
			sp := span{from: from, rows: rows}

			// A row of no rows is still one line: it was drawn, so it is
			// somewhere.
			tall := max(rows, 1)

			if sp.last() != from+tall-1 {
				t.Errorf("a row at %d of %d rows ends at %d", from, rows, sp.last())
			}

			if !sp.holds(from) {
				t.Errorf("a row at %d of %d rows does not hold its own first line", from, rows)
			}

			if !sp.holds(sp.last()) {
				t.Errorf("a row at %d of %d rows does not hold its own last line", from, rows)
			}

			if sp.holds(from - 1) {
				t.Errorf("a row at %d of %d rows holds the line above it", from, rows)
			}

			if sp.holds(sp.last() + 1) {
				t.Errorf("a row at %d of %d rows holds the line below it", from, rows)
			}

			for line := from; line <= sp.last(); line++ {
				if !sp.holds(line) {
					t.Errorf("a row at %d of %d rows does not hold line %d", from, rows, line)
				}
			}
		}
	}
}

// TestHowWideASentenceIsDrawn, which is the window less its margins and
// capped so that a sentence does not stretch across a monitor — and floored
// so that a narrow terminal still draws something to read.
func TestHowWideASentenceIsDrawn(t *testing.T) {
	last := 0

	for w := range 200 {
		got := content(w)

		if got < 24 {
			t.Errorf("a window of %d columns draws %d wide, which is narrower than anything readable", w, got)
		}

		if got > 110 {
			t.Errorf("a window of %d columns draws %d wide, stretching a sentence across a monitor", w, got)
		}

		if got < last {
			t.Errorf("a window of %d columns draws %d wide, narrower than the %d before it", w, got, last)
		}

		last = got
	}

	// The margins are four columns, and the two ends are where the floor
	// and the cap take over. A reader on a hundred-column terminal reads a
	// sentence ninety-six wide, and moving that by four moves every
	// wrapped rule on the screen.
	for _, one := range []struct{ window, drawn int }{
		{0, 24}, {20, 24}, {28, 24}, {29, 25}, {100, 96}, {114, 110}, {200, 110},
	} {
		if got := content(one.window); got != one.drawn {
			t.Errorf("a window of %d columns draws %d wide, want %d", one.window, got, one.drawn)
		}
	}
}
