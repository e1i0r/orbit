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
