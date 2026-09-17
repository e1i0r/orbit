package known

// Deciding about a rule with everything it has put you through in front of
// you.
//
// The screen above is read in the middle of something else, and what it
// offers there is cheap and reversible: pause it, and come back. This is the
// coming back. It is opened on purpose, it shows what happened before it
// offers anything, and it is the only place a rule's fate is decided.
//
// There is no score. A rule that works perfectly never stops anything — the
// model reads it and obeys it — so "it stopped the work zero times" means two
// opposite things and no number tells them apart. What is shown is the
// friction, in sentences, and the friction is all gestures of yours.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// openDetail opens the rule under the cursor, with everything about it.
func (s State) openDetail(e Env) State {
	if _, ok := s.onFact(); !ok {
		return s
	}

	s.reading = true

	return s
}

// readingKey is every key while a rule is open.
//
// Four decisions and a way out. Correcting reuses the line the screen already
// types into, because a rule said better and a rule narrowed are the same
// edit — and the third field of that line is where it goes.
func (s State) readingKey(msg tea.KeyPressMsg, e Env) (State, Out) {
	if s.editing {
		return s.editingKey(msg, e)
	}

	switch msg.Code {
	case tea.KeyEscape:
		s.reading, s.deep, s.forgetting = false, 0, false
		return s, Out{}
	case tea.KeyUp:
		s.deep = max(s.deep-1, 0)
		return s.letGo(), Out{}
	case tea.KeyDown:
		s.deep++
		return s.letGo(), Out{}
	case 'c', 'C':
		return s.letGo().correctFact(e), Out{}
	case 'o', 'O':
		return s.decideAgainst(e)
	case 'u', 'U':
		return s.haveItApplyAgain(e)
	case 'f', 'F':
		return s.forgetFact(e)
	}

	return s.letGo(), Out{}
}

// decideAgainst switches the rule off: it stays where it is, and nothing is
// told it.
//
// Offered here and nowhere else, because this is the one place the evidence
// is already in front of whoever presses it. Disagreeing with a rule and
// losing the record that it existed are different things, and only the first
// of them happens.
func (s State) decideAgainst(e Env) (State, Out) {
	return s.stand(e, knowledge.Off, e.Words.T("knowledge.switched_off",
		"it is off; it stays where it is and nothing is told it"))
}

// haveItApplyAgain is the answer to both ways a rule got here: paused on
// purpose, or skipped once and left waiting.
func (s State) haveItApplyAgain(e Env) (State, Out) {
	return s.stand(e, knowledge.Active, e.Words.T("knowledge.applies_again",
		"it applies again, and nothing is waiting on it"))
}

// stand puts the rule where the reader decided, and stops asking about it.
//
// The sentence arrives already said rather than as a key to look up: a key
// held in a variable is a key nothing can check against the other language,
// and a screen that silently lost half its Spanish is exactly what that check
// is for.
func (s State) stand(e Env, where knowledge.State, sentence string) (State, Out) {
	was, ok := s.onFact()
	if e.Replace == nil || !ok {
		return s, Out{}
	}

	now := was
	now.State, now.Why, now.Review = where, "", false

	if err := e.Replace(was, now); err != nil {
		return s, said(err.Error())
	}

	s.reading = false

	return s.Sync(e), said(sentence)
}

// story is what the open rule has put you through, and nothing at all when
// the window was built without a store to read it out of.
func (s State) story(e Env) []string {
	f, ok := s.onFact()
	if e.Story == nil || !ok {
		return nil
	}

	return e.Story(f)
}

// chosen is the row under the cursor being opened, which is the second of
// the pointer's two clicks and nothing a key does not also do.
//
// A rule opens its review, where its fate is decided with the evidence in
// front of you. A sentence in the tray opens in the line, because correcting
// it is how most of them are accepted — and neither gesture decides anything
// that escape does not take back.
func (s State) chosen(e Env) (State, Out) {
	if _, waiting := s.onSaid(); waiting {
		return s.editSaid(e), Out{}
	}

	return s.openDetail(e), Out{}
}
