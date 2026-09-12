package known

// The tray: what you said that nobody has answered yet.
//
// You tell the supervisor "never push a pull request without the tests
// passing". That sentence is already the answer — it is about your code, in
// your words, and you meant it — so the only thing left is to be asked
// whether to keep it. This is where the asking happens.
//
// It sits above what Orbit already knows rather than below it. This is the
// one part of the screen with a question in it, and a question under two
// pages of facts is a question nobody answers.

import (
	"github.com/e1i0r/orbit/internal/knowledge"
)

// onSaid is the sentence under the cursor, and false once the cursor has
// moved past the tray into what Orbit already knows.
func (s State) onSaid() (Said, bool) {
	if s.sel >= len(s.waiting) {
		return Said{}, false
	}

	return s.waiting[s.sel], true
}

// onFact is the fact under the cursor, and false while the cursor is still
// in the tray.
func (s State) onFact() (knowledge.Fact, bool) {
	at := s.sel - len(s.waiting)
	if at < 0 || at >= len(s.facts) {
		return knowledge.Fact{}, false
	}

	return s.facts[at], true
}

// keepAsSaid keeps the sentence under the cursor word for word, which is the
// answer when there is nothing to correct.
func (s State) keepAsSaid(e Env) (State, Out) {
	one, waiting := s.onSaid()
	if !waiting {
		return s, Out{}
	}

	return s.keepWith(one, one.Text, "", e)
}

// keepWith writes it down as a fact of yours, and takes it out of the tray.
//
// The words are a parameter rather than the sentence's own because
// correcting is how most of these are accepted: what you meant is what you
// typed the second time, and being asked is worth nothing if the only
// answers are yes and no.
func (s State) keepWith(one Said, phrase, check string, e Env) (State, Out) {
	if e.Keep == nil {
		return s, Out{}
	}

	if err := e.Keep(one.At, phrase, check); err != nil {
		return s, said(err.Error())
	}

	return s.Sync(e), said(e.Words.T("knowledge.kept",
		"Orbit knows it now, and every phase will be told"))
}

// dropSaid says it was not a rule.
func (s State) dropSaid(e Env) (State, Out) {
	one, waiting := s.onSaid()
	if !waiting || e.Drop == nil {
		return s, Out{}
	}

	if err := e.Drop(one.At); err != nil {
		return s, said(err.Error())
	}

	return s.Sync(e), said(e.Words.T("knowledge.left_said",
		"left in the thread where you said it"))
}
