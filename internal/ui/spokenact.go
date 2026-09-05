package ui

// Doing what was typed into the supervisor, once spoken.go has said what it
// was. The window writes nothing itself: each of these hands the sentence to
// a port and reports what came back.

import "strings"

// act carries out one gesture and answers with what to say about it.
func (m Model) act(said spoken) Model {
	switch said.Kind {
	case saidRule, saidAware:
		return m.learn(said)
	case saidNote:
		return m.noteOn(said)
	default:
		// A gesture nobody finished typing. Saying nothing back would look
		// like the window had swallowed it.
		return m.say(m.opts.Words.T("supervisor.said_nothing",
			"that gesture has nothing after it; write what you want remembered"))
	}
}

// learn writes down a fact about the code.
//
// A rule that stops the work needs a check to enforce it, and one typed in a
// sentence has none — so it is written as a fact that warns, and the window
// says so rather than letting somebody believe a gate is now watching for
// them. The check is added afterwards, where a fact can be edited.
func (m Model) learn(said spoken) Model {
	p := m.opts.Words
	if m.opts.Learn == nil {
		return m.say(p.T("supervisor.cannot_learn", "this window cannot write down what it knows"))
	}

	stops := said.Kind == saidRule
	if err := m.opts.Learn(stops, said.Scope, m.repoForFact(), said.Phrase); err != nil {
		return m.say(err.Error())
	}

	m = m.remember(said, stops).syncSupervisor()

	if stops {
		return m.say(p.T("supervisor.learned_rule",
			"written down for {where}. It has no check yet, so it is said and not enforced: add one to make a gate of it.",
			about("where", m.whereFact(said))))
	}

	return m.say(p.T("supervisor.learned_aware",
		"written down for {where}", about("where", m.whereFact(said))))
}

// whereFact is the scope in the words the operator used, for the sentence
// that confirms what was written.
func (m Model) whereFact(said spoken) string {
	p := m.opts.Words

	switch said.Scope {
	case "":
		if repo := m.repoForFact(); repo != "" {
			return shortRepo(repo)
		}

		return p.T("supervisor.scope_general", "everything")
	case "general":
		return p.T("supervisor.scope_general", "everything")
	default:
		return said.Scope
	}
}

// repoForFact is the repository a fact with no scope is about: the one the
// task under the cursor is worked in, and otherwise the only one the board
// has. A board of several repositories with nothing selected cannot answer,
// and a fact that cannot say where it applies is written as a general one —
// which the sentence above says out loud.
func (m Model) repoForFact() string {
	if r, ok := m.selected(); ok && r.task.RepoPath != "" {
		return r.task.RepoPath
	}

	if len(m.board.RepoList) == 1 {
		return m.board.RepoList[0].Path
	}

	return ""
}

// shortRepo is a repository by its last segment, which is what a reader
// calls it.
func shortRepo(path string) string {
	parts := strings.Split(strings.TrimRight(path, "/"), "/")

	return parts[len(parts)-1]
}

// noteOn puts a line in one task's notes.
func (m Model) noteOn(said spoken) Model {
	p := m.opts.Words
	if m.opts.NoteTask == nil {
		return m.say(p.T("supervisor.cannot_note", "this window cannot write notes on a task"))
	}

	if err := m.opts.NoteTask(said.Task, said.Phrase); err != nil {
		return m.say(err.Error())
	}

	return m.say(p.T("supervisor.noted_on", "noted on {task}", about("task", said.Task)))
}

// remember puts the fact in the thread as well as in the store.
//
// A rule written in a conversation has to stay in the conversation. Without
// this the gesture wrote a file somewhere, flashed a sentence at the foot of
// the screen for twenty seconds, and left the thread exactly as it was — so
// the one place the operator was looking had no record that anything had
// happened. It is also what the supervisor reads next time: a model that
// cannot see a rule was written will keep working as though it was not.
func (m Model) remember(said spoken, stops bool) Model {
	if m.opts.RecordSupervisor == nil {
		return m
	}

	word := awareWord
	if stops {
		word = ruleWord
	}

	line := word + " " + m.whereFact(said) + ": " + said.Phrase
	if err := m.opts.RecordSupervisor("operator", "tui", line); err != nil {
		return m.say(err.Error())
	}

	return m
}
