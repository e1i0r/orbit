package menu

// The menu drawn, the list following its cursor, and what a click lands on.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/point"
)

// TestEachMenuSaysWhichOneItIs. The two open on the same keystroke and
// answer different questions, and a reader who meant the task's and got the
// board's should be able to see that from the menu rather than from a verb
// that is not in it.
func TestEachMenuSaysWhichOneItIs(t *testing.T) {
	e := world(t)

	board := Open("", e).Title(e)
	if !strings.Contains(board, "one task") {
		t.Errorf("the board's menu is titled %q, want it to say it is about no one task", board)
	}

	task := Open(theTask, e).Title(e)
	if !strings.Contains(task, theTask) {
		t.Errorf("a task's menu is titled %q, want it to name the task", task)
	}

	// And the title is drawn, not merely computed.
	rows := strings.Join(Open("", e).View(20, 100, e), "\n")
	if !strings.Contains(rows, board) {
		t.Errorf("the menu drew %q, want the title in it", rows)
	}
}

// TestAMenuWithNoRoomDrawsNothing.
func TestAMenuWithNoRoomDrawsNothing(t *testing.T) {
	e := world(t)

	if rows := Open("", e).View(0, 40, e); rows != nil {
		t.Errorf("a menu with no rows drew %v, want nothing", rows)
	}
}

// TestAMenuWhoseTaskLeftTheBoardSaysSo, rather than drawing an empty list a
// reader would read as a task with nothing that can be done to it.
func TestAMenuWhoseTaskLeftTheBoardSaysSo(t *testing.T) {
	e := world(t)

	rows := Open(gone, e).View(6, 60, e)
	if !strings.Contains(ansi.Strip(strings.Join(rows, "\n")), "no longer on the board") {
		t.Errorf("the menu of a task that left drew %v, want it to say the task is gone", rows)
	}
}

// TestTheMenuFillsTheRoomItIsGiven, title first and the entries under it.
func TestTheMenuFillsTheRoomItIsGiven(t *testing.T) {
	e := world(t)

	drawn := Open("", e).View(10, 60, e)
	if len(drawn) != 10 {
		t.Fatalf("the menu drew %d lines in a body of 10, want 10", len(drawn))
	}

	if !strings.Contains(ansi.Strip(drawn[0]), "about one task") {
		t.Errorf("the first row is %q, want the title of the menu", drawn[0])
	}

	if !strings.Contains(ansi.Strip(drawn[TitleRows]), "reconcile") {
		t.Errorf("the first entry is drawn at %q, want the first command", drawn[TitleRows])
	}
}

// TestTheHeadingsAreDrawnWhereTheEntriesAreCounted. The menu is hit-tested
// by counting rows from the title, so a heading that drew two lines would
// put every click below it on the wrong verb.
func TestTheHeadingsAreDrawnWhereTheEntriesAreCounted(t *testing.T) {
	e := inside(t)

	s := Open(theTask, e)
	rows := s.View(e.Frame.Body.H, e.Frame.Body.W, e)

	for i, entry := range s.Entries(e) {
		if entry.Head || entry.Title == "" {
			continue
		}

		row := i + TitleRows
		if row >= len(rows) {
			break
		}

		if !strings.Contains(ansi.Strip(rows[row]), entry.Title) {
			t.Fatalf("entry %d (%q) is not on the row it is counted at: %q", i, entry.Title, rows[row])
		}
	}
}

// TestAHeadingIsNotSomethingToClickOn.
func TestAHeadingIsNotSomethingToClickOn(t *testing.T) {
	e := inside(t)

	s := Open(theTask, e)

	head := -1

	for i, entry := range s.Entries(e) {
		if entry.Head && entry.Title != "" {
			head = i
			break
		}
	}

	if head < 0 {
		t.Fatal("the menu has no heading to click on")
	}

	if got := s.Hit(cells.Gutter, e.Frame.Body.Y+TitleRows+head, e); got.Kind != point.None {
		t.Errorf("clicking the heading answered %+v, want nothing", got)
	}
}

// TestAClickPastTheListIsNothing: the rows above the entries and the ones
// past the end of them both.
func TestAClickPastTheListIsNothing(t *testing.T) {
	e := world(t)

	s := Open("", e)

	if got := s.Hit(0, 9999, e); got.Kind != point.None {
		t.Errorf("a click far past the body answered %+v, want nothing", got)
	}

	if got := s.Hit(0, e.Frame.Body.Y, e); got.Kind != point.None {
		t.Errorf("a click on the title answered %+v, want nothing", got)
	}

	if got := s.Hit(0, e.Frame.Body.Y+TitleRows+len(s.Entries(e)), e); got.Kind != point.None {
		t.Errorf("a click below the last entry answered %+v, want nothing", got)
	}
}

// TestAClickIsKeyedByWhatIdentifiesTheEntry — the name for a command, the
// glyph for a verb — because the list is recomputed between press and
// release and an index could point at a different row by then.
func TestAClickIsKeyedByWhatIdentifiesTheEntry(t *testing.T) {
	e := world(t)

	got := Open("", e).Hit(0, e.Frame.Body.Y+TitleRows, e)
	if got.Kind != point.MenuEntry || got.Key != "reconcile" {
		t.Errorf("clicking the board's first row answered %+v, want the command it names", got)
	}

	onTask := Open(theTask, e).Hit(0, e.Frame.Body.Y+TitleRows, e)
	if onTask.Kind != point.MenuEntry || onTask.Key != "task" {
		t.Errorf("clicking a task's first row answered %+v, want the family it drills into", onTask)
	}
}

