package ui

// Where the window and the knowledge screen meet.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/known"
)

// knownEnv is the world the knowledge screen was written against.
func (m Model) knownEnv() known.Env {
	return known.Env{
		Words:   m.opts.Words,
		Keys:    m.keys,
		All:     m.opts.KnowsAll,
		Replace: m.opts.ReplaceFact,
		Turn:    m.opts.TurnFact,
		Waiting: m.opts.Waiting,
		Keep:    m.opts.KeepRule,
		Drop:    m.opts.DropRule,
		Repo:    m.oneRepo(),
	}
}

// oneRepo is the single repository the board is on, and nothing when there
// is more than one to choose between.
func (m Model) oneRepo() string {
	if len(m.board.RepoList) == 1 {
		return m.board.RepoList[0].Path
	}

	return ""
}

// tookKnowledge does what the screen asked the window for.
func (m Model) tookKnowledge(next known.State, out known.Out) (tea.Model, tea.Cmd) {
	m.knowledge = next

	if out.Leave {
		m.screen = screenList
	}

	if out.Said != "" {
		m = m.say(out.Said)
	}

	return m, nil
}

// openKnowledge puts the screen up, with the store read.
func (m Model) openKnowledge() Model {
	m.knowledge = known.Open(m.knownEnv())
	m.screen = screenKnowledge

	return m
}

// knowledgeKey hands one keystroke to the screen.
func (m Model) knowledgeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, out := m.knowledge.Key(msg, m.knownEnv())

	return m.tookKnowledge(next, out)
}

// syncKnowledge reads the store again, for a window that has just changed
// what is in it.
func (m Model) syncKnowledge() Model {
	m.knowledge = m.knowledge.Sync(m.knownEnv())

	return m
}

// factCount is what the header's chip says, read from what was last loaded.
func (m Model) factCount() int {
	return m.knowledge.Count()
}

// knowledgeRows is the screen drawn.
func (m Model) knowledgeRows(h, w int) []string {
	return m.knowledge.View(h, w, m.knownEnv())
}
