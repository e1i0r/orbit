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

// editFact opens what the cursor is on, with what it already says in the
// line.
//
// The same two fields for both halves of the screen: a fact is corrected far
// more often than it is rewritten, and a sentence in the tray is accepted by
// correcting it more often than by agreeing with it word for word. A
// sentence has no check yet, because nobody has been asked for one.
func (s State) editFact(e Env) State {
	// The place line opens with the folder the work was in, so that
	// agreeing with it is enter and disagreeing is typing over it.
	if one, waiting := s.onSaid(); waiting {
		return s.typeInto(one.Text, "", one.Where)
	}

	f, ok := s.onFact()
	if e.Replace == nil || !ok {
		return s
	}

	return s.typeInto(f.Phrase, f.Check, f.Scope.Path)
}

// typeInto puts the three fields up with what is already in them, and the
// caret in the first.
func (s State) typeInto(phrase, check, where string) State {
	s.editing, s.field = true, factPhrase
	s.in[factPhrase] = typing.New(phrase)
	s.in[factCheck] = typing.New(check)
	s.in[factWhere] = typing.New(where)

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
	s.sel = s.last()

	return s.typeInto("", "", "")
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

// saveFact writes what was typed and closes the line: a sentence in the tray
// becomes a fact, and a fact already written is corrected in place.
//
// The fact a correction replaces travels with it, because the file is named
// after the sentence when nothing else names it: saving alone would leave
// the old copy behind, still told and still refusing work.
//
// A sentence emptied is refused rather than written. A fact with nothing in
// it says nothing, and deleting one is not something this gesture does.
func (s State) saveFact(e Env) (State, Out) {
	phrase := strings.TrimSpace(s.in[factPhrase].Val)
	if phrase == "" {
		return s, said(e.Words.T("knowledge.needs_words", "a fact with no sentence says nothing"))
	}

	check := strings.TrimSpace(s.in[factCheck].Val)
	where := strings.TrimSpace(s.in[factWhere].Val)
	s.editing = false

	if one, waiting := s.onSaid(); waiting {
		return s.keepWith(one, phrase, check, where, e)
	}

	was, ok := s.onFact()
	if !ok {
		return s, Out{}
	}

	now, moved := was, was.Scope
	now.Phrase, now.Check = phrase, check

	// A path typed into a fact that is about no checkout has nowhere to be
	// relative to, so it is refused rather than filed against a repository
	// picked for somebody.
	switch {
	case where == "":
		if moved.Kind == knowledge.Dir || moved.Kind == knowledge.File {
			now.Scope = knowledge.Scope{Kind: knowledge.Repo, Repo: moved.Repo}
		}
	case moved.Repo == "":
		return s, said(e.Words.T("knowledge.no_repo_for_path",
			"this one is about no checkout, so there is nothing for {path} to be inside",
			about("path", where)))
	default:
		at, err := knowledge.At(moved.Repo, where)
		if err != nil {
			return s, said(err.Error())
		}

		now.Scope = at
	}

	if err := e.Replace(was, now); err != nil {
		return s, said(err.Error())
	}

	return s.Sync(e), Out{}
}
