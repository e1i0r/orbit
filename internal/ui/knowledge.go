package ui

// The screen that lists what Orbit knows.
//
// It is obligatory rather than nice: a system that puts sentences into the
// model's context and refuses work over them, with no way to see which
// sentences, is a system nobody leaves switched on. If it cannot be seen it
// is not trusted, and what is not trusted gets turned off wholesale.
//
// A screen of its own and not a section of the repositories list. What Orbit
// knows is wider than the checkouts — the general facts and the ones about a
// language belong to no repository at all — and hanging the whole of it off
// one of them leaves those with nowhere to go.

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// knowledgeState is the screen: which fact the cursor is on, and the facts as
// they were last read.
//
// The facts are held rather than asked for while drawing, for the reason the
// supervisor's side holds its own: the port reads two directories off disk,
// and a frame is drawn ten times a second.
type knowledgeState struct {
	sel   int
	facts []knowledge.Fact
}

func (m Model) openKnowledge() Model {
	m.screen = screenKnowledge
	m.knowledge = knowledgeState{}

	return m.syncKnowledge()
}

func (m Model) abandonKnowledge() Model {
	m.knowledge = knowledgeState{}
	m.screen = screenList

	return m
}

// syncKnowledge reads the facts again, keeping the cursor on something real.
func (m Model) syncKnowledge() Model {
	if m.opts.KnowsAll == nil {
		return m
	}

	m.knowledge.facts = m.opts.KnowsAll()
	m.knowledge.sel = min(max(m.knowledge.sel, 0), max(len(m.knowledge.facts)-1, 0))

	return m
}

// knowledgeKey is every key on this screen.
func (m Model) knowledgeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape || key.Matches(msg, m.keys.Back):
		return m.abandonKnowledge(), nil
	case msg.Code == tea.KeyUp:
		m.knowledge.sel = max(m.knowledge.sel-1, 0)
		return m, nil
	case msg.Code == tea.KeyDown:
		m.knowledge.sel = min(m.knowledge.sel+1, max(len(m.knowledge.facts)-1, 0))
		return m, nil
	case msg.Code == tea.KeySpace:
		return m.turnFact(), nil
	}

	return m, nil
}

// turnFact turns the fact under the cursor off, or on again.
//
// Off and not deleted: disagreeing with a fact and losing the record that it
// was ever there are different things, and the second is not something a
// keystroke should do. What it stops is the fact being told and the gate
// refusing work over it.
func (m Model) turnFact() Model {
	if m.opts.TurnFact == nil || m.knowledge.sel >= len(m.knowledge.facts) {
		return m
	}

	f := m.knowledge.facts[m.knowledge.sel]
	f.Off = !f.Off

	if err := m.opts.TurnFact(f); err != nil {
		return m.say(err.Error())
	}

	return m.syncKnowledge()
}

// ordered is the facts in the order the screen draws them: the ones that
// belong to no repository first, then each repository's own.
//
// General first because it is what applies everywhere and what somebody
// looking for "why did it do that" checks before anything narrower.
func (m Model) orderedKnowledge() (rootless, owned []knowledge.Fact) {
	for _, f := range m.knowledge.facts {
		if f.Scope.Kind == knowledge.General || f.Scope.Kind == knowledge.Language {
			rootless = append(rootless, f)
			continue
		}

		owned = append(owned, f)
	}

	return rootless, owned
}
