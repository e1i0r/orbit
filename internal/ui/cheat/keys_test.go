package cheat

// The sheet's own keyboard: what scrolls it, what closes it, and where it
// goes back to.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/words"
)

// sheet is the screen open on the fixture Env, opened from screen 3 —
// whichever one the window calls that.
func sheet(t *testing.T) (State, Env) {
	t.Helper()
	t.Setenv("ORBIT_HOME", t.TempDir())

	return Open(fromSomewhere), world(t, words.For("en"))
}

// fromSomewhere is a screen number of the window's, which this package never
// interprets: it hands it back and that is all.
const fromSomewhere = 3

// TestScrollingStopsAtTheTopAndHasNoCeilingOfItsOwn — the drawing is what
// clamps the bottom, because only it knows how many lines there are.
func TestScrollingStopsAtTheTopAndHasNoCeilingOfItsOwn(t *testing.T) {
	s, e := sheet(t)

	down, _ := s.Key(press(tea.KeyDown), e)
	if down.offset != 1 {
		t.Errorf("down = offset %d, want 1", down.offset)
	}

	up, _ := down.Key(press(tea.KeyUp), e)
	if up.offset != 0 {
		t.Errorf("up = offset %d, want 0", up.offset)
	}

	if again, _ := up.Key(press(tea.KeyUp), e); again.offset != 0 {
		t.Errorf("up at the top = offset %d, want it to stay there", again.offset)
	}
}

// TestFourKeysCloseIt, one of which is the key that opened it: ? is how a
// reader gets here and the first thing they will press to leave.
func TestFourKeysCloseIt(t *testing.T) {
	s, e := sheet(t)

	for _, msg := range []tea.KeyPressMsg{
		{Code: tea.KeyEscape},
		{Code: tea.KeyEnter},
		{Code: '?', Text: "?"},
		{Code: 'q', Text: "q"},
	} {
		next, out := s.Key(msg, e)
		if !out.Leave {
			t.Errorf("%v did not close the sheet", msg)
		}

		if out.Back != fromSomewhere {
			t.Errorf("%v went back to screen %d, want %d", msg, out.Back, fromSomewhere)
		}

		if next != (State{}) {
			t.Errorf("%v left the sheet holding %+v", msg, next)
		}
	}

	if _, out := s.Key(press('z'), e); out.Leave {
		t.Error("a key the sheet has no use for closed it")
	}
}

// TestScrollingPastTheEndClampsRatherThanSlicingOutOfRange.
func TestScrollingPastTheEndClampsRatherThanSlicingOutOfRange(t *testing.T) {
	s, e := sheet(t)

	title := e.Words.T("help.title", "Help and keyboard shortcuts (cheat sheet)")

	full := s.View(40, 100, e)
	if len(full) == 0 || !strings.Contains(strings.Join(full, "\n"), title) {
		t.Fatalf("the sheet at the top does not show its title: %v", full)
	}

	s.offset = 3

	scrolled := s.View(40, 100, e)
	if len(scrolled) != len(full) {
		t.Errorf("scrolled three rows the sheet is %d lines, want it padded back to %d", len(scrolled), len(full))
	}

	if strings.Contains(strings.Join(scrolled, "\n"), title) {
		t.Error("scrolled three rows the sheet still shows its title")
	}

	s.offset = 100000
	if got := s.View(40, 100, e); len(got) == 0 {
		t.Error("an offset past the end drew nothing, want it clamped to the last line")
	}

	if got := s.View(0, 100, e); got != nil {
		t.Errorf("no room at all drew %v, want nothing", got)
	}
}

// press is one keystroke as the event loop delivers it. A key with no
// character of its own — an arrow, escape — carries no text, which is how
// the event loop tells the two apart.
func press(code rune) tea.KeyPressMsg {
	if code < ' ' || code > '~' {
		return tea.KeyPressMsg{Code: code}
	}

	return tea.KeyPressMsg{Code: code, Text: string(code)}
}
