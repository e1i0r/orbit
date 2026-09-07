package menu

// The cursor, and what choosing a row asks the window for.

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// wheelRows is how far one notch of the wheel moves, as the window turns
// it: several rows at a time, which is what lands a cursor on the line
// between two blocks.
const wheelRows = 3

// down and up are the keys, and enter is the one that chooses.
var (
	down  = tea.KeyPressMsg{Code: tea.KeyDown}
	up    = tea.KeyPressMsg{Code: tea.KeyUp}
	enter = tea.KeyPressMsg{Code: tea.KeyEnter}
	esc   = tea.KeyPressMsg{Code: tea.KeyEsc}
)

// TestTheCursorSkipsTheHeadings, both when the menu opens and when it is
// walked across the line between the two blocks — a cursor parked on a
// heading is a row that looks chosen and does nothing.
func TestTheCursorSkipsTheHeadings(t *testing.T) {
	e := inside(t)

	s := Open(theTask, e)
	if s.Entries(e)[s.At()].Head {
		t.Fatalf("the menu opened on %+v, want the first entry there is to choose", s.Entries(e)[s.At()])
	}

	for i := range s.Entries(e) {
		s, _ = s.Key(down, e)
		if got := s.Entries(e)[s.At()]; got.Head {
			t.Fatalf("walking down landed on %+v after %d moves, want a row that does something", got, i+1)
		}
	}

	for i := range s.Entries(e) {
		s, _ = s.Key(up, e)
		if got := s.Entries(e)[s.At()]; got.Head {
			t.Fatalf("walking back up landed on %+v after %d moves, want a row that does something", got, i+1)
		}
	}
}

// TestTheWheelLandsOnSomethingChoosable. It moves several rows at a time,
// and one of those jumps lands on the line between the two blocks.
func TestTheWheelLandsOnSomethingChoosable(t *testing.T) {
	e := inside(t)

	s := Open(theTask, e)
	for range s.Entries(e) {
		s = s.Wheel(wheelRows, e)
		if got := s.Entries(e)[s.At()]; got.Head {
			t.Fatalf("the wheel landed on %+v, want a row that does something", got)
		}
	}

	for range s.Entries(e) {
		s = s.Wheel(-wheelRows, e)
		if got := s.Entries(e)[s.At()]; got.Head {
			t.Fatalf("the wheel back up landed on %+v, want a row that does something", got)
		}
	}
}

// TestWalkingDownReachesTheVerbs: the block below the panes is reachable
// from the keyboard, not only by pressing the verb's own key.
func TestWalkingDownReachesTheVerbs(t *testing.T) {
	e := inside(t)

	s := Open(theTask, e)
	for range s.Entries(e) {
		s, _ = s.Key(down, e)
	}

	got := s.Entries(e)[s.At()]
	if got.Head || got.Pane != "" {
		t.Errorf("the bottom of the menu is %+v, want a verb about the task", got)
	}
}

// TestChoosingAPaneShowsIt, and takes the menu down with it.
func TestChoosingAPaneShowsIt(t *testing.T) {
	e := inside(t)

	s := Open(theTask, e)

	_, out := s.Enter(e)
	if !out.Leave || out.Pane != "1" {
		t.Errorf("choosing the first pane asked for %+v, want the menu down and pane 1 shown", out)
	}
}

// TestThePaneKeysWorkFromTheMenuToo. The menu is where a reader goes to
// find out which key a pane is on, and pressing it there should be pressing
// it.
func TestThePaneKeysWorkFromTheMenuToo(t *testing.T) {
	e := inside(t)

	for _, k := range []string{"6", "w", "W"} {
		_, out := Open(theTask, e).Key(tea.KeyPressMsg{Text: k}, e)
		if !out.Leave || out.Pane == "" {
			t.Errorf("pressing %q on the menu asked for %+v, want the pane it names", k, out)
		}
	}
}

// TestAPaneKeyOnTheBoardsMenuIsNotAPane. There is no task being read there,
// so the digits are nothing and the menu stays up rather than going
// somewhere.
func TestAPaneKeyOnTheBoardsMenuIsNotAPane(t *testing.T) {
	e := world(t)

	_, out := Open("", e).Key(tea.KeyPressMsg{Text: "6"}, e)
	if !nothing(out) {
		t.Errorf("pressing 6 on the board's menu asked for %+v, want nothing", out)
	}
}

