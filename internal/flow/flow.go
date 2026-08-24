// Package flow holds the phases of a task as data rather than as code.
//
// The program this replaces soldered its pipeline into 1,080 lines of shell:
// setup, implement, gates, pull request, three review rounds, settle, in that
// order, for ever. Here a flow is a list, adding one is writing a file, and
// the interpreter that walks it is a few dozen lines.
package flow

import "fmt"

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
// is silent. A flow file that said "repository" where it meant "repo" used
// to load, grant nothing, and leave the engine's own default posture in
// charge — a wider grant than the file asked for, arriving through a typo,
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

// Phase is one step, and the five things that decide how it runs.
type Phase struct {
	Name     string `json:"name"`
	Engine   string `json:"engine"`
	Model    string `json:"model,omitempty"`
	Effort   string `json:"effort,omitempty"`
	Thinking string `json:"thinking,omitempty"`

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
}

// Flow is an ordered list of phases under a name.
type Flow struct {
	Name   string  `json:"name"`
	Phases []Phase `json:"phases"`
}

// Validate reports the first thing that would make a flow unrunnable.
func (f Flow) Validate() error {
	if f.Name == "" {
		return fmt.Errorf("the flow has no name")
	}
	if len(f.Phases) == 0 {
		return fmt.Errorf("flow %q has no phases", f.Name)
	}
	seen := make(map[string]bool, len(f.Phases))
	for i, p := range f.Phases {
		if p.Name == "" {
			return fmt.Errorf("flow %q: phase %d has no name", f.Name, i+1)
		}
		if p.Engine == "" {
			return fmt.Errorf("flow %q: phase %q names no engine", f.Name, p.Name)
		}
		if seen[p.Name] {
			return fmt.Errorf("flow %q: two phases are called %q", f.Name, p.Name)
		}
		seen[p.Name] = true
		// A permission nobody defined is refused here, at load, rather than
		// where it would otherwise surface: after a worktree, a process and
		// a bill. Load calls Validate as it decodes, so a flow file with a
		// typo in it never becomes a run at all.
		for _, perm := range p.Permissions {
			switch perm {
			case PermissionRead, PermissionRepo, PermissionNetwork:
			default:
				return fmt.Errorf("flow %q: phase %q asks for the permission %q, which is not one of %s, %s and %s", f.Name, p.Name, perm, PermissionRead, PermissionRepo, PermissionNetwork)
			}
		}
	}
	return nil
}
