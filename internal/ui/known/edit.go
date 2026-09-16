package known

// Opening the form, and writing down what was typed into it.

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/typing"
)

// editSaid opens a sentence from the tray with what it already says in the
// form.
//
// A sentence is accepted by correcting it more often than by agreeing with
// it word for word, which is the whole reason the tray asks instead of
// telling. The place opens with the folder the work was in, so that agreeing
// is enter and disagreeing is walking the options.
func (s State) editSaid(e Env) State {
	one, waiting := s.onSaid()
	if e.Keep == nil || !waiting {
		return s
	}

	return s.typeInto(one.Text, "", one.Where, false)
}

// pauseFact opens the one question a pause asks: what it is being paused for.
//
// The cheapest thing this screen can do to a rule, and the only reversible
// one. Switching a rule off decides its fate; pausing it says not now and
// asks somebody to come back — which is what you mean when a rule stops you
// in the middle of something else.
//
// The reason is the whole difference between the two. A pause with no reason
// is a switch under another name, and the reason is what somebody reads when
// they come back: the only thing that will tell them whether it made sense.
func (s State) pauseFact(e Env) State {
	f, ok := s.onFact()
	if e.Replace == nil || !ok || f.State == knowledge.Paused {
		return s
	}

	s.pausing = true

	return s.typeInto("", "", "", false)
}

// correctFact opens the rule under the cursor with everything it holds.
func (s State) correctFact(e Env) State {
	f, ok := s.onFact()
	if e.Replace == nil || !ok {
		return s
	}

	return s.typeInto(f.Phrase, f.Check, f.Scope.Path, false)
}

// newFact opens an empty form, filed against the repository being worked in.
//
// Most rules are written in the supervisor, mid-conversation, which is where
// somebody is when they think of one. This is for the one they think of
// while reading the others.
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

	return s.typeInto("", "", "", true)
}

// typeInto puts the form up with what is already in it, and the cursor on
// the first row.
func (s State) typeInto(phrase, check, where string, fresh bool) State {
	s.editing, s.fresh, s.field = true, fresh, rowPhrase
	s.in[factPhrase] = typing.New(phrase)
	s.in[factCheck] = typing.New(check)
	s.in[factWhere] = typing.New(where)

	return s
}

// hereScope is what a rule written on this screen is about: the one
// repository the window is on, and everything when there is more than one to
// choose between. Choosing one for somebody is how a rule ends up on the
// wrong project.
func hereScope(e Env) knowledge.Scope {
	if e.Repo != "" {
		return knowledge.Scope{Kind: knowledge.Repo, Repo: e.Repo}
	}

	return knowledge.Scope{Kind: knowledge.General}
}

// editingKey is every key while the form is up.
func (s State) editingKey(msg tea.KeyPressMsg, e Env) (State, Out) {
	rows := s.rows2(e)
	here := max(s.at(rows), 0)

	if r, up := s.picked(e); up {
		return s.pickingKey(msg, r), Out{}
	}

	switch msg.Code {
	case tea.KeyEscape:
		return s.shut(), Out{}
	case tea.KeyEnter:
		return s.pressed(rows[here], e)
	case tea.KeyTab, tea.KeyDown:
		s.field = rows[(here+1)%len(rows)].which
		return s, Out{}
	case tea.KeyUp:
		s.field = rows[(here+len(rows)-1)%len(rows)].which
		return s, Out{}
	case tea.KeyLeft:
		return s.sideways(rows[here], -1, e), Out{}
	case tea.KeyRight:
		return s.sideways(rows[here], 1, e), Out{}
	case tea.KeyBackspace:
		return s.factEdit(rows[here], func(in *typing.Field) { in.Backspace() }), Out{}
	case tea.KeyDelete:
		return s.factEdit(rows[here], func(in *typing.Field) { in.DeleteForward() }), Out{}
	case tea.KeyHome:
		return s.factEdit(rows[here], (*typing.Field).LineStart), Out{}
	case tea.KeyEnd:
		return s.factEdit(rows[here], (*typing.Field).LineEnd), Out{}
	}

	if msg.Text != "" {
		return s.factEdit(rows[here], func(in *typing.Field) { in.Insert(msg.Text) }), Out{}
	}

	return s, Out{}
}

