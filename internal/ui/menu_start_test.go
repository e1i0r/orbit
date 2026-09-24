package ui

import (
	"testing"

	"github.com/e1i0r/orbit/internal/ui/menu"
	"github.com/e1i0r/orbit/internal/words"
)

// TestTheMenusStartOpensTheDialogOnTheDiffTab. The menu sends a verb as
// its letter, and on the diff tab n is the next hunk: "start a run" moved
// the diff and started nothing.
func TestTheMenusStartOpensTheDialogOnTheDiffTab(t *testing.T) {
	m, _ := openIn(t, words.For("en"), "ACME-2698", fixtureEntries(), wideDiff())
	m = showing(t, m, tabDiff)

	next, _ := m.tookMenu(menu.State{}, menu.Out{Leave: true, Send: m.keys.Start.Help().Key})
	if got := asModel(t, next).screen; got != screenStart {
		t.Errorf("start a run from the menu on the diff tab left screen %v, want the start dialog", got)
	}
}

// TestTheWindowsKeysWorkOnATasksScreen. A is drawn in the bar on every
// screen, : is the command line from anywhere and / the board's filter,
// and on a task's screen all three did nothing and said nothing.
func TestTheWindowsKeysWorkOnATasksScreen(t *testing.T) {
	m, _ := openIn(t, words.For("en"), "ACME-2698", fixtureEntries(), wideDiff())

	next, _ := m.detailKey(keystroke(":"))
	if !asModel(t, next).palette.Up() {
		t.Error(": on a task's screen left the command line down")
	}

	next, _ = m.detailKey(keystroke("/"))
	if got := asModel(t, next); got.screen != screenList || !got.filtering {
		t.Errorf("/ on a task's screen left screen %v, filtering %v; want the board's filter",
			got.screen, got.filtering)
	}

	was := m.autopilotOn()

	next, _ = m.detailKey(keystroke("A"))
	if got := asModel(t, next).autopilotOn(); got == was {
		t.Errorf("A on a task's screen left autopilot %v", got)
	}
}
