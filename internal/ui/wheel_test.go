package ui

// One notch of the wheel, on every screen there is.
//
// The wheel is the gesture a reader uses without thinking, and it is the one
// most easily left behind: a screen added without a branch here scrolls
// nothing, silently, and reads as a window that has stopped. Most of this
// file read zero.
//
// Three notches of one row rather than one of three, so that a wheel and a
// held arrow key are the same gesture as far as everything downstream is
// concerned — which is why what is asserted is that the screen moved, in the
// screen's own units, rather than by how much.

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/layout"
)

// notch is one turn of the wheel over the body.
func notch(up bool) tea.Mouse {
	button := tea.MouseWheelDown
	if up {
		button = tea.MouseWheelUp
	}

	return tea.MouseWheelMsg{X: 40, Y: 10, Button: button}.Mouse()
}

// screenful is the window as a reader sees it, so a test can say the wheel
// moved something without knowing which field holds the offset.
func screenful(t *testing.T, m Model) string {
	t.Helper()

	return ansi.Strip(m.View().Content)
}

// TestTheWheelOnlyTurnsOverTheBody. The header and the bar are not lists,
// and a notch that moved the board because the pointer was over the status
// line is a window that scrolls when the reader is not pointing at it.
func TestTheWheelOnlyTurnsOverTheBody(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.cursor = 2

	for y := range 30 {
		if m.frame.At(y) == layout.RegionBody {
			continue
		}

		turned := m.wheel(tea.MouseWheelMsg{X: 40, Y: y, Button: tea.MouseWheelDown}.Mouse())
		if turned.cursor != m.cursor {
			t.Errorf("a notch at row %d, outside the body, moved the cursor to %d", y, turned.cursor)
		}
	}
}

// TestASidewaysWheelDoesNothing. The pane that can scroll that way — a diff
// wider than the terminal — is scrolled with ←→, and a sideways wheel is a
// gesture few mice have and fewer readers expect.
func TestASidewaysWheelDoesNothing(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.cursor = 2

	for _, button := range []tea.MouseButton{tea.MouseWheelLeft, tea.MouseWheelRight} {
		turned := m.wheel(tea.MouseWheelMsg{X: 40, Y: 10, Button: button}.Mouse())
		if turned.cursor != m.cursor {
			t.Errorf("a sideways notch moved the cursor to %d", turned.cursor)
		}
	}
}

// TestTheBoardMovesTheCursorAndNotTheOffset. The offset is not the caller's
// to set: follow owns it, and it is what keeps the cursor on screen and the
// last page from scrolling past its end.
func TestTheBoardMovesTheCursorAndNotTheOffset(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.cursor = 0

	down := m.wheel(notch(false))
	if down.cursor <= 0 {
		t.Fatalf("a notch down left the cursor at %d, want it moved", down.cursor)
	}

	up := down.wheel(notch(true))
	if up.cursor >= down.cursor {
		t.Errorf("a notch up left the cursor at %d, want it above %d", up.cursor, down.cursor)
	}

	// And it stops at the ends rather than running past them.
	for range 40 {
		up = up.wheel(notch(true))
	}

	if up.cursor < 0 {
		t.Errorf("the cursor ran off the top to %d", up.cursor)
	}
}

// TestNoScreenLetsTheWheelFallThroughToTheBoard.
//
// The failure this is about: a screen added without a branch in wheel does
// not scroll, and the notch reaches the board underneath — so a reader
// turning the wheel over the settings silently moves the cursor on a list
// they are not looking at, and finds it somewhere else when they go back.
//
// One case per screen, and the assertion is the same for all of them
// because it does not depend on what happens to be on the screen: whatever
// the notch did, it did not move the board.
func TestNoScreenLetsTheWheelFallThroughToTheBoard(t *testing.T) {
	screens := map[string]screen{
		"the settings":     screenSettings,
		"what orbit knows": screenKnowledge,
		"the engines":      screenEngines,
		"the cheat sheet":  screenHelp,
		"the supervisor":   screenSupervisor,
		"the flows":        screenFlows,
		"the quota":        screenQuota,
		"a task":           screenDetail,
		"writing one":      screenCompose,
		"the repositories": screenRepos,
	}

	for name, on := range screens {
		t.Run(name, func(t *testing.T) {
			m, _ := testModel(t, 100, 30)
			m.cursor = 1
			m.screen = on

			for _, up := range []bool{false, true} {
				turned := m.wheel(notch(up))

				if turned.cursor != m.cursor {
					t.Errorf("a notch over %s moved the board's cursor to %d", name, turned.cursor)
				}

				if turned.screen != on {
					t.Errorf("a notch over %s left the screen", name)
				}
			}
		})
	}
}

// TestTheScreensThatAreListsMoveUnderIt. The three the fixture fills: a
// screen that draws a list and does not turn under the wheel is a list the
// reader has to reach for the arrows on.
func TestTheScreensThatAreListsMoveUnderIt(t *testing.T) {
	lists := map[string]screen{
		"the settings":    screenSettings,
		"the engines":     screenEngines,
		"the cheat sheet": screenHelp,
	}

	for name, on := range lists {
		t.Run(name, func(t *testing.T) {
			m, _ := testModel(t, 100, 30)
			m.screen = on

			before := screenful(t, m)

			after := m
			for range 6 {
				after = after.wheel(notch(false))
			}

			if screenful(t, after) == before {
				t.Errorf("six notches over %s changed nothing", name)
			}
		})
	}
}

// TestAListWithOneRowChosenDoesNotScrollThePageUnderIt. The menu and the
// palette own the keyboard while they are up, so the wheel moves the pick
// rather than the screen behind it — two things moving for one gesture is
// the reader losing their place in one of them.
func TestAListWithOneRowChosenDoesNotScrollThePageUnderIt(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.cursor = 1

	for name, open := range map[string]func(Model) Model{
		"the palette": Model.openPalette,
		"the menu":    Model.openMenuForContext,
	} {
		t.Run(name, func(t *testing.T) {
			opened := open(m)

			turned := opened.wheel(notch(false))
			if turned.cursor != opened.cursor {
				t.Errorf("a notch over %s moved the board to %d", name, turned.cursor)
			}
		})
	}
}

// TestARunBeingWatchedOffersNothingToScrollBackFor. Its output keeps its own
// tail on screen, so the wheel does nothing rather than scrolling something
// that is not being shown.
func TestARunBeingWatchedOffersNothingToScrollBackFor(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.watchUp = true
	m.cursor = 2

	if turned := m.wheel(notch(false)); turned.cursor != m.cursor {
		t.Errorf("a notch under a watched run moved the board to %d", turned.cursor)
	}
}

// TestABindingWithNoKeysSendsNothing. A clicked hint and a notch both send
// the keystroke a binding is reached by, and an empty keystroke is one every
// map ignores anyway — the difference is that this says so.
func TestABindingWithNoKeysSendsNothing(t *testing.T) {
	if got := firstKey(key.NewBinding()); got != "" {
		t.Errorf("a binding with no keys sends %q, want nothing", got)
	}

	if got := firstKey(key.NewBinding(key.WithKeys("j", "down"))); got != "j" {
		t.Errorf("a binding sends %q, want the first key it is reached by", got)
	}
}
