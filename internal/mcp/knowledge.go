package mcp

// The two tools that read what Orbit knows and offer it something new.
//
// This is the door that matters most for the store growing on its own. The
// supervisor's line is where a person says a rule; this is where the agent
// that just hit a wall offers what it found, mid-task, so that the next run
// against that code can be told rather than finding out again.
//
// Offered and not written. It used to write straight through, on the trade
// that a fact waiting for a person is a fact the next run does not have —
// and the price turned out to be higher than that. A fact carries a check,
// and a fact with a check is a gate: every phase of every future run in that
// repository executes that command and is sent back when it fails. A check
// that is wrong, or slow, is an hour of a task spent on something nobody
// agreed to.
//
// So it goes in the same tray as everything a person says, and waits the
// same way. One flow: a rule that holds for me and not for the model is not
// a rule. What is lost is said out loud in the answer — between the agent
// finding it and somebody reading it, the next run is not told.

import (
	"fmt"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/learn"
)

// learn offers a rule about the code the task is worked in, and it waits in
// the tray until somebody says yes.
//
// The repository is the task's own and never an argument. A caller that
// named its own scope could offer a rule about everywhere, and everywhere is
// the prompt of every phase of every run of every project on this machine —
// which is not something anybody agreed to by asking an agent to fix a bug.
// It is the rule openTaskRepo already keeps for every other tool here: act
// on the checkout the record was folded from, and no other.
//
// The place is checked here rather than when it is kept. The agent is the
// one that can fix a path it typed wrong, and it is still holding the
// context to do it; a person reading the tray a day later is not.
func (sn Session) learn(args map[string]any) CallToolResult {
	phrase := strings.TrimSpace(stringArg(args, "phrase"))
	if phrase == "" {
		return refuse(fmt.Errorf("a rule needs a sentence: what is true about this code"))
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
		return refuse(fmt.Errorf("task %s is against no repository, and what is offered here "+
			"is offered about one", t.ID))
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

	err = learn.Propose(sb.store, learn.Said{
		At: time.Now().UTC(), Text: phrase,
		By: learn.AModel, About: t.ID, Repo: t.RepoPath, Path: scope.Path,
	})
	if err != nil {
		return refuse(err)
	}

	// What did not happen, said out loud. An agent told "written down"
	// carries on believing the next run is already warned, plans around a
	// rule nothing is enforcing, and finds out the hard way — so the answer
	// says where the sentence is and what it is not doing yet.
	return done("offered about %s, %s. It is waiting in `orbit rules` for somebody to keep it, "+
		"so no run is told it yet.", t.Repo, factWhere(scope))
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
