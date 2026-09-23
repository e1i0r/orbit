package settings

// The arithmetic the table is drawn from, at every height the window has.
//
// The screen was read at one height with the cursor in the middle of it,
// and every number under it — the room the table is given, the line it
// starts at, the rounding onto a row — has two ends nothing was asking
// about.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/layout"
)

// TestTheTableHasRoomForItsRowsAtEveryHeight. The room the table is given
// is the body less the head and the foot, and a sum that is one out draws a
// row the screen has no line for — or leaves a line the table could have
// used blank.
func TestTheTableHasRoomForItsRowsAtEveryHeight(t *testing.T) {
	for _, c := range []struct {
		h, want int
	}{
		{0, 1},
		{1, 1},
		{headLines + footLines, 1},
		{headLines + footLines + 1, 1},
		{headLines + footLines + 2, 2},
		{20, 15},
		{60, 55},
	} {
		if got := room(c.h); got != c.want {
			t.Errorf("a body of %d lines leaves %d for the table, want %d", c.h, got, c.want)
		}
	}
}

// TestAWindowWithNoBodyStartsTheTableAtTheTop. Every window passes through
// no height at all while somebody drags its corner, and a scroll computed
// against a height the table will not be drawn at is a table parked past its
// own end.
func TestAWindowWithNoBodyStartsTheTableAtTheTop(t *testing.T) {
	e := short(t, env(t, newFile()))

	s := Open(e)
	for range 8 {
		s, _ = s.Key(down(), e)
	}

	if s.Off(e) == 0 {
		t.Fatal("this test needs a table that has been scrolled")
	}

	none := e
	none.Frame = layout.Frame{Body: layout.Strip{Y: 3, H: 0, W: 100}}

	if got := s.Off(none); got != 0 {
		t.Errorf("a window with no body starts the table at line %d", got)
	}

	negative := e
	negative.Frame = layout.Frame{Body: layout.Strip{Y: 3, H: -2, W: 100}}

	if got := s.Off(negative); got != 0 {
		t.Errorf("a window of no height at all starts the table at line %d", got)
	}
}

// TestTheRowTheCursorIsOnIsOnTheScreen, at every height the table can be
// drawn at and on every row it has. The scroll is two sums and a rounding,
// and each of them is a place the row the reader is moving to can end up a
// line below the last one drawn — which is a cursor that walks off the
// bottom and leaves no mark of having gone there.
func TestTheRowTheCursorIsOnIsOnTheScreen(t *testing.T) {
	base := env(t, newFile())

	for h := 1; h <= 40; h++ {
		e := base
		e.Frame = layout.Frame{Body: layout.Strip{Y: 3, H: h, W: 100}}

		s := Open(e)
		rows := s.Rows(e)

		for i := range rows {
			if i > 0 {
				s, _ = s.Key(down(), e)
			}

			off, view := s.Off(e), room(h)
			from, last := s.Chosen()*rowLines, s.Chosen()*rowLines+1

			// A dial is its name and the sentence under it. Wherever
			// there is room for the two, both are on the screen;
			// where there is room for one line only, one of them is.
			shown := func(line int) bool { return line >= off && line <= off+view-1 }

			if view > 1 && (!shown(from) || !shown(last)) {
				t.Fatalf("at a body of %d the cursor's row is lines %d and %d, outside the %d from %d",
					h, from, last, view, off)
			}

			if view == 1 && !shown(from) && !shown(last) {
				t.Fatalf("at a body of %d neither line of the cursor's row is the one line drawn, from %d",
					h, off)
			}

			// And the table never scrolls past its own end.
			if last := rowLines * len(rows); off > max(0, last-view) {
				t.Fatalf("at a body of %d the table starts at line %d of %d", h, off, last)
			}
		}
	}
}

// TestTheScreenIsExactlyAsTallAsTheBody, whatever is in it. A screen a line
// short leaves the row underneath it showing through, and one a line over
// pushes the foot off the window.
func TestTheScreenIsExactlyAsTallAsTheBody(t *testing.T) {
	base := env(t, newFile())

	// Past the height of the whole table too: a window taller than what is
	// in it is a window drawing rows the table has none of.
	for h := 1; h <= 80; h++ {
		e := base
		e.Frame = layout.Frame{Body: layout.Strip{Y: 3, H: h, W: 100}}

		if got := len(Open(e).View(h, 100, e)); got != h {
			t.Errorf("a body of %d lines drew %d", h, got)
		}
	}
}

