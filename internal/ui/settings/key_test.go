package settings

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestTheKeysMoveAndWrap. The cursor walks the table and comes round, which
// is what a list with no scrollbar has to do to reach its last row.
func TestTheKeysMoveAndWrap(t *testing.T) {
	e := env(t, newFile())
	s := Open(e)
	rows := len(s.Rows(e))

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyUp}, e)
	if s.Chosen() != rows-1 {
		t.Errorf("up from the first row went to %d, want the last (%d)", s.Chosen(), rows-1)
	}

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyDown}, e)
	if s.Chosen() != 0 {
		t.Errorf("down from the last row went to %d, want the first", s.Chosen())
	}
}

// TestTypingIntoARow is the way a value no dial holds is set: a number, or a
// model this build has never heard of.
func TestTypingIntoARow(t *testing.T) {
	f := newFile()
	e := env(t, f)

	s := Open(e).Point(2, e) // unread-cap

	s, _ = s.Key(tea.KeyPressMsg{Code: 'e', Text: "e"}, e)
	if !s.Editing() {
		t.Fatal("e did not open the line")
	}

	if s.Typed() != "3" {
		t.Errorf("the line opened holding %q, want what the row already says", s.Typed())
	}

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyBackspace}, e)
	s, _ = s.Key(tea.KeyPressMsg{Code: '7', Text: "7"}, e)

	s, out := s.Key(tea.KeyPressMsg{Code: tea.KeyEnter}, e)
	if s.Editing() {
		t.Error("saving left the line open")
	}

	if f.held["unread-cap"] != "7" {
		t.Errorf("the file holds a cap of %q, want 7", f.held["unread-cap"])
	}

	if out.Said == "" {
		t.Error("a write said nothing to the band")
	}
}

// TestALineAbandonedChangesNothing.
func TestALineAbandonedChangesNothing(t *testing.T) {
	f := newFile()
	e := env(t, f)

	s := Open(e).Point(2, e).Edit("999")

	s, out := s.Key(tea.KeyPressMsg{Code: tea.KeyEscape}, e)
	if s.Editing() || out.Said != "" {
		t.Error("esc out of a line said something or left it open")
	}

	if f.held["unread-cap"] != "3" {
		t.Errorf("the file holds %q, want what it held before the line was opened", f.held["unread-cap"])
	}
}

// TestEveryWayToTurnADial. Space, enter, the arrows and the vim letters all
// reach the same dial, because a reader's hands are already somewhere.
func TestEveryWayToTurnADial(t *testing.T) {
	for _, k := range []tea.KeyPressMsg{
		{Code: tea.KeyRight},
		{Code: 'l', Text: "l"},
		{Code: tea.KeySpace, Text: " "},
		{Code: tea.KeyLeft},
		{Code: 'h', Text: "h"},
	} {
		f := newFile()
		e := env(t, f)

		if _, out := Open(e).Point(1, e).Key(k, e); out.Said == "" {
			t.Errorf("%v turned nothing", k)
		}
	}
}

// TestTheVimLettersMoveToo.
func TestTheVimLettersMoveToo(t *testing.T) {
	e := env(t, newFile())

	s, _ := Open(e).Key(tea.KeyPressMsg{Code: 'j', Text: "j"}, e)
	if s.Chosen() != 1 {
		t.Errorf("j went to row %d", s.Chosen())
	}

	s, _ = s.Key(tea.KeyPressMsg{Code: 'k', Text: "k"}, e)
	if s.Chosen() != 0 {
		t.Errorf("k went back to row %d", s.Chosen())
	}
}

// TestQuitLeavesTheScreenAndNotTheWindow. q closes what is on top, which is
// what it does on every screen that is not the board.
func TestQuitLeavesTheScreenAndNotTheWindow(t *testing.T) {
	e := env(t, newFile())

	if _, out := Open(e).Key(tea.KeyPressMsg{Code: 'q', Text: "q"}, e); !out.Close {
		t.Error("q did not close the screen")
	}
}

// TestAKeyThatMeansNothingHereDoesNothing, rather than reaching the board
// underneath it.
func TestAKeyThatMeansNothingHereDoesNothing(t *testing.T) {
	e := env(t, newFile())

	s, out := Open(e).Key(tea.KeyPressMsg{Code: 'z', Text: "z"}, e)
	if out.Said != "" || out.Close || out.Dials != nil || s.Chosen() != 0 || s.Editing() {
		t.Errorf("z did something: %+v", out)
	}
}

// TestALineOnARowThatIsNotThereCloses. The cursor can be past the end when
// the table shrinks under it — an engine that lost its models, a flow
// deleted while the screen was open.
func TestALineOnARowThatIsNotThereCloses(t *testing.T) {
	e := env(t, newFile())

	s, out := Open(e).Point(999, e).Edit("x").Key(tea.KeyPressMsg{Code: tea.KeyEnter}, e)
	if s.Editing() || out.Said != "" {
		t.Errorf("saving a row that is not there said %q and left editing %v", out.Said, s.Editing())
	}
}

// TestTurningADialThatIsNotThere is the same shape from the mouse's side.
func TestTurningADialThatIsNotThere(t *testing.T) {
	e := env(t, newFile())

	if out := Open(e).Point(999, e).Cycle(1, e); out.Said != "" || out.Dials != nil {
		t.Errorf("turning a row that is not there answered %+v", out)
	}
}

// TestTypingRunesIntoTheLine.
func TestTypingRunesIntoTheLine(t *testing.T) {
	e := env(t, newFile())

	s := Open(e).Point(0, e).Edit("")
	for _, r := range "es" {
		s, _ = s.Key(tea.KeyPressMsg{Code: r, Text: string(r)}, e)
	}

	if s.Typed() != "es" {
		t.Errorf("the line holds %q", s.Typed())
	}

	// A key with no text — a bare modifier, a function key — writes nothing.
	before := s.Typed()

	s, _ = s.Key(tea.KeyPressMsg{Code: tea.KeyF1}, e)
	if s.Typed() != before {
		t.Errorf("a key with nothing to type wrote %q", s.Typed())
	}
}

// TestXPutsTheRowBack. The screen and the command line are two ways to the
// same file, and a setting a reader can choose on screen and only unchoose
// in a terminal is half a screen.
func TestXPutsTheRowBack(t *testing.T) {
	f := newFile()
	f.held["autopilot"], f.held["unread-cap"] = "on", "40"

	e := env(t, f)

	s := Open(e)
	s.sel = rowOf(t, s, e, "autopilot")

	next, out := s.Key(tea.KeyPressMsg{Text: "x"}, e)

	if f.held["autopilot"] == "on" {
		t.Error("x left autopilot on")
	}

	if !strings.Contains(out.Said, "back to") {
		t.Errorf("the screen said %q, want it to say what the setting went back to", out.Said)
	}

	// And it is the whole gesture: nothing is left open, and the next key
	// lands on the list rather than in a field.
	if next.editing {
		t.Error("x left the row being typed into")
	}
}

// rowOf is where one setting sits on the screen.
func rowOf(t *testing.T, s State, e Env, key string) int {
	t.Helper()

	for i, r := range s.Rows(e) {
		if r.Key == key {
			return i
		}
	}

	t.Fatalf("the screen has no row for %q", key)

	return 0
}
