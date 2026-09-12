package ui

// Where the window and the menu meet.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/ui/menu"
	"github.com/e1i0r/orbit/internal/ui/point"
)

// menuEnv is the world the menu was written against. The commands are
// translated here: the screen is handed sentences, not the printer and a
// closure to call it with.
func (m Model) menuEnv() menu.Env {
	p := m.opts.Words

	cmds := make([]menu.Command, 0, len(m.opts.Commands))

	for _, c := range m.opts.Commands {
		row := menu.Command{Name: c.Name, Refused: c.Refused, AboutATask: c.AboutATask}

		if c.About != nil {
			row.About = c.About(p)
		}

		for _, kid := range c.Children {
			about := ""
			if kid.About != nil {
				about = kid.About(p)
			}

			row.Children = append(row.Children, menu.Child{Name: kid.Name, About: about})
		}

		if c.Because != nil {
			row.Because = c.Because(p)
		}

		cmds = append(cmds, row)
	}

	return menu.Env{
		Words:    p,
		Keys:     m.keys,
		Frame:    m.frame,
		Detail:   m.screen == screenDetail,
		Commands: cmds,
		Panes:    m.paneMenu(),
		Verbs:    m.taskVerbs,
		Args:     func(id string) []string { return repoArgs(m.taskRepoPath(id), id) },
	}
}

// taskVerbs is what can be done to one task, which the window answers
// because the board is the window's: the menu is handed the verbs and the
// reason each refusal gives, and asks nothing about the run behind them.
func (m Model) taskVerbs(id string) ([]keymap.Affordance, bool) {
	t, ok := m.task(id)
	if !ok {
		return nil, false
	}

	return m.keys.Affordances(t, m.conditions(t)), true
}

// tookMenu does what the menu asked the window for. Nothing here decides:
// a keystroke goes through the map a pressed key goes through, a command
// through the watch every other command is run in, and a message through
// the box it is typed into.
func (m Model) tookMenu(next menu.State, out menu.Out) (tea.Model, tea.Cmd) {
	// Which task the menu was about, taken before it is replaced: a
	// command that needs a message is handed the box rather than run, and
	// the box has to know what it will be talking to.
	id := m.menu.Task()

	m.menu = next
	if out.Leave {
		m.menu = menu.State{}
	}

	switch {
	case out.Pane != "":
		if t, ok := keyToPane(out.Pane); ok {
			m = m.showTab(t)
		}

		return m, nil
	case out.Ask:
		return m.openMessage(out.Run, id), nil
	case out.Run != "":
		return m.launchNamed(out.Run, out.Args)
	case out.Send != "":
		return m.sendKey(keystroke(out.Send))
	}

	return m, nil
}

// openMenu brings the menu up on a target.
func (m Model) openMenu(id string) Model {
	m.menu = menu.Open(id, m.menuEnv())

	return m
}

// closeMenu takes it down.
func (m Model) closeMenu() Model {
	m.menu = menu.State{}

	return m
}

// openMenuForContext is m: the menu for the row under the cursor, for the
// task being viewed one level down, or — with no cursor at all — the
// board's own menu.
func (m Model) openMenuForContext() Model {
	if m.screen == screenDetail && m.detail != "" {
		return m.openMenu(m.detail)
	}

	if r, ok := m.selected(); ok && !r.head {
		return m.openMenu(r.task.ID)
	}

	return m.openMenu("")
}

// menuKey hands one keystroke to the menu.
func (m Model) menuKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, out := m.menu.Key(msg, m.menuEnv())

	return m.tookMenu(next, out)
}

// chooseMenu acts on the selection.
func (m Model) chooseMenu() (tea.Model, tea.Cmd) {
	next, out := m.menu.Enter(m.menuEnv())

	return m.tookMenu(next, out)
}

// chooseMenuEntry is the entry a click landed on, named rather than
// numbered.
func (m Model) chooseMenuEntry(key string) (tea.Model, tea.Cmd) {
	next, out := m.menu.Choose(key, m.menuEnv())

	return m.tookMenu(next, out)
}

// menuPick moves the selection, for the wheel.
func (m Model) menuPick(d int) Model {
	m.menu = m.menu.Wheel(d, m.menuEnv())

	return m
}

// menuRows is the menu drawn in the body.
func (m Model) menuRows(h, w int) []string {
	return m.menu.View(h, w, m.menuEnv())
}

// hitMenu is what the list has at that cell.
func (m Model) hitMenu(x, y int) point.Target {
	return m.menu.Hit(x, y, m.menuEnv())
}
