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
	// Off is a fact somebody disagreed with. It stays, and stops being
	// told: disagreeing with a fact and losing the record that it existed
	// are different things.
	Off bool
}

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

	if f.Source <= unsourced || f.Source > FromProduction {
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
		if !f.Off && keep(f.Scope) {
			kept = append(kept, f)
		}
	}

	slices.SortStableFunc(kept, func(a, b Fact) int {
		return a.Scope.Depth() - b.Scope.Depth()
	})

	return kept
}
