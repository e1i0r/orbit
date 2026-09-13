package mcp

// The two tools that read and write what Orbit knows.
//
// This is the door that matters most for the store growing on its own. The
// supervisor's line is where a person writes a fact; this is where the agent
// that just hit a wall writes down what it found, mid-task, so the next run
// against that code is told before it starts rather than finding out again.
//
// Which is why a fact written here comes from the record and not from a
// person: the screen that lists facts says where each one came from, and the
// whole of why one can be trusted is that it can be traced back.
//
// It is written straight through rather than waiting to be agreed with, and
// that is the trade this tool exists for — a fact that waits for a person is
// a fact the next run does not have. Two things make it a fair trade, and
// both are below: it can only be about the repository the task is worked in,
// and it says where it came from.

import (
	"fmt"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// learn writes down a fact about the code the task is worked in.
//
// The repository is the task's own and never an argument. A caller that
// named its own scope could write a fact about everywhere, and everywhere is
// the prompt of every phase of every run of every project on this machine —
// which is not something anybody agreed to by asking an agent to fix a bug.
// It is the rule openTaskRepo already keeps for every other tool here: act
// on the checkout the record was folded from, and no other.
func (sn Session) learn(args map[string]any) CallToolResult {
	phrase := strings.TrimSpace(stringArg(args, "phrase"))
	if phrase == "" {
		return refuse(fmt.Errorf("a fact needs a sentence: what is true about this code"))
	}

	sb, err := sn.readBoard()
	if err != nil {
		return refuse(err)
	}

	defer sb.close()

	t, err := findTask(sb.board, stringArg(args, "task_id"))
	if err != nil {
		return refuse(err)
	}

	if t.RepoPath == "" {
		return refuse(fmt.Errorf("task %s is against no repository, and what is written here "+
			"is written about one", t.ID))
	}

	// The path the agent named, inside its own task's checkout and nowhere
	// else. An agent that just hit something knows better than anybody
	// which file it was in, and a rule filed against the whole project when
	// it was true of one folder is a rule that will be skipped everywhere
	// else until somebody narrows it by hand.
	scope, err := knowledge.At(t.RepoPath, stringArg(args, "path"))
	if err != nil {
		return refuse(err)
	}

	f := knowledge.Fact{
		Scope:  scope,
		Source: knowledge.FromRecord,
		Phrase: phrase,
		Stops:  boolArg(args, "stops"),
		Check:  strings.TrimSpace(stringArg(args, "check")),
		Ref:    t.ID,
		At:     time.Now().UTC(),
	}

	where, err := knowledge.NewStore(sb.store.Root()).Save(f)
	if err != nil {
		return refuse(err)
	}

	// Said out loud rather than left to be discovered: a rule asked to stop
	// with no check does not stop, and an agent told "written down" would
	// carry on believing a gate is now watching for it. The repository is
	// named for the same reason — one told only "written down" carries on
	// believing it wrote something everybody would be told.
	if f.Stops && f.Action() != knowledge.Stops {
		return done("written down about %s at %s. It has no check, so it is told and not enforced: "+
			"give it a command that exits non-zero when the rule is broken to make a gate of it.",
			t.Repo, where)
	}

	return done("written down about %s at %s. Every run against %s is told, and nothing "+
		"outside it.", t.Repo, where, factWhere(f.Scope))
}

// knowledgeOf answers what Orbit knows, for an agent that would rather ask
// before it plans than find out afterwards.
func (sn Session) knowledgeOf(args map[string]any) CallToolResult {
	s, err := sn.open()
	if err != nil {
		return refuse(err)
	}

	defer func() { _ = s.Close() }() //nolint:errcheck // the answer is already made

	facts, err := knowledge.NewStore(s.Root()).Load(strings.TrimSpace(stringArg(args, "repo")))
	if err != nil {
		return refuse(err)
	}

	facts = knowledge.InScope(facts)
	if len(facts) == 0 {
		return done("nothing has been written down about this code yet")
	}

	var b strings.Builder

	for _, f := range facts {
		fmt.Fprintf(&b, "- %s%s: %s\n", stopsMark(f), factWhere(f.Scope), f.Phrase)
	}

	return done("%s", strings.TrimRight(b.String(), "\n"))
}

// stopsMark says which facts the gate will refuse work over, because being
// advised and being sent back are different instructions.
func stopsMark(f knowledge.Fact) string {
	if f.Action() == knowledge.Stops {
		return "[stops] "
	}

	return ""
}

// factWhere is how far a fact reaches, in the words the tool answers with.
func factWhere(s knowledge.Scope) string {
	switch s.Kind {
	case knowledge.General:
		return "everywhere"
	case knowledge.Language:
		return "in " + s.Lang
	case knowledge.Repo:
		return "in this repository"
	case knowledge.Symbol:
		return "in " + s.Path + "#" + s.Symbol
	default:
		return "in " + s.Path
	}
}