// TestTheMenuFollowsItsCursorDownAndBackUp: a task's panes and verbs
// together are more rows than a small window has, and a verb the reader
// cannot see is a verb they do not have.
func TestTheMenuFollowsItsCursorDownAndBackUp(t *testing.T) {
	e := inside(t)

	s := Open(theTask, e)

	n := len(s.Entries(e))
	for range n {
		s = s.Wheel(1, e)
	}

	rows := view(e.Frame.Body.H)
	if off := s.offsetIn(n, rows); s.At() < off || s.At() >= off+rows {
		t.Errorf("the cursor is at %d and the window shows %d..%d", s.At(), off, off+rows)
	}

	drawn := strings.Join(s.View(e.Frame.Body.H, e.Frame.Body.W, e), "\n")
	if want := s.Entries(e)[s.At()].Title; !strings.Contains(ansi.Strip(drawn), want) {
		t.Errorf("the chosen entry %q is not on the screen it is chosen on", want)
	}

	for range n {
		s = s.Wheel(-1, e)
	}

	if s.offset != 0 {
		t.Errorf("the list came back to the top at offset %d, want 0", s.offset)
	}
}

// TestAnEntryIsWhereItWasDrawnAfterScrolling, or a click lands on the row
// the reader was not looking at.
func TestAnEntryIsWhereItWasDrawnAfterScrolling(t *testing.T) {
	e := inside(t)
	e.Frame.Body.H = 8

	top := Open(theTask, e)
	s, _ := top.Point(index(t, top, e, "task")).Enter(e)

	for range s.Entries(e) {
		s = s.Wheel(1, e)
	}

	if s.offset == 0 {
		t.Fatal("the whole menu fits in this window, so there is nothing to scroll")
	}

	rows := s.View(e.Frame.Body.H, e.Frame.Body.W, e)

	at := s.At() - s.offset + TitleRows
	if !strings.Contains(ansi.Strip(rows[at]), s.Entries(e)[s.At()].Title) {
		t.Fatalf("row %d is %q, want the chosen entry", at, rows[at])
	}

	got := s.Hit(cells.Gutter, e.Frame.Body.Y+at, e)
	if want := ident(s.Entries(e)[s.At()]); got.Key != want {
		t.Errorf("clicking the chosen row answered %+v, want the entry drawn there (%q)", got, want)
	}
}

// TestAHeadingIsShownWithTheEntryUnderIt when the list scrolls back to one:
// "pause" at the top of the screen does not say whether it is a pane or a
// verb.
func TestAHeadingIsShownWithTheEntryUnderIt(t *testing.T) {
	e := inside(t)

	s := Open(theTask, e)
	for range s.Entries(e) {
		s = s.Wheel(1, e)
	}

	// Back up to the first entry under the second heading, which is the
	// one the list has to scroll to reach.
	es := s.Entries(e)

	want := -1

	for i, entry := range es {
		if i > 0 && es[i-1].Head && entry.Head && i+1 < len(es) {
			want = i + 1
		}
	}

	if want < 0 {
		t.Fatal("the menu has no second block to scroll back to")
	}

	s = s.Point(want).keepSeen(e)
	if s.offset > want-1 {
		t.Errorf("the list scrolled to %d with the entry at %d, want its heading shown above it", s.offset, want)
	}
}

// TestTheDescriptionsStartUnderTheSameDot. Names are different lengths —
// "skip this phase" against "ask" — and a dot per row at its own column
// reads as noise rather than a table.
func TestTheDescriptionsStartUnderTheSameDot(t *testing.T) {
	e := world(t)

	rows := Open(theTask, e).View(30, 100, e)

	var dots []int

	for _, r := range rows {
		line := ansi.Strip(r)
		if !strings.Contains(line, "·") {
			continue
		}

		// In runes, not bytes: the cursor's mark is multibyte, and a
		// byte index would read the selected row two columns to the
		// right of where it is drawn.
		before, _, _ := strings.Cut(line, "·")
		dots = append(dots, len([]rune(before)))
	}

	if len(dots) < 2 {
		t.Fatalf("the menu drew %d dotted rows, want at least two to align", len(dots))
	}

	for _, at := range dots[1:] {
		if at != dots[0] {
			t.Errorf("a description starts at column %d, want every dot under %d", at, dots[0])
		}
	}
}

// TestTheSelectedRowReadsWhole. The cursor's paint used to sit on top of
// the inner ones, whose dark ink on the selection ground read as a blank
// bar with the cursor on it.
func TestTheSelectedRowReadsWhole(t *testing.T) {
	e := world(t)

	s := Open(theTask, e).Point(1)
	rows := s.View(30, 100, e)

	got := ansi.Strip(rows[1+TitleRows])
	want := s.Entries(e)[1].Title

	if !strings.Contains(got, want) {
		t.Errorf("the selected row reads %q, want it to carry %q", got, want)
	}
}
