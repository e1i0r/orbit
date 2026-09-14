package knowledge

// One fact: a sentence about some code, where it came from, and whether it
// merely says something or refuses to let the work through.

import (
	"fmt"
	"slices"
	"time"
)

// Source is where a fact came from, and there are four because there are
// four ways Orbit finds anything out. None of them is "the model thought so".
type Source int

const (
	// unsourced is the zero value, and it is not a source. A fact built by
	// somebody who forgot to say where it came from would otherwise pass as
	// having been read off the code, which is the one source nobody has to
	// justify — the mistake would look like the most trustworthy answer.
	unsourced Source = iota
	// FromCode is read off the map: "ledger only appends" because Write
	// inserts and nothing updates. It is regenerated rather than stored.
	FromCode
	// Human is somebody saying it, at a gate or in the supervisor.
	Human
	// FromRecord is a lesson: a gate rejected something, or an attempt
	// failed, and what happened became a fact with the scope of what the
	// task touched. This is the one that grows without anybody writing.
	FromRecord
	// FromProduction is an incident. Nothing reads these yet; the source
	// exists so the shape does not have to change when something does.
	FromProduction
	// FromDocs is read out of what the project already says about itself:
	// the CONTRIBUTING, the README, the docs, and the notes each engine
	// keeps in its own file.
	//
	// Its own source and not Human, even though a person wrote every word
	// of those. What a person typed into the supervisor they meant, now,
	// about this; what a CONTRIBUTING says is what somebody meant two years
	// ago and may have stopped meaning — and a reader deciding whether to
	// keep a rule is owed that difference. Ref carries the file and the
	// line, so they can go and look.
	FromDocs
)

// State is where a rule stands: whether it applies, and whether somebody
// stopped it applying.
//
// Three and not two. A rule used to be said or not said, and that one switch
// is what forced switching a rule off when what was needed was something
// else entirely — "skip this one while we get the coverage up" is not
// disagreeing with it.
type State int

const (
	// Active is confirmed and working. It is the zero value, so a fact
	// somebody wrote by hand with a header of two lines applies, which is
	// what they meant by writing it.
	Active State = iota
	// Paused is stopped by a person, with a reason written down. It is not
	// a switch: the reason is what they will read when they come back, and
	// the only thing that will tell them whether it made sense.
	Paused
	// Off is somebody deciding against it. It stays and stops being told:
	// disagreeing with a rule and losing the record that it existed are
	// different things.
	Off
)

// Action is what a fact does when the work reaches its scope.
type Action int

const (
	// Warns puts the sentence in front of the agent and lets it work.
	Warns Action = iota
	// Stops refuses the work at the gate.
	Stops
)

// A Fact is everything Orbit knows about one piece of code.
//
// Phrase is the only field written for a reader — it is what the agent is
// told and what a person sees in the Knowledge screen. The rest is what
// Orbit uses to decide when to say it and what to do about it.
type Fact struct {
	// ID is what this rule is called, for as long as it exists. It is
	// coined once, when Orbit first writes the fact down, and nothing
	// changes it after that — not correcting the sentence, not moving the
	// scope, not turning it off.
	//
	// It is here because everything else about a fact can change. The file
	// is named after the sentence, so rewording one renames it, and what
	// the record wrote down about the old name stops being findable. A
	// cycle that cannot say "this is the same thing you said differently
	// three weeks ago" is not a cycle: it is a pile of separate rules.
	//
	// Empty is allowed and is not an error. A fact somebody wrote by hand
	// has no id until Orbit writes it, and writing files by hand is half
	// the reason these are files.
	ID     string
	Scope  Scope
	Source Source
	// Phrase is the fact, in a sentence. It is what gets read.
	Phrase string
	// Stops is what the fact was asked to do. Whether it can is Action's
	// answer, not this field's: see there.
	Stops bool
	// Check is the command that says yes or no without opinion. A fact that
	// wants to stop is worth nothing without one.
	Check string
	// Ref names what the fact came out of — a task, a decision, an
	// incident — so that a reader can go and see for themselves.
	Ref string
	// At is when it entered, and Used is how many times it has been told.
	// Both are for the person deciding whether to keep it.
	At   time.Time
	Used int
	// State is where the rule stands: applying, paused, or switched off.
	State State
	// Why is the reason it was paused, in the words of whoever paused it.
	// A pause with no reason is a switch, and a switch is the thing this
	// was added to stop being the only answer.
	Why string
	// Review is a rule waiting for somebody to decide about it. It is
	// apart from the state because the two are different questions:
	// skipping a rule at a run leaves it applying and asks for a decision,
	// and pausing it stops it applying and asks for the same decision.
	// Folded together, one of those two would have to lie about whether
	// the rule is still in the prompt.
	Review bool
	// from is the file this fact was read out of, and empty for one that
	// has never been on disk.
	//
	// Unexported because nobody outside sets it and nobody should: it is
	// what the store saw, not something a screen decides. What it is for is
	// replacing — a file somebody wrote by hand is called whatever they
	// called it, and a replacement that worked out the old name from the
	// fact's own fields left that file behind, still told and still
	// refusing work.
	from string
}