// pickingKey is every key while one row's list of options is open. It is a
// list with one row chosen and nothing else, so it answers what every such
// list in this window answers and nothing more.
func (s State) pickingKey(msg tea.KeyPressMsg, r aRow) State {
	switch msg.Code {
	case tea.KeyEscape:
		s.picking = false
	case tea.KeyEnter:
		return s.takePick(r)
	case tea.KeyUp:
		s.pick = max(s.pick-1, 0)
	case tea.KeyDown:
		s.pick = min(s.pick+1, len(r.options)-1)
	default:
		next, moved := s.pickKey(msg.Code, r)
		if moved {
			return next
		}
	}

	return s
}

// sideways is ←→ on a row: it walks the options of a row that has them, and
// moves the caret inside one that is only typed into.
func (s State) sideways(r aRow, d int, e Env) State {
	if len(r.options) > 0 {
		return s.walk(r, d, e)
	}

	return s.factEdit(r, func(in *typing.Field) { in.MoveBy(d) })
}

// pressed is enter on a row: it does the button, and otherwise saves — the
// form has one obvious answer and enter is it, wherever the cursor is.
func (s State) pressed(r aRow, e Env) (State, Out) {
	if r.which == rowCancel {
		return s.shut(), Out{}
	}

	// A row with more answers than fit beside it opens them instead, which
	// is what enter means on the diff's file selector and on every other
	// list in this window.
	if len(r.options) >= pickFrom {
		return s.openPicker(r), Out{}
	}

	return s.saveFact(e)
}

// shut closes the form without writing anything.
func (s State) shut() State {
	s.editing, s.pausing, s.fresh, s.picking, s.deep = false, false, false, false, 0

	return s
}

// factEdit does something to the line behind the row, and nothing at all to
// a row that has no line.
func (s State) factEdit(r aRow, do func(*typing.Field)) State {
	if r.button != "" || r.which == rowDoes {
		return s
	}

	in := s.in[r.typed]
	do(&in)
	s.in[r.typed] = in

	return s
}

// savePause stops the rule under the cursor applying, and sends it to be
// looked at again.
//
// A reason emptied is refused rather than written, for the reason a sentence
// emptied is: pausing with nothing typed is the switch beside it, and the
// switch is not what was asked for.
func (s State) savePause(e Env) (State, Out) {
	why := strings.TrimSpace(s.in[factPhrase].Val)
	if why == "" {
		return s, said(e.Words.T("knowledge.pause_needs_why",
			"say what you are pausing it for; it is what you will read when you come back"))
	}

	was, ok := s.onFact()
	if !ok {
		return s, Out{}
	}

	now := was
	now.State, now.Why, now.Review = knowledge.Paused, why, true

	if err := e.Replace(was, now); err != nil {
		return s, said(err.Error())
	}

	s = s.shut()
	s.reading = false

	return s.Sync(e), said(e.Words.T("knowledge.paused",
		"it is paused, and waiting for you to decide about it"))
}

// saveFact writes what was typed and closes the form: a sentence in the tray
// becomes a rule, and a rule already written is corrected in place.
//
// The rule a correction replaces travels with it, because the file is named
// after the sentence when nothing else names it: saving alone would leave
// the old copy behind, still told and still refusing work.
func (s State) saveFact(e Env) (State, Out) {
	if s.pausing {
		return s.savePause(e)
	}

	phrase := strings.TrimSpace(s.in[factPhrase].Val)
	if phrase == "" {
		return s, said(e.Words.T("knowledge.needs_words", "a rule with no sentence says nothing"))
	}

	check := strings.TrimSpace(s.in[factCheck].Val)
	where := strings.TrimSpace(s.in[factWhere].Val)

	if one, waiting := s.onSaid(); waiting {
		return s.shut().keepWith(one, phrase, check, where, e)
	}

	return s.saveOver(phrase, check, where, e)
}

// saveOver writes the corrected rule over the one it came from.
func (s State) saveOver(phrase, check, where string, e Env) (State, Out) {
	was, ok := s.onFact()
	if !ok {
		return s, Out{}
	}

	now, moved := was, was.Scope
	now.Phrase, now.Check, now.Stops = phrase, check, check != ""

	// A path typed into a rule that is about no checkout has nowhere to be
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

	// Correcting a rule is a decision about it, so it stops waiting for
	// one — which is the whole reason somebody opened it.
	now.Review = false

	if err := e.Replace(was, now); err != nil {
		return s, said(err.Error())
	}

	return s.shut().Sync(e), Out{}
}
