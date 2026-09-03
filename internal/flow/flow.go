// Package flow holds the phases of a task as data rather than as code.
//
// The program this replaces soldered its pipeline into 1,080 lines of shell:
// setup, implement, gates, pull request, three review rounds, settle, in that
// order, for ever. Here a flow is a list, adding one is writing a file, and
// the interpreter that walks it is a few dozen lines.
package flow

import (
	"fmt"
	"slices"
)

// The closed vocabulary a phase may use to say what it is allowed to touch.
//
// A flow file is where a permission is declared, and Validate is what reads
// that file, so the names live here. internal/engine spells the same three
// out again, because that is where they become a real command line and the
// two packages may not import each other — both have an empty row in the
// layer table, which is what keeps the window from being able to start a
// model. Three duplicated strings are the price of that, paid knowingly.
//
// The set is closed rather than open because the failure mode of an open one
// is silent. A flow file that says "repository" where it meant "repo" would
// otherwise load, grant nothing, and leave the engine's own default posture
// in charge — a wider grant than the file asked for, arriving through a typo,
// with nothing anywhere saying so.
const (
	// PermissionRead may read the worktree and may run no command that
	// writes.
	PermissionRead = "read"
	// PermissionRepo may read and write inside the worktree and run the
	// repository's own commands.
	PermissionRepo = "repo"
	// PermissionNetwork may reach the network.
	PermissionNetwork = "network"
)

// Permissions is the whole vocabulary, in the order a reader meets it.
//
// It exists for the caller that has to offer the set rather than check one
// of it: internal/mcp declares these as the enum of a tool argument. Three
// string literals of its own would be a different duplication from the one
// above — internal/mcp may import this package, and does — so they would buy
// nothing and go stale on the day a fourth permission is added, telling a
// supervising model the set is three long when it is four.
func Permissions() []string {
	return []string{PermissionRead, PermissionRepo, PermissionNetwork}
}

// Phase is one step, and the five things that decide how it runs.
type Phase struct {
	Name       string `json:"name"`
	Engine     string `json:"engine"`
	Model      string `json:"model,omitempty"`
	Effort     string `json:"effort,omitempty"`
	Thinking   string `json:"thinking,omitempty"`
	Prompt     string `json:"prompt,omitempty"`
	FeedOutput bool   `json:"feed_output,omitempty"`

	// Wait is this phase's default answer to "should this stop for a human?".
	// It is a default and not a switch, which is the distinction that took a
	// deleted function to learn. task.Run puts every phase to its gate, and
	// the gate reads the autopilot setting and the reader's control word at
	// that moment: a phase with Wait true stops unless autopilot is on, and a
	// phase with Wait false runs unless the reader has pressed pause. A Flow
	// is copied by value into a run, so anything decided about waiting before
	// the run started could never hear about a switch flipped while it was
	// going.
	Wait bool `json:"wait,omitempty"`

	// Permissions is what this phase is allowed to touch, in the closed
	// vocabulary above. It was inert for two plans — engine.Request had no
	// field that could carry it, so the built-in flows shipped
	// "permissions": ["repo"] while the adapter passed no permission flag
	// at all and the engine's own default was the real posture, stated
	// nowhere. It is now carried to the engine, mapped to that engine's
	// flags, and written into phase.started so a run's posture is
	// recoverable from the log.
	//
	// An empty list is not "no opinion". It is the phase that asks for
	// nothing, and the engine turns it into the most restrictive posture it
	// can state rather than into an absence of flags.
	Permissions []string `json:"permissions,omitempty"`
	Gates       []Gate   `json:"gates,omitempty"`

	// Loop makes this phase a block rather than a step: the phases inside
	// it go round until every check passes, and never more than Max times.
	//
	// A phase is one or the other. An engine and a loop on the same phase
	// would leave the interpreter to decide which of the two the flow
	// meant, and whichever it picked would surprise the person who wrote
	// the file.
	Loop *Loop `json:"loop,omitempty"`
}

// Gate is a named verification check run after a phase engine finishes.
type Gate struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// DefaultAttempts is how many times a phase whose gate refused it is run,
// counting the first run. Three is the number a flow that says nothing gets.
//
// It is a default rather than a fixed rule because how many shots a phase
// deserves is a property of the flow: `quick` wants one and to be told, and
// a long review cycle can afford more. What it is not is unbounded — a phase
// that cannot satisfy its own gate spends money on every turn of the loop,
// and the run that never stops is the one nobody is watching.
const DefaultAttempts = 3

