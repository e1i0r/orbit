package settings

// The table is taller than the screen, and this is the suite that says so.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/layout"
)

// short is a window whose body is too small for the whole table: nine dials
// at three lines each, against a body of fifteen.
func short(t *testing.T, e Env) Env {
	t.Helper()

	e.Frame = layout.Frame{Body: layout.Strip{Y: 3, H: 15, W: 100}}

	return e
}

// down is the down arrow, which is how the cursor walks the table.
func down() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyDown} }

// drawn is the screen as one string, for a test asking whether a name is on
// it at all.
func drawn(s State, e Env) string {
	return strings.Join(s.View(e.Frame.Body.H, e.Frame.Body.W, e), "\n")
}

// TestTheLastSettingCanBeSeen. The table outgrew the screen without anybody
// deciding it should, and the dials past the bottom were reachable by a
// cursor that left no mark of having gone there.
func TestTheLastSettingCanBeSeen(t *testing.T) {
	e := short(t, env(t, newFile()))

	s := Open(e)
	if on := drawn(s, e); strings.Contains(on, "theme") {
		t.Fatal("the last setting is on a screen this test needs it to be off")
	}

	rows := s.Rows(e)
	for range len(rows) - 1 {
		s, _ = s.Key(down(), e)
	}

	if s.Chosen() != len(rows)-1 {
		t.Fatalf("the cursor is on row %d, want the last of %d", s.Chosen(), len(rows))
	}

	if on := drawn(s, e); !strings.Contains(on, rows[len(rows)-1].Key) {
		t.Errorf("the cursor is on %q and the screen does not draw it", rows[len(rows)-1].Key)
	}
}

// TestTheTitleStaysWhileTheTableScrolls. The head says what is being looked
// at and the foot says how to leave, and both are wanted most by the reader
// who has scrolled furthest.
func TestTheTitleStaysWhileTheTableScrolls(t *testing.T) {
	e := short(t, env(t, newFile()))

	s := Open(e)
	for range len(s.Rows(e)) - 1 {
		s, _ = s.Key(down(), e)
	}

	on := drawn(s, e)
	for _, want := range []string{"Settings", "edit"} {
		if !strings.Contains(on, want) {
			t.Errorf("a scrolled screen lost %q", want)
		}
	}

	if got := len(s.View(e.Frame.Body.H, e.Frame.Body.W, e)); got != e.Frame.Body.H {
		t.Errorf("a scrolled screen drew %d rows in %d", got, e.Frame.Body.H)
	}
}

// TestGoingBackToTheTopBringsTheTableWithIt. The cursor wraps at both ends,
// and a view left at the bottom would draw a table the cursor has left.
func TestGoingBackToTheTopBringsTheTableWithIt(t *testing.T) {
	e := short(t, env(t, newFile()))

	s := Open(e)
	for range len(s.Rows(e)) {
		s, _ = s.Key(down(), e)
	}

	if s.Chosen() != 0 {
		t.Fatalf("the cursor is on row %d after wrapping, want the first", s.Chosen())
	}

	if s.Off(e) != 0 {
		t.Errorf("the table is %d lines down with the cursor at the top", s.Off(e))
	}
}

// TestTheWheelMovesOneSettingANotch. A setting is three lines tall, so a
// notch counted in lines would crawl through a single dial.
func TestTheWheelMovesOneSettingANotch(t *testing.T) {
	e := short(t, env(t, newFile()))

	s := Open(e).Scroll(1, e)
	if s.Chosen() != 1 {
		t.Errorf("one notch moved the cursor to row %d, want the second", s.Chosen())
	}

	// The wheel does not wrap. A list that came back to the top under a
	// wheel still being pushed downwards is one nobody can hold still.
	rows := len(s.Rows(e))
	for range rows * 2 {
		s = s.Scroll(1, e)
	}

	if s.Chosen() != rows-1 {
		t.Errorf("the wheel ran past the end to row %d of %d", s.Chosen(), rows)
	}

	for range rows * 2 {
		s = s.Scroll(-1, e)
	}

	if s.Chosen() != 0 || s.Off(e) != 0 {
		t.Errorf("the wheel ran past the top: row %d, %d lines down", s.Chosen(), s.Off(e))
	}
}

// TestATallScreenNeverScrolls. Everything here answers zero when the table
// fits, which is what keeps the mouse's arithmetic right on a big terminal.
func TestATallScreenNeverScrolls(t *testing.T) {
	e := env(t, newFile())
	e.Frame = layout.Frame{Body: layout.Strip{Y: 3, H: 60, W: 100}}

	s := Open(e)
	for range len(s.Rows(e)) - 1 {
		s, _ = s.Key(down(), e)
	}

	if s.Off(e) != 0 {
		t.Errorf("a table that fits scrolled %d lines", s.Off(e))
	}
}

// TestTheViewIsHeldInsideATableThatShrank. The model and effort dials are
// the engine's, so choosing another engine can leave the view parked past
// the end — which draws as blank rows with nothing saying why.
func TestTheViewIsHeldInsideATableThatShrank(t *testing.T) {
	e := short(t, env(t, newFile()))

	s := Open(e)
	for range len(s.Rows(e)) - 1 {
		s, _ = s.Key(down(), e)
	}

	deep := s.Off(e)
	if deep == 0 {
		t.Fatal("the table never scrolled, so there is nothing to hold")
	}

	// Half the table, and the same screen: the offset it was left at is
	// past the end of what there now is.
	e.Engines = func() []string { return []string{"zeta"} }
	e.Flows = func() []string { return []string{"cover"} }

	small := e
	small.Kept = nil

	if held := s.Off(small); held != 0 {
		t.Errorf("an empty table is %d lines down, want the top", held)
	}
}
