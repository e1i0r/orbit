package ui

// Where the window and the repository list meet.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/repos"
)

// reposEnv is the world the repository list was written against.
func (m Model) reposEnv() repos.Env {
	return repos.Env{
		Words:  m.opts.Words,
		Keys:   m.keys,
		Board:  m.board,
		Filter: m.repoFilter,
	}
}

// tookRepos does what the screen asked the window for.
func (m Model) tookRepos(next repos.State, out repos.Out) (tea.Model, tea.Cmd) {
	m.repolist = next

	// The filter is the window's: the board is drawn through it long after
	// this screen has closed.
	switch {
	case out.Cleared:
		m.repoFilter = ""
	case out.Filter != "":
		m.repoFilter = out.Filter
	}

	if out.Leave {
		m.screen = screenList
	}

	if out.Said != "" {
		m = m.say(out.Said)
	}

	return m, nil
}

// openRepos puts the list up.
func (m Model) openRepos() Model {
	m.repolist = repos.Open()
	m.screen = screenRepos

	return m
}

// repolistKey hands one keystroke to the list.
func (m Model) repolistKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, out := m.repolist.Key(msg, m.reposEnv())

	return m.tookRepos(next, out)
}

// chooseRepo takes the repository a click landed on.
func (m Model) chooseRepo(name string) (tea.Model, tea.Cmd) {
	next, out := m.repolist.Choose(name, m.reposEnv())

	return m.tookRepos(next, out)
}

// collectRepos is every repository the board found, which the form that
// starts a task and the pointer both read.
func (m Model) collectRepos() []repos.Item {
	return repos.List(m.reposEnv())
}

// repolistRows is the list drawn.
func (m Model) repolistRows(h, w int) []string {
	return m.repolist.View(h, w, m.reposEnv())
}
