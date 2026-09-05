package task

// What Orbit knows, written into the prompt.
//
// This is how a fact reaches the model at all, and it is deliberately the
// dullest mechanism available: Orbit already writes the prompt for every
// phase, so the sentences go in it. No hook in anybody's CLI, nothing for the
// operator to configure, and nothing that breaks when claude or codex ship
// next week — which they do. It arrives every time, rather than when a model
// remembers to ask.
//
// What it costs is precision. A hook fires on the file about to be edited and
// can carry the three sentences that are about that file; this carries
// everything known about the repository, because at the time the prompt is
// written nothing has been touched yet. With a few dozen facts that is a
// paragraph, which is a price worth paying for a delivery that never fails.

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
)

// knownIntro says what the section is and what the marks in it mean.
//
// The model is told where these came from because that is what makes them
// worth obeying: they are not this prompt's opinion, they are what the people
// and the runs before it established, and they outlive the engine reading
// them.
const knownIntro = "What Orbit has learned about this code, kept across sessions and engines. " +
	"Lines marked **stops** are refused by a gate that runs after you, not merely advised: " +
	"work that breaks one comes back.\n\n"

// whatIsKnown is the section, and nothing at all when nothing is known.
//
// An empty heading is worse than no heading: it is a question the model has
// to answer for itself — whether Orbit knows nothing about this code, or
// knows something this program dropped on the way.
func whatIsKnown(facts []knowledge.Fact) string {
	if len(facts) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString("\n## What Orbit knows\n\n")
	b.WriteString(knownIntro)

	for _, f := range facts {
		fmt.Fprintf(&b, "- %s%s%s\n", stopMark(f), about(f.Scope), strings.TrimSpace(f.Phrase))
	}

	return b.String()
}

// stopMark is what a fact that stops the work is written with. A sentence that
// advises and a sentence that will send the work back are different
// instructions, and a model can only act on the difference if it is drawn.
func stopMark(f knowledge.Fact) string {
	if f.Action() == knowledge.Stops {
		return "**stops** "
	}

	return ""
}

// about says where a fact applies, when that is narrower than everything.
//
// A repository-wide prompt carries facts of several scopes at once, so a
// sentence about one directory has to say which one — otherwise a rule about
// the ledger reads as a rule about the whole checkout, which is how a fact
// gets applied where it does not belong.
func about(s knowledge.Scope) string {
	switch s.Kind {
	case knowledge.Language:
		return "in " + s.Lang + ": "
	case knowledge.Dir, knowledge.File:
		return "in " + s.Path + ": "
	case knowledge.Symbol:
		return "in " + s.Path + "#" + s.Symbol + ": "
	default:
		return ""
	}
}

// knows is what Orbit has learned about the code this phase runs in.
//
// It is read from disk on each phase rather than once per run, because a
// phase can take minutes and somebody can write a rule while it does — the
// next phase should be told it. Both roots are read: the repository's own
// facts, which travel with it, and the general and per-language ones, which
// live in the state root and do not.
//
// A store that cannot be read costs the facts and not the run. The sentences
// are how a rule is explained; the gate is how one is enforced, and the gate
// reads its own copy. Failing the whole task because one file has a typo in
// it would trade a run for a paragraph.
func (r phaseRun) knows() []knowledge.Fact {
	return factsFor(r.store, r.task)
}

// factsFor reads both roots for one task and orders what it finds. It is the
// one door: the prompt says the sentences and the gates enforce the ones
// that can enforce themselves, and neither should be reading a different set
// from the other.
func factsFor(s *store.Store, t Task) []knowledge.Fact {
	if s == nil {
		return nil
	}

	facts, err := knowledge.NewStore(s.Root()).Load(t.Repo.Path)
	if err != nil {
		logger.Error("task/run", "%s: what orbit knows was not read: %v", t.ID, err)

		return nil
	}

	return knowledge.InScope(facts)
}