// Tells is whether this rule reaches a phase at all: it is what is written
// into every prompt and what the gate refuses work over. Only an active rule
// does — a paused one is the reader saying not now, and one switched off is
// them saying no.
func (f Fact) Tells() bool { return f.State == Active }

// Action is what this fact actually does, which is not always what it was
// asked to do.
//
// Warning is a sentence and needs nothing else. Stopping is the gate refusing
// the work, and refusing needs something that answers yes or no without an
// opinion in it — a command, a pattern over the diff, a test that runs. A
// fact that asked to stop and brought no check would never fire while reading
// as though it would, so it warns, and the screen says so.
func (f Fact) Action() Action {
	if f.Stops && f.Check != "" {
		return Stops
	}

	return Warns
}

// Validate reports the first thing that would make a fact untrustworthy.
//
// Every fact has a source and a scope; without something behind it, it does
// not get in. That is not tidiness — a sentence in the agent's context that
// nobody can trace is indistinguishable from one the model made up, and the
// whole point of keeping this outside the model is that it can be traced.
func (f Fact) Validate() error {
	if f.Phrase == "" {
		return fmt.Errorf("a fact with no sentence says nothing")
	}

	if f.Source <= unsourced || f.Source > FromDocs {
		return fmt.Errorf("the fact %q comes from nowhere", f.Phrase)
	}

	return f.Scope.validate(f.Phrase)
}

// validate reports a scope that names less than its kind needs.
func (s Scope) validate(phrase string) error {
	switch s.Kind {
	case General:
		return nil
	case Language:
		if s.Lang == "" {
			return fmt.Errorf("the fact %q is about a language and names none", phrase)
		}
	case Repo, Dir, File, Symbol:
		if s.Repo == "" {
			return fmt.Errorf("the fact %q is about code and names no repository", phrase)
		}

		if s.Kind != Repo && s.Path == "" {
			return fmt.Errorf("the fact %q is about a path and names none", phrase)
		}

		if s.Kind == Symbol && s.Symbol == "" {
			return fmt.Errorf("the fact %q is about a symbol and names none", phrase)
		}
	default:
		return fmt.Errorf("the fact %q has a scope of no kind", phrase)
	}

	return nil
}

// InScope is every fact that is on, widest first, with nothing to narrow
// them by.
//
// It is what a phase is told: when the prompt is written nothing has been
// touched yet, so there is no file to ask about and the answer is everything
// known about the code the phase is going to work in. For is the same
// ordering once there is a file to narrow it by.
func InScope(all []Fact) []Fact {
	return ordered(all, func(Scope) bool { return true })
}

// Every is all of them in the same order, the ones that were turned off
// included.
//
// It is what the screen that lists them shows. InScope is what a phase is
// told, and a fact turned off is not told — but a screen that dropped them
// too would be a screen with no way to turn one back on, and a file on disk
// that nothing in the window admits exists.
func Every(all []Fact) []Fact {
	kept := slices.Clone(all)

	slices.SortStableFunc(kept, func(a, b Fact) int {
		return a.Scope.Depth() - b.Scope.Depth()
	})

	return kept
}

// For is every fact that reaches a target, widest first.
//
// The order is the point. The agent reads them in this order, so what was
// written about the file itself is the last thing it reads and has the last
// word over what was written about every repository. Facts that were turned
// off are not told at all.
//
// Two facts of the same depth keep the order they came in, which is the
// order they were written down: between "in Go, never discard an error" and
// "in Go, wrap errors with %w", neither outranks the other and the older one
// is read first.
func For(t Target, all []Fact) []Fact {
	return ordered(all, func(s Scope) bool { return s.Covers(t) })
}

// ordered keeps the facts that are on and that the test lets through, widest
// first. The sort is stable, so two facts of the same depth stay in the order
// they were written down.
func ordered(all []Fact, keep func(Scope) bool) []Fact {
	kept := make([]Fact, 0, len(all))

	for _, f := range all {
		if f.Tells() && keep(f.Scope) {
			kept = append(kept, f)
		}
	}

	slices.SortStableFunc(kept, func(a, b Fact) int {
		return a.Scope.Depth() - b.Scope.Depth()
	})

	return kept
}