// TestTheTopOfTheViewIsTheTopOfARow. A view that lands mid-row draws that
// row's blank separator under the title, which reads as a gap somebody left
// there and spends a line of the table on nothing — except on a screen with
// less room than a row, where rounding up would push the row the cursor is
// on off the top of the view it was brought into.
func TestTheTopOfTheViewIsTheTopOfARow(t *testing.T) {
	for _, c := range []struct {
		off, view, want int
	}{
		{1, rowLines, rowLines},
		{2, rowLines, rowLines},
		{3, rowLines, rowLines},
		{4, rowLines + 1, 2 * rowLines},
		{0, rowLines, 0},

		// A view with less room than a row keeps the line it was given.
		{1, rowLines - 1, 1},
		{2, 1, 2},
	} {
		rows := make([]Row, 5)
		starts, _ := placed(rows)

		if got := atTheTop(c.off, c.view, rows, starts); got != c.want {
			t.Errorf("a view of %d lines starting at %d starts at %d, want %d",
				c.view, c.off, got, c.want)
		}
	}
}

// TestAKeyPressedAtNoHeightLeavesTheTableAtTheTop. The height a keystroke
// scrolls against is not always the height the next frame is drawn at: a
// window being dragged passes through none, and a table scrolled against
// that is parked somewhere the next frame has to undo.
func TestAKeyPressedAtNoHeightLeavesTheTableAtTheTop(t *testing.T) {
	e := env(t, newFile())
	e.Frame = layout.Frame{Body: layout.Strip{Y: 3, H: 0, W: 100}}

	s := Open(e)
	for range 8 {
		s, _ = s.Key(down(), e)
	}

	if s.off != 0 {
		t.Errorf("eight rows walked in a window with no body left the table at line %d", s.off)
	}
}

// TestOneRowOnTheScreenIsTheChosenOne. The mark is what says which row a
// keystroke acts on, so a screen drawing it beside every row but one is a
// reader turning the dial they are not looking at.
func TestOneRowOnTheScreenIsTheChosenOne(t *testing.T) {
	e := env(t, newFile())
	e.Frame = layout.Frame{Body: layout.Strip{Y: 3, H: 80, W: 100}}

	s := Open(e)
	s, _ = s.Key(down(), e)

	rows := s.Rows(e)
	chosen := rows[s.Chosen()]

	marked := 0

	for _, l := range s.View(80, 100, e) {
		line := ansi.Strip(l)
		if !strings.Contains(line, "▸") {
			continue
		}

		marked++

		if !strings.Contains(line, chosen.Key) {
			t.Errorf("the mark is beside %q and the cursor is on %q", line, chosen.Key)
		}
	}

	if marked != 1 {
		t.Errorf("%d rows are marked as the chosen one, want one", marked)
	}
}

// TestTheDotIsOnTheOptionTheRowHolds. A row of pills says what a setting is
// by marking one of them, so a mark on the option beside it is a screen
// reporting a value the file does not hold — and the reader turns the dial
// to a place it is already at.
func TestTheDotIsOnTheOptionTheRowHolds(t *testing.T) {
	f := newFile()
	f.held["unread-cap"] = "5" // one of 0, 3, 5, 10, 20

	e := env(t, f)
	e.Frame = layout.Frame{Body: layout.Strip{Y: 3, H: 80, W: 100}}

	s := Open(e)
	s.sel = rowOf(t, s, e, "unread-cap")

	for _, l := range s.View(80, 100, e) {
		line := ansi.Strip(l)
		if !strings.Contains(line, "unread-cap") {
			continue
		}

		if !strings.Contains(line, "● 5 ") {
			t.Errorf("the row holding 5 marks nothing as 5: %q", line)
		}

		for _, other := range []string{"● 0 ", "● 3 ", "● 10 ", "● 20 "} {
			if strings.Contains(line, other) {
				t.Errorf("the row holding 5 also marks %q: %q", strings.TrimSpace(other), line)
			}
		}
	}
}

// TestACursorLeftPastTheEndOfATableThatShrankParksAtTheTop. The model and
// effort dials belong to the engine, so choosing another engine can take
// rows out from under a cursor that is already below them — and a scroll
// computed from a row that is no longer there is a table parked at a line
// with nothing on it.
func TestACursorLeftPastTheEndOfATableThatShrankParksAtTheTop(t *testing.T) {
	e := short(t, env(t, newFile()))

	s := Open(e)
	for range 8 {
		s, _ = s.Key(down(), e)
	}

	if s.off == 0 {
		t.Fatal("this test needs a table that has been scrolled")
	}

	s.sel = len(s.Rows(e))

	if got := s.keepSeen(e).off; got != 0 {
		t.Errorf("a cursor one past the last row left the table at line %d, want the top", got)
	}
}
