package known

// Taking a rule off the disk, from the one screen where the evidence is
// already in front of whoever presses it.
//
// Offered only for a rule nothing has happened to, and that is the whole of
// the design. A rule somebody decided and stopped wanting is switched off:
// it stays where it is, nothing is told it, and what it put you through is
// still readable. A rule that never should have been written — a duplicate
// of one already there, a test rule, one that came out wrong — is not that,
// and leaving it in the list for ever is a screen that fills with things
// nobody meant.
//
// The key is absent rather than present and refusing. A control that says no
// is a control somebody goes looking for a way to force.

import (
	"github.com/e1i0r/orbit/internal/knowledge"
)

// didNothing is whether the open rule has a history, read off the same story
// the screen is already showing.
//
// The screen's own reading and not a second question to the port: what is
// under "what it has put you through" is exactly what decides this, and two
// answers to one question is how a button comes to disagree with the text
// above it.
func (s State) didNothing(e Env) bool {
	if e.Forget == nil {
		return false
	}

	if _, ok := s.onFact(); !ok {
		return false
	}

	return len(s.story(e)) == 0
}

// forgetFact is the key, in two presses.
//
// The first arms it and the second does it, which is the confirmation every
// irreversible gesture in this window has. A rule is gone for good, and a
// single keystroke that loses something is a keystroke somebody hits by
// accident on the way to another one.
func (s State) forgetFact(e Env) (State, Out) {
	f, ok := s.onFact()
	if !s.didNothing(e) || !ok {
		return s, Out{}
	}

	if !s.forgetting {
		s.forgetting = true

		return s, said(e.Words.T("knowledge.forget_sure",
			"press it again to forget this rule for good; it has done nothing, so nothing is lost"))
	}

	return s.forgotten(f, e)
}

// forgotten does it, and puts the reader back in the list.
func (s State) forgotten(f knowledge.Rule, e Env) (State, Out) {
	if err := e.Forget(f); err != nil {
		s.forgetting = false

		return s, said(err.Error())
	}

	s.reading, s.forgetting = false, false

	return s.Sync(e), said(e.Words.T("knowledge.forgotten",
		"it is gone. It had done nothing, so there was nothing to lose"))
}

// letGo disarms the key, for every other gesture on this screen. A press
// armed and then left armed is a rule one keystroke from being lost while
// somebody reads something else.
func (s State) letGo() State {
	s.forgetting = false

	return s
}
