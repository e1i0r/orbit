package known

// Every key this screen answers, and what is deliberately not among them.

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// key is the dispatch itself.
func (s State) key(msg tea.KeyPressMsg, e Env) (State, Out) {
	if s.reviewing {
		return s.reviewKey(msg, e)
	}

	if s.editing {
		return s.editingKey(msg, e)
	}

	switch {
	case msg.Code == tea.KeyEscape || key.Matches(msg, e.Keys.Back):
		return State{}, Out{Leave: true}
	case msg.Code == tea.KeyUp:
		return s.Move(-1, e), Out{}
	case msg.Code == tea.KeyDown:
		return s.Move(1, e), Out{}
	case msg.Code == tea.KeyPgUp:
		return s.Move(-pageRules, e), Out{}
	case msg.Code == tea.KeyPgDown:
		return s.Move(pageRules, e), Out{}
	case msg.Code == tea.KeyHome:
		return s.Move(-s.last(), e), Out{}
	case msg.Code == tea.KeyEnd:
		return s.Move(s.last(), e), Out{}
	case msg.Code == 'e' || msg.Code == 'E':
		return s.editSaid(e), Out{}
	case msg.Code == 'k' || msg.Code == 'K':
		return s.keepAsSaid(e)
	case msg.Code == 'd' || msg.Code == 'D':
		return s.dropSaid(e)
	case msg.Code == 'n' || msg.Code == 'N':
		return s.newFact(e), Out{}
	case msg.Code == 'p' || msg.Code == 'P':
		return s.pauseFact(e), Out{}
	case msg.Code == 'r' || msg.Code == 'R':
		return s.openReview(e), Out{}
	}

	return s, Out{}
}

// What this screen does not do any more, on purpose.
//
// Switching a rule off, rewording one and moving where it applies all decide
// a rule's fate, and this screen is glanced at in the middle of something
// else. They live in the review now — `r` — where the evidence is read
// before anything is decided.
//
// What is left here is reading, and pausing: cheap, reversible, and it asks
// the question rather than answering it.

// ordered is the facts in the order the screen draws them: the ones that
// belong to no repository first, then each repository's own.
//
// General first because it is what applies everywhere and what somebody
// looking for "why did it do that" checks before anything narrower.
func (s State) ordered() (rootless, owned []knowledge.Fact) {
	for _, f := range s.facts {
		if f.Scope.Kind == knowledge.General || f.Scope.Kind == knowledge.Language {
			rootless = append(rootless, f)
			continue
		}

		owned = append(owned, f)
	}

	return rootless, owned
}
