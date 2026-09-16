package knowledge

// One rule: a sentence about some code, where it came from, and whether it
// merely says something or refuses to let the work through.

import (
	"fmt"
	"slices"
	"time"
)

// Action is what a rule does when the work reaches its scope.
type Action int

const (
	// Warns puts the sentence in front of the agent and lets it work.
	Warns Action = iota
	// Stops refuses the work at the gate.
	Stops
)

// A Rule is everything Orbit knows about one piece of code.
//
// Phrase is the only field written for a reader — it is what the agent is
// told and what a person sees in the Knowledge screen. The rest is what
// Orbit uses to decide when to say it and what to do about it.
type Rule struct {
	// ID is what this rule is called, for as long as it exists. It is
	// coined once, when Orbit first writes the rule down, and nothing
	// changes it after that — not correcting the sentence, not moving the
	// scope, not turning it off.
	//
	// It is here because everything else about a rule can change. The file
	// is named after the sentence, so rewording one renames it, and what
	// the record wrote down about the old name stops being findable. A
	// cycle that cannot say "this is the same thing you said differently
	// three weeks ago" is not a cycle: it is a pile of separate rules.
	//
	// Empty is allowed and is not an error. A rule somebody wrote by hand
	// has no id until Orbit writes it, and writing files by hand is half
	// the reason these are files.
	ID     string
	Scope  Scope
	Source Source
	// Phrase is the rule, in a sentence. It is what gets read.
	Phrase string
	// Stops is what the rule was asked to do. Whether it can is Action's
	// answer, not this field's: see there.
	Stops bool
	// Check is the command that says yes or no without opinion. A rule that
	// wants to stop is worth nothing without one.
	Check string
	// Ref names what the rule came out of — a task, a decision, an
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
	// from is the file this rule was read out of, and empty for one that
	// has never been on disk.
	//
	// Unexported because nobody outside sets it and nobody should: it is
	// what the store saw, not something a screen decides. What it is for is
	// replacing — a file somebody wrote by hand is called whatever they
	// called it, and a replacement that worked out the old name from the
	// rule's own fields left that file behind, still told and still
	// refusing work.
	from string
}

// Tells is whether this rule reaches a phase at all: it is what is written
// into every prompt and what the gate refuses work over. Only an active rule
// does — a paused one is the reader saying not now, and one switched off is
// them saying no.
func (f Rule) Tells() bool { return f.State == Active }

// Action is what this rule actually does, which is not always what it was
// asked to do.
//
// Warning is a sentence and needs nothing else. Stopping is the gate refusing
// the work, and refusing needs something that answers yes or no without an
// opinion in it — a command, a pattern over the diff, a test that runs. A
// rule that asked to stop and brought no check would never fire while reading
// as though it would, so it warns, and the screen says so.
func (f Rule) Action() Action {
	if f.Stops && f.Check != "" {
		return Stops
	}

	return Warns
}

// Standing is what a rule is doing, as one answer rather than three fields.
//
// Five, and they are exclusive. Whether a rule is waiting to be decided
// about, whether somebody stopped it applying, and what it would do if it
// were applying are three questions in the record — and every surface that
// shows a rule has to fold them into the one thing a reader wants to know.
// Folded here, the cockpit and the browser cannot disagree about a rule they
// are both looking at.
type Standing int

// The five, in the order a list draws them: what wants an answer first, then
// what is working, then what is not.
const (
	// Waiting is a rule sent back to be decided about — it stopped somebody,
	// or somebody paused it. It is first whatever else is true of it: a
	// question filed under what it happens to do meanwhile is a question
	// nobody answers.
	Waiting Standing = iota
	// Blocks runs its command at the gate and sends the work back when that
	// command fails. Says only reaches the prompt.
	Blocks
	Says
	// Stopped is paused, and Silent is switched off. They are apart because
	// they are different answers: one is not now, the other is no.
	Stopped
	Silent
)

// Standing folds a rule down to the one of the five it is in.
func (f Rule) Standing() Standing {
	switch {
	case f.Review:
		return Waiting
	case f.State == Paused:
		return Stopped
	case !f.Tells():
		return Silent
	case f.Action() == Stops:
		return Blocks
	}

	return Says
}

// Validate reports the first thing that would make a rule untrustworthy.
//
// Every rule has a source and a scope; without something behind it, it does
// not get in. That is not tidiness — a sentence in the agent's context that
// nobody can trace is indistinguishable from one the model made up, and the
// whole point of keeping this outside the model is that it can be traced.
func (f Rule) Validate() error {
	if f.Phrase == "" {
		return fmt.Errorf("a rule with no sentence says nothing")
	}

	if f.Source <= unsourced || f.Source > FromGates {
		return fmt.Errorf("the rule %q comes from nowhere", f.Phrase)
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
			return fmt.Errorf("the rule %q is about a language and names none", phrase)
		}
	case Repo, Dir, File, Symbol:
		if s.Repo == "" {
			return fmt.Errorf("the rule %q is about code and names no repository", phrase)
		}

		if s.Kind != Repo && s.Path == "" {
			return fmt.Errorf("the rule %q is about a path and names none", phrase)
		}

		if s.Kind == Symbol && s.Symbol == "" {
			return fmt.Errorf("the rule %q is about a symbol and names none", phrase)
		}
	default:
		return fmt.Errorf("the rule %q has a scope of no kind", phrase)
	}

	return nil
}

// InScope is every rule that is on, widest first, with nothing to narrow
// them by.
//
// It is what a phase is told: when the prompt is written nothing has been
// touched yet, so there is no file to ask about and the answer is everything
// known about the code the phase is going to work in. For is the same
// ordering once there is a file to narrow it by.
func InScope(all []Rule) []Rule {
	return ordered(all, func(Scope) bool { return true })
}

// Every is all of them in the same order, the ones that were turned off
// included.
//
// It is what the screen that lists them shows. InScope is what a phase is
// told, and a rule turned off is not told — but a screen that dropped them
// too would be a screen with no way to turn one back on, and a file on disk
// that nothing in the window admits exists.
func Every(all []Rule) []Rule {
	kept := slices.Clone(all)

	slices.SortStableFunc(kept, func(a, b Rule) int {
		return a.Scope.Depth() - b.Scope.Depth()
	})

	return kept
}

// For is every rule that reaches a target, widest first.
//
// The order is the point. The agent reads them in this order, so what was
// written about the file itself is the last thing it reads and has the last
// word over what was written about every repository. Rules that were turned
// off are not told at all.
//
// Two rules of the same depth keep the order they came in, which is the
// order they were written down: between "in Go, never discard an error" and
// "in Go, wrap errors with %w", neither outranks the other and the older one
// is read first.
func For(t Target, all []Rule) []Rule {
	return ordered(all, func(s Scope) bool { return s.Covers(t) })
}

// ordered keeps the rules that are on and that the test lets through, widest
// first. The sort is stable, so two rules of the same depth stay in the order
// they were written down.
func ordered(all []Rule, keep func(Scope) bool) []Rule {
	kept := make([]Rule, 0, len(all))

	for _, f := range all {
		if f.Tells() && keep(f.Scope) {
			kept = append(kept, f)
		}
	}

	slices.SortStableFunc(kept, func(a, b Rule) int {
		return a.Scope.Depth() - b.Scope.Depth()
	})

	return kept
}
