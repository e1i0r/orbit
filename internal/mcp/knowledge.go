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

import (
	"fmt"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
)

// learn writes down a fact.
func (sn Session) learn(args map[string]any) CallToolResult {
	phrase := strings.TrimSpace(stringArg(args, "phrase"))
	if phrase == "" {
		return refuse(fmt.Errorf("a fact needs a sentence: what is true about this code"))
	}

	s, err := sn.open()
	if err != nil {
		return refuse(err)
	}

	defer func() { _ = s.Close() }() //nolint:errcheck // the answer is already made

	scope, err := factScope(args)
	if err != nil {
		return refuse(err)
	}

	f := knowledge.Fact{
		Scope:  scope,
		Source: knowledge.FromRecord,
		Phrase: phrase,
		Stops:  boolArg(args, "stops"),
		Check:  strings.TrimSpace(stringArg(args, "check")),
		Ref:    strings.TrimSpace(stringArg(args, "task_id")),
		At:     time.Now().UTC(),
	}

	where, err := knowledge.NewStore(s.Root()).Save(f)
	if err != nil {
		return refuse(err)
	}

	// Said out loud rather than left to be discovered: a rule asked to stop
	// with no check does not stop, and an agent told "written down" would
	// carry on believing a gate is now watching for it.
	if f.Stops && f.Action() != knowledge.Stops {
		return done("written down at %s. It has no check, so it is told and not enforced: "+
			"give it a command that exits non-zero when the rule is broken to make a gate of it.", where)
	}

	return done("written down at %s", where)
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

// factScope reads the scope out of what the tool was given.
//
// A repository is named by path and not guessed from where this server was
// started: a supervising model works across several, and a fact filed against
// the wrong one is a rule applied where nobody put it.
func factScope(args map[string]any) (knowledge.Scope, error) {
	lang := strings.TrimSpace(stringArg(args, "lang"))
	repo := strings.TrimSpace(stringArg(args, "repo"))

	switch {
	case lang != "" && repo != "":
		return knowledge.Scope{}, fmt.Errorf("a fact is about a language or about a repository, not both")
	case lang != "":
		return knowledge.Scope{Kind: knowledge.Language, Lang: lang}, nil
	case repo != "":
		return knowledge.Scope{Kind: knowledge.Repo, Repo: repo}, nil
	default:
		return knowledge.Scope{Kind: knowledge.General}, nil
	}
}
