package ui

// The help overlay as the window reaches it: which screen it remembers to
// return to. The sheet's own keyboard is tested in internal/ui/cheat.

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestTheSheetGoesBackToWhereItWasOpenedFrom, which is the window's own
// business: the sheet is handed a number and hands it back.
func TestTheSheetGoesBackToWhereItWasOpenedFrom(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.screen = screenSettings

	m = m.openHelp()
	if m.screen != screenHelp {
		t.Fatalf("? from settings left the window on %v", m.screen)
	}

	back, _ := m.helpKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if got := asModel(t, back); got.screen != screenSettings {
		t.Errorf("esc left the window on %v, want the settings it was opened from", got.screen)
	}

	// Opening the sheet while it is already up does not remember itself: a
	// reader who leaves it would have nowhere to go.
	m.screen = screenHelp

	again, _ := m.openHelp().helpKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if got := asModel(t, again); got.screen != screenList {
		t.Errorf("esc from a sheet opened over a sheet left the window on %v, want the board", got.screen)
	}
}