// TestChoosingACommandRunsItWithWhatItNeeds, and one that takes a message
// opens the box instead: the menu has nothing to fill the sentence in with.
func TestChoosingACommandRunsItWithWhatItNeeds(t *testing.T) {
	e := world(t)

	s := Open(theTask, e)

	for _, c := range []struct {
		name string
		ask  bool
	}{{name: "approve"}, {name: "note", ask: true}} {
		at := index(t, s, e, c.name)

		_, out := s.Point(at).Enter(e)
		if out.Run != c.name || out.Ask != c.ask {
			t.Errorf("choosing %s asked for %+v, want run=%q ask=%v", c.name, out, c.name, c.ask)
		}

		if len(out.Args) == 0 {
			t.Errorf("choosing %s ran it with nothing, want the repository and the id", c.name)
		}
	}
}

// TestChoosingAVerbSendsItsKeystroke, which is how a refused verb refuses
// here with the sentence it refuses everywhere else: the window puts it
// through the map a pressed key goes through.
func TestChoosingAVerbSendsItsKeystroke(t *testing.T) {
	e := world(t)

	s := Open(theTask, e)

	_, out := s.Enter(e)
	if out.Send != "p" {
		t.Errorf("choosing the first verb asked for %+v, want the p it is bound to", out)
	}
}

// TestChoosingNothingDoesNothing: a cursor past the end of a list that got
// shorter, and a heading, both answer with the menu left as it was.
func TestChoosingNothingDoesNothing(t *testing.T) {
	e := inside(t)

	s := Open(theTask, e)

	if _, out := s.Point(99).Enter(e); !nothing(out) {
		t.Errorf("choosing past the end asked for %+v, want nothing", out)
	}

	if _, out := s.Point(0).Enter(e); !nothing(out) {
		t.Errorf("choosing the heading asked for %+v, want nothing", out)
	}
}

// TestEscTakesTheMenuDownAndNothingElse.
func TestEscTakesTheMenuDownAndNothingElse(t *testing.T) {
	e := world(t)

	_, out := Open("", e).Key(esc, e)
	if !out.Leave || out.Run != "" || out.Send != "" || out.Pane != "" {
		t.Errorf("esc asked for %+v, want the menu down and nothing done", out)
	}
}

// TestEnterChoosesWhatTheCursorIsOn, which is the same door the pointer
// reaches on a second click.
func TestEnterChoosesWhatTheCursorIsOn(t *testing.T) {
	e := world(t)

	_, out := Open("", e).Key(enter, e)
	if out.Run != "reconcile" {
		t.Errorf("enter on the board's menu asked for %+v, want reconcile run", out)
	}
}

// TestAClickMovesTheCursorBeforeItChooses. The list is recomputed between
// press and release, so a click carries what identifies the entry rather
// than where it sat — and a row that was not the chosen one is chosen
// first, not acted on.
func TestAClickMovesTheCursorBeforeItChooses(t *testing.T) {
	e := world(t)

	s := Open("", e)

	next, out := s.Choose("export", e)
	if !nothing(out) {
		t.Fatalf("the first click on export asked for %+v, want the cursor moved and nothing done", out)
	}

	if got := next.Entries(e)[next.At()]; got.Command != "export" {
		t.Fatalf("the cursor moved to %+v, want export", got)
	}

	_, out = next.Choose("export", e)
	if out.Run != "export" {
		t.Errorf("the second click asked for %+v, want export run", out)
	}
}

// TestAClickOnNothingIsNothing: an entry that has left the list between the
// press and the release.
func TestAClickOnNothingIsNothing(t *testing.T) {
	e := world(t)

	if _, out := Open("", e).Choose("gone", e); !nothing(out) {
		t.Errorf("clicking an entry that is no longer there asked for %+v, want nothing", out)
	}
}

// nothing is an Out that asked the window for none of the four things it
// can ask for.
func nothing(out Out) bool {
	return !out.Leave && out.Pane == "" && out.Send == "" && out.Run == ""
}

// index is where a named command sits in the menu as it is drawn, which is
// not where it sits in the table: the menu leaves entries out and adds
// others.
func index(t *testing.T, s State, e Env, name string) int {
	t.Helper()

	for i, entry := range s.Entries(e) {
		if entry.Command == name {
			return i
		}
	}

	t.Fatalf("no %s on the menu", name)

	return 0
}
