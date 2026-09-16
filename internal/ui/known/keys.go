package known

// Every key this screen answers, and what is deliberately not among them.

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// key is the dispatch itself.
func (s State) key(msg tea.KeyPressMsg, e Env) (State, Out) {
	if s.reading {
		return s.readingKey(msg, e)
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
	case msg.Code == tea.KeyEnter:
		return s.chosen(e)
	case msg.Code == 'k' || msg.Code == 'K':
		return s.keepAsSaid(e)
	case msg.Code == 'd' || msg.Code == 'D':
		return s.dropSaid(e)
	case msg.Code == 'n' || msg.Code == 'N':
		return s.newFact(e), Out{}
	case msg.Code == 'p' || msg.Code == 'P':
		return s.pauseFact(e), Out{}
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
