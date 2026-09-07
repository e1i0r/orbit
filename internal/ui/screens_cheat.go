package ui

// Where the window and the cheat sheet meet.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/cheat"
)

// cheatEnv is the world the cheat sheet was written against. The two lists
// are built here rather than kept there: the verbs come from the keymap and
// the sentences ? answers with, and the tabs from the detail screen's own
// menu.
func (m Model) cheatEnv() cheat.Env {
	bindings := m.keys.TaskVerbs()

	verbs := make([]cheat.Verb, 0, len(bindings))
	for _, b := range bindings {
		verbs = append(verbs, cheat.Verb{Key: b.Help().Key, Says: m.meaning(firstKey(b))})
	}

	panes := m.paneMenu()

	tabs := make([]cheat.Tab, 0, len(panes))
	for _, p := range panes {
		tabs = append(tabs, cheat.Tab{Glyph: "[" + p.Key + "]", Title: p.Title, Detail: p.Detail})
	}

	return cheat.Env{Words: m.opts.Words, Keys: m.keys, Verbs: verbs, Tabs: tabs}
}

// openHelp puts the sheet up.
func (m Model) openHelp() Model {
	prev := m.screen
	if prev == screenHelp {
		prev = screenList
	}

	m.help = cheat.Open(int(prev))
	m.screen = screenHelp

	return m
}

// helpKey hands one keystroke to the sheet.
func (m Model) helpKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, out := m.help.Key(msg, m.cheatEnv())
	m.help = next

	if out.Leave {
		m.screen = screen(out.Back)
		if m.screen == screenHelp {
			m.screen = screenList
		}
	}

	return m, nil
}

// helpRows is the sheet drawn.
func (m Model) helpRows(h, w int) []string {
	return m.help.View(h, w, m.cheatEnv())
}
