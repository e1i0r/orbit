package supervisor

// Doing what was typed into the supervisor, once spoken.go has said what it
// was. The screen writes nothing itself: each of these hands the sentence to
// a port and reports what came back.

import (
	"github.com/e1i0r/orbit/internal/ui/fact"
	"github.com/e1i0r/orbit/internal/ui/spoken"
)

// act carries out one gesture and answers with what to say about it.
func (s State) act(line spoken.Line, e Env) (State, Out) {
	switch line.Kind {
	case spoken.Rule, spoken.Aware:
		return s.learn(line, e)
	case spoken.Note:
		return s.noteOn(line, e)
	case spoken.Chats:
		return s.openConversationList().clearedLine(), Out{}
	case spoken.New:
		return s.startConversation(e)
	default:
		// A gesture nobody finished typing. Saying nothing back would look
		// like the window had swallowed it.
		return s, said(e.Words.T("supervisor.said_nothing",
			"that gesture has nothing after it; write what you want remembered"))
	}
}

// learn writes down a fact about the code.
//
// A rule that stops the work needs a check to enforce it, and one typed in a
// sentence has none — so it is written as a fact that warns, and the window
// says so rather than letting somebody believe a gate is now watching for
// them. The check is added afterwards, where a fact can be edited.
func (s State) learn(line spoken.Line, e Env) (State, Out) {
	p := e.Words
	if e.Learn == nil {
		return s, said(p.T("supervisor.cannot_learn", "this window cannot write down what it knows"))
	}

	stops := line.Kind == spoken.Rule
	if err := e.Learn(stops, line.Scope, e.Repo, line.Phrase); err != nil {
		return s, said(err.Error())
	}

	s, out := s.remember(line, stops, e)

	// Synced whether or not the journal had something to say. e.Learn has
	// already written the fact down by here, and returning early on a
	// record that would not take the line left the rule stored and missing
	// from the column that lists what Orbit knows.
	s = s.Sync(e)

	if out.Said != "" {
		return s, out
	}

	if stops {
		return s, said(p.T("supervisor.learned_rule",
			"written down for {where}. It has no check yet, so it is said and not enforced: add one to make a gate of it.",
			about("where", s.whereFact(line, e))))
	}

	return s, said(p.T("supervisor.learned_aware",
		"written down for {where}", about("where", s.whereFact(line, e))))
}

// whereFact is the scope in the words the operator used, for the sentence
// that confirms what was written.
//
// A fact with no scope is about the repository the window is on — the task
// under the cursor, or the only one the board has. A board of several with
// nothing selected cannot answer, and then the fact is written as a general
// one, which the sentence says out loud.
func (s State) whereFact(line spoken.Line, e Env) string {
	p := e.Words

	switch line.Scope {
	case "":
		if e.Repo != "" {
			return fact.Repo(e.Repo)
		}

		return p.T("supervisor.scope_general", "everything")
	case "general":
		return p.T("supervisor.scope_general", "everything")
	default:
		return line.Scope
	}
}

// noteOn puts a line in one task's notes.
func (s State) noteOn(line spoken.Line, e Env) (State, Out) {
	p := e.Words
	if e.Note == nil {
		return s, said(p.T("supervisor.cannot_note", "this window cannot write notes on a task"))
	}

	if err := e.Note(line.Task, line.Phrase); err != nil {
		return s, said(err.Error())
	}

	return s, said(p.T("supervisor.noted_on", "noted on {task}", about("task", line.Task)))
}

// remember puts the fact in the thread as well as in the store.
//
// A rule written in a conversation has to stay in the conversation. Without
// this the gesture wrote a file somewhere, flashed a sentence at the foot of
// the screen for twenty seconds, and left the thread exactly as it was — so
// the one place the operator was looking had no record that anything had
// happened. It is also what the supervisor reads next time: a model that
// cannot see a rule was written will keep working as though it was not.
func (s State) remember(line spoken.Line, stops bool, e Env) (State, Out) {
	if e.Record == nil {
		return s, Out{}
	}

	word := spoken.AwareWord
	if stops {
		word = spoken.RuleWord
	}

	text := word + " " + s.whereFact(line, e) + ": " + line.Phrase
	if err := e.Record(s.conversation, "operator", "tui", text); err != nil {
		return s, said(err.Error())
	}

	return s, Out{}
}
