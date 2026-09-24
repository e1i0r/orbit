package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/words"
)

// TestLTurnsTheLanguage. L is drawn and explained as the language, and
// nothing answered it: on the board it opened the map tab no row shows,
// and on a task's screen it opened the map. It now does what a click on
// the header's badge does, on both.
func TestLTurnsTheLanguage(t *testing.T) {
	board, _ := testModel(t, 100, 30)
	task, _ := openIn(t, words.For("en"), "ACME-2698", fixtureEntries(), wideDiff())

	for name, press := range map[string]func() (tea.Model, tea.Cmd){
		"the board":       func() (tea.Model, tea.Cmd) { return board.listKey(keystroke("L")) },
		"a task's screen": func() (tea.Model, tea.Cmd) { return task.detailKey(keystroke("L")) },
	} {
		next, cmd := press()
		got := settled(t, asModel(t, next), cmd)

		if langBadge(got) != "ES" {
			t.Errorf("L on %s left the badge %q, want ES", name, langBadge(got))
		}

		if got.tab == tabMap {
			t.Errorf("L on %s opened the map", name)
		}
	}
}

// settled hands a model every message its command answers with, the
// batches opened, the way the program would.
func settled(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()

	if cmd == nil {
		return m
	}

	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			m = settled(t, m, c)
		}

		return m
	}

	next, _ := m.Update(msg)

	return asModel(t, next)
}
