package known

// Writing a fact down, and correcting one.
//
// Two fields and not the whole record. A fact's scope and its source are what
// make it traceable, and neither is something to retype — the source is where
// it came from, which nobody may edit, and the scope is moved with its own
// gesture rather than by typing a path.

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/typing"
)

// editFact opens the fact under the cursor, with what it already says in the
// line: a fact is corrected far more often than it is rewritten.
func (s State) editFact(e Env) State {
	if e.Replace == nil || s.sel >= len(s.facts) {
		return s
	}

	f := s.facts[s.sel]
	s.editing, s.field = true, factPhrase
	s.in[factPhrase] = typing.New(f.Phrase)
	s.in[factCheck] = typing.New(f.Check)

	return s
}

// newFact opens an empty line, scoped to the repository being worked in.
//
// Most facts are written in the supervisor, mid-conversation, which is where
// somebody is when they think of one. This is for the one they think of while
// reading the others.
func (s State) newFact(e Env) State {
	if e.Replace == nil {
		return s
	}

	s.facts = append(s.facts, knowledge.Fact{
		Scope:  hereScope(e),
		Source: knowledge.Human,
		At:     time.Now().UTC(),
	})
	s.sel = len(s.facts) - 1
	s.editing, s.field = true, factPhrase
	s.in[factPhrase] = typing.New("")
	s.in[factCheck] = typing.New("")

	return s
}

// hereScope is what a fact written on this screen is about: the one
// repository the window is on, and everything when there is more than one to
// choose between. Choosing one for somebody is how a rule ends up on the
// wrong project.
func hereScope(e Env) knowledge.Scope {
	if e.Repo != "" {
		return knowledge.Scope{Kind: knowledge.Repo, Repo: e.Repo}
	}

	return knowledge.Scope{Kind: knowledge.General}
}

// editingKey is every key while a fact is being corrected.
func (s State) editingKey(msg tea.KeyPressMsg, e Env) (State, Out) {
	switch msg.Code {
	case tea.KeyEscape:
		s.editing = false
		return s, Out{}
	case tea.KeyEnter:
		return s.saveFact(e)
	case tea.KeyTab:
		s.field = (s.field + 1) % factFields
		return s, Out{}
	case tea.KeyBackspace:
		return s.factEdit(func(in *typing.Field) { in.Backspace() }), Out{}
	case tea.KeyDelete:
		return s.factEdit(func(in *typing.Field) { in.DeleteForward() }), Out{}
	case tea.KeyLeft:
		return s.factEdit(func(in *typing.Field) { in.MoveBy(-1) }), Out{}
	case tea.KeyRight:
		return s.factEdit(func(in *typing.Field) { in.MoveBy(1) }), Out{}
	case tea.KeyHome:
		return s.factEdit((*typing.Field).LineStart), Out{}
	case tea.KeyEnd:
		return s.factEdit((*typing.Field).LineEnd), Out{}
	}

	if msg.Text != "" {
		return s.factEdit(func(in *typing.Field) { in.Insert(msg.Text) }), Out{}
	}

	return s, Out{}
}

// factEdit does something to the field being typed into.
func (s State) factEdit(do func(*typing.Field)) State {
	in := s.in[s.field]
	do(&in)
	s.in[s.field] = in

	return s
}

// saveFact writes the correction and closes the line.
//
// The fact it replaces travels with it, because the file is named after the
// sentence when nothing else names it: saving alone would leave the old copy
// behind, still told and still refusing work.
//
// A sentence emptied is refused rather than written. A fact with nothing in
// it says nothing, and deleting one is not something this gesture does.
func (s State) saveFact(e Env) (State, Out) {
	was := s.facts[s.sel]

	now := was
	now.Phrase = strings.TrimSpace(s.in[factPhrase].Val)
	now.Check = strings.TrimSpace(s.in[factCheck].Val)

	if now.Phrase == "" {
		return s, said(e.Words.T("knowledge.needs_words", "a fact with no sentence says nothing"))
	}

	s.editing = false

	if err := e.Replace(was, now); err != nil {
		return s, said(err.Error())
	}

	return s.Sync(e), Out{}
}