// Flow is an ordered list of phases under a name.
type Flow struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// Attempts is how many times one phase may be run before the run gives
	// up on it. Zero is not "no attempts": it is a flow that did not say,
	// and AttemptCap reads it as the default. Every flow file written
	// before this field existed decodes to zero, which is what makes them
	// mean what they meant.
	Attempts int `json:"attempts,omitempty"`
	// DiffBudget is how many lines a task walking this flow may change
	// before it stops and asks. Zero is no budget, the working zero every
	// setting in Orbit has.
	//
	// It is per flow and not per phase because it is about the change, not
	// about the step that made it: what a reader agrees to is the size of
	// what they will have to read, and which phase wrote which line of it
	// is not a thing they agreed anything about.
	DiffBudget int `json:"diff_budget,omitempty"`
	// AllowNewDependencies lets a task add libraries without stopping to
	// ask. The working zero is the careful one: a flow that says nothing
	// stops, because a dependency is what a project carries afterwards and
	// the agent is not who decides that.
	AllowNewDependencies bool `json:"allow_new_dependencies,omitempty"`
	// AllowContradictions lets a task change code against a decision it
	// recorded earlier without stopping. The working zero checks, and the
	// check costs nothing in a repository whose decisions say nothing
	// about the files being worked on: it is asked only when one of them
	// reaches the change.
	AllowContradictions bool `json:"allow_contradictions,omitempty"`
	// Validate_ has the supervisor read the finished run against the task
	// it was written for, and decide: it is done, it needs another run, or
	// it needs a person.
	//
	// Off by default, unlike the other two, because this one costs a model
	// call on every task that finishes rather than only when something
	// looks wrong. A flow that wants the last word asks for it.
	//
	// The trailing underscore keeps the field from colliding with Validate,
	// which is what says a flow can be walked at all — the JSON name is
	// what a flow file writes, and it is `validate`.
	Validate_ bool    `json:"validate,omitempty"`
	Phases    []Phase `json:"phases"`
}

// validate is what has to be true of one phase, wherever it sits: at the
// top of a flow, or inside a loop of one.
//
// It is a method so that a loop's phases are held to the same rules as the
// flow's own. Two lists of checks would agree until the day somebody added
// a rule to one of them.
func (p Phase) validate(flow string, siblings int) error {
	if p.Loop != nil && p.Engine != "" {
		return fmt.Errorf("flow %q: phase %q is both an engine and a loop, and only one of those can run", flow, p.Name)
	}

	if p.Loop == nil && p.Engine == "" {
		return fmt.Errorf("flow %q: phase %q names no engine", flow, p.Name)
	}

	// A permission nobody defined is refused here, at load, rather than
	// where it would otherwise surface: after a worktree, a process and a
	// bill. Load calls Validate as it decodes, so a flow file with a typo
	// in it never becomes a run at all.
	for _, perm := range p.Permissions {
		if !slices.Contains(Permissions(), perm) {
			return fmt.Errorf("flow %q: phase %q asks for unknown permission %q", flow, p.Name, perm)
		}
	}

	for gi, g := range p.Gates {
		if g.Name == "" {
			return fmt.Errorf("flow %q: phase %q has a gate at position %d with no name", flow, p.Name, gi+1)
		}

		if g.Command == "" {
			return fmt.Errorf("flow %q: phase %q gate %q has no command", flow, p.Name, g.Name)
		}
	}

	return p.Loop.validate(flow, p.Name)
}

// AttemptCap is how many times a phase of this flow may be run.
func (f Flow) AttemptCap() int {
	if f.Attempts <= 0 {
		return DefaultAttempts
	}

	return f.Attempts
}

// Validate reports the first thing that would make a flow unrunnable.
func (f Flow) Validate() error {
	if f.Name == "" {
		return fmt.Errorf("the flow has no name")
	}

	if len(f.Phases) == 0 {
		return fmt.Errorf("flow %q has no phases", f.Name)
	}

	// A negative cap is refused rather than read as the default, because the
	// two are different mistakes: zero is a file that did not mention
	// attempts, and -1 is a file that meant something by it. Reading the
	// second as three would run a phase three times for somebody who wrote
	// down that they wanted otherwise.
	if f.DiffBudget < 0 {
		return fmt.Errorf("flow %q allows %d changed lines, which is not a number of lines", f.Name, f.DiffBudget)
	}

	if f.Attempts < 0 {
		return fmt.Errorf("flow %q allows %d attempts, which is not a number of attempts", f.Name, f.Attempts)
	}

	seen := make(map[string]bool, len(f.Phases))
	for i, p := range f.Phases {
		if p.Name == "" {
			return fmt.Errorf("flow %q: phase %d has no name", f.Name, i+1)
		}

		if seen[p.Name] {
			return fmt.Errorf("flow %q: phase %q appears more than once", f.Name, p.Name)
		}

		seen[p.Name] = true

		if err := p.validate(f.Name, len(f.Phases)); err != nil {
			return err
		}
	}

	return nil
}
