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
	"strings"
	"time"

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
	// read is whether the port has been asked at all. It is not len(facts):
	// a workspace where nothing has been written down answers an empty
	// list, and without this the header's chip would ask again on every
	// board refresh — two directories off disk, twice a second, to be told
	// zero each time.
	read bool

	// editing is the fact under the cursor being corrected in place, and in
	// holds the two things about it that are text: what it says, and the
	// command that decides whether it can stop the work.
	//
	// Two fields and not the whole record. A fact's scope and its source are
	// what make it traceable, and neither is something to retype — the
	// source is where it came from, which nobody may edit, and the scope is
	// moved with its own gesture rather than by typing a path.
	editing bool
	field   int
	in      [factFields]input
}

// The two fields of a fact that are typed into.
const (
	factPhrase = iota
	factCheck
	factFields
)

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
	m.knowledge.read = true
	m.knowledge.sel = min(max(m.knowledge.sel, 0), max(len(m.knowledge.facts)-1, 0))

	return m
}

// knowledgeKey is every key on this screen.
func (m Model) knowledgeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.knowledge.editing {
		return m.editingKey(msg)
	}

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
	case msg.Code == 'e' || msg.Code == 'E':
		return m.editFact(), nil
	case msg.Code == 'n' || msg.Code == 'N':
		return m.newFact(), nil
	case msg.Code == tea.KeyLeft:
		return m.moveFact(wider), nil
	case msg.Code == tea.KeyRight:
		return m.moveFact(narrower), nil
	}

	return m, nil
}

// editFact opens the fact under the cursor, with what it already says in the
// line: a fact is corrected far more often than it is rewritten.
func (m Model) editFact() Model {
	if m.opts.ReplaceFact == nil || m.knowledge.sel >= len(m.knowledge.facts) {
		return m
	}

	f := m.knowledge.facts[m.knowledge.sel]
	m.knowledge.editing, m.knowledge.field = true, factPhrase
	m.knowledge.in[factPhrase] = newInput(f.Phrase)
	m.knowledge.in[factCheck] = newInput(f.Check)

	return m
}

// newFact opens an empty line, scoped to the repository being worked in.
//
// Most facts are written in the supervisor, mid-conversation, which is where
// somebody is when they think of one. This is for the one they think of while
// reading the others.
func (m Model) newFact() Model {
	if m.opts.ReplaceFact == nil {
		return m
	}

	m.knowledge.facts = append(m.knowledge.facts, knowledge.Fact{
		Scope:  m.hereScope(),
		Source: knowledge.Human,
		At:     time.Now().UTC(),
	})
	m.knowledge.sel = len(m.knowledge.facts) - 1
	m.knowledge.editing, m.knowledge.field = true, factPhrase
	m.knowledge.in[factPhrase] = newInput("")
	m.knowledge.in[factCheck] = newInput("")

	return m
}

// hereScope is what a fact written on this screen is about: the one
// repository on the board, and everything when there is more than one to
// choose between. Choosing one for somebody is how a rule ends up on the
// wrong project.
func (m Model) hereScope() knowledge.Scope {
	if len(m.board.RepoList) == 1 {
		return knowledge.Scope{Kind: knowledge.Repo, Repo: m.board.RepoList[0].Path}
	}

	return knowledge.Scope{Kind: knowledge.General}
}

// The two directions a fact can be moved in.
const (
	wider    = -1
	narrower = 1
)

// moveFact widens a fact to everything, or narrows it back to the repository.
//
// It is the common correction: a rule written in the supervisor is about the
// repository being worked in by default, and then turns out to be true
// everywhere. The language level is not on this ladder — a fact is about a
// language because somebody said so, not because it drifted there.
func (m Model) moveFact(dir int) Model {
	if m.opts.ReplaceFact == nil || m.knowledge.sel >= len(m.knowledge.facts) {
		return m
	}

	was := m.knowledge.facts[m.knowledge.sel]

	now := was
	switch {
	case dir == wider && was.Scope.Kind == knowledge.Repo:
		now.Scope = knowledge.Scope{Kind: knowledge.General}
	case dir == narrower && was.Scope.Kind == knowledge.General:
		here := m.hereScope()
		if here.Kind != knowledge.Repo {
			return m.say(m.opts.Words.T("knowledge.no_repo_here",
				"there is more than one repository here; say which with the supervisor"))
		}

		now.Scope = here
	default:
		return m
	}

	if err := m.opts.ReplaceFact(was, now); err != nil {
		return m.say(err.Error())
	}

	return m.syncKnowledge()
}

// editingKey is every key while a fact is being corrected.
func (m Model) editingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case tea.KeyEscape:
		m.knowledge.editing = false
		return m, nil
	case tea.KeyEnter:
		return m.saveFact(), nil
	case tea.KeyTab:
		m.knowledge.field = (m.knowledge.field + 1) % factFields
		return m, nil
	case tea.KeyBackspace:
		return m.factEdit(func(in *input) { in.backspace() }), nil
	case tea.KeyDelete:
		return m.factEdit(func(in *input) { in.deleteForward() }), nil
	case tea.KeyLeft:
		return m.factEdit(func(in *input) { in.moveBy(-1) }), nil
	case tea.KeyRight:
		return m.factEdit(func(in *input) { in.moveBy(1) }), nil
	case tea.KeyHome:
		return m.factEdit((*input).lineStart), nil
	case tea.KeyEnd:
		return m.factEdit((*input).lineEnd), nil
	}

	if msg.Text != "" {
		return m.factEdit(func(in *input) { in.insert(msg.Text) }), nil
	}

	return m, nil
}

// factEdit does something to the field being typed into.
func (m Model) factEdit(do func(*input)) Model {
	in := m.knowledge.in[m.knowledge.field]
	do(&in)
	m.knowledge.in[m.knowledge.field] = in

	return m
}

// saveFact writes the correction and closes the line.
//
// The fact it replaces travels with it, because the file is named after the
// sentence when nothing else names it: saving alone would leave the old copy
// behind, still told and still refusing work.
//
// A sentence emptied is refused rather than written. A fact with nothing in
// it says nothing, and deleting one is not something this gesture does.
func (m Model) saveFact() Model {
	was := m.knowledge.facts[m.knowledge.sel]

	now := was
	now.Phrase = strings.TrimSpace(m.knowledge.in[factPhrase].val)
	now.Check = strings.TrimSpace(m.knowledge.in[factCheck].val)

	if now.Phrase == "" {
		return m.say(m.opts.Words.T("knowledge.needs_words", "a fact with no sentence says nothing"))
	}

	m.knowledge.editing = false

	if err := m.opts.ReplaceFact(was, now); err != nil {
		return m.say(err.Error())
	}

	return m.syncKnowledge()
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

// factCount is how many facts the header's chip says there are.
//
// It reads what was last loaded and never the port: the header is drawn on
// every frame, and the port walks every repository on the board. What loads
// it is the first board that arrives and anything that writes a fact.
func (m Model) factCount() int {
	return len(m.knowledge.facts)
}
