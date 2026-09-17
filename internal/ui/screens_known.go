package ui

// Where the window and the knowledge screen meet.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/known"
	"github.com/e1i0r/orbit/internal/ui/point"
)

// knownEnv is the world the knowledge screen was written against.
func (m Model) knownEnv() known.Env {
	return known.Env{
		Words:    m.opts.Words,
		Keys:     m.keys,
		Frame:    m.frame,
		All:      m.opts.KnowsAll,
		Replace:  m.opts.ReplaceFact,
		Story:    m.opts.RuleStory,
		Turn:     m.opts.TurnFact,
		Waiting:  m.opts.Waiting,
		Enforced: m.opts.RepoGates,
		Places:   m.opts.RepoFolders,
		Commands: m.opts.RepoChecks,
		Keep:     m.opts.KeepRule,
		Drop:     m.opts.DropRule,
		Forget:   m.opts.ForgetRule,
		Repos:    m.repoPaths(),
	}
}

// repoPaths is every checkout on the board, in the order it lists them.
func (m Model) repoPaths() []string {
	out := make([]string, 0, len(m.board.RepoList))
	for _, one := range m.board.RepoList {
		out = append(out, one.Path)
	}

	return out
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

// hitKnowledge is what the screen has at that cell.
func (m Model) hitKnowledge(x, y int) point.Target {
	return m.knowledge.Hit(x, y, m.knownEnv())
}

// clickedKnowledge is a row pointed at: one click puts the cursor on it, a
// second opens it — the two-step every list in the window has.
func (m Model) clickedKnowledge(i int) (tea.Model, tea.Cmd) {
	next, already := m.knowledge.PointAt(i, m.knownEnv())
	m.knowledge = next

	if !already {
		return m, nil
	}

	return m.tookKnowledge(m.knowledge.Chosen(m.knownEnv()))
}

// pickedKnowledge is an option of an open list, clicked.
func (m Model) pickedKnowledge(at int) (tea.Model, tea.Cmd) {
	m.knowledge = m.knowledge.Pick(at, m.knownEnv())

	return m, nil
}

// wheelKnowledge is one notch of the wheel over the list.
func (m Model) wheelKnowledge(d int) Model {
	m.knowledge = m.knowledge.Scroll(d, m.knownEnv())

	return m
}

// knowledgeRows is the screen drawn.
func (m Model) knowledgeRows(h, w int) []string {
	return m.knowledge.View(h, w, m.knownEnv())
}
