package ui

// Where the window and the cheat sheet meet.

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/cheat"
)

// keyed is every command a key already says: [n] starts, [a] asks the
// note, [d] marks read. Listing one of them again under its family
// would read the same verb twice in two voices.
var keyed = map[string]bool{
	"start": true, "pause": true, "resume": true, "continue": true,
	"skip": true, "cancel": true, "requeue": true, "note": true,
	"read": true, "delete": true, "take": true,
}

// cheatEnv is the world the cheat sheet was written against. The two lists
// are built here rather than kept there: the verbs come from the keymap and
// the sentences ? answers with, and the tabs from the detail screen's own
// menu.
func (m Model) cheatEnv() cheat.Env {
	bindings := m.keys.TaskVerbs()

	verbs := make([]cheat.Verb, 0, len(bindings))
	for _, b := range bindings {
		// The compose key writes the task down on the board; every other
		// key acts on one task.
		family := "task"
		if key.Matches(firstKey(b), m.keys.Compose) {
			family = "board"
		}

		verbs = append(verbs, cheat.Verb{Key: b.Help().Key, Says: m.meaning(firstKey(b)), Family: family})
	}

	// And what only a command does: the task's and the pull request's,
	// grouped under their families the way the menu drills into them.
	// What a key already says is not listed twice: the sheet would read
	// every verb in two voices saying near the same thing.
	for _, c := range m.opts.Commands {
		if c.Name != "task" && c.Name != "pr" {
			continue
		}

		for _, kid := range c.Children {
			if keyed[kid.Name] {
				continue
			}

			about := ""
			if kid.About != nil {
				about = kid.About(m.opts.Words)
			}

			verbs = append(verbs, cheat.Verb{Key: c.Name + " " + kid.Name, Says: about, Family: c.Name})
		}
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
