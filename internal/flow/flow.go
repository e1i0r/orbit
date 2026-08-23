// Package flow holds the phases of a task as data rather than as code.
//
// The program this replaces soldered its pipeline into 1,080 lines of shell:
// setup, implement, gates, pull request, three review rounds, settle, in that
// order, for ever. Here a flow is a list, adding one is writing a file, and
// the interpreter that walks it is a few dozen lines.
package flow

import "fmt"

// Phase is one step, and the five things that decide how it runs.
//
// One of the five is not read by anything yet. It is documented as inert
// rather than deleted: the design defines the five-field object, so removing
// it would put the code out of step with the authority — but a field that
// looks live and is not is a lie, and this one is the security posture. It is
// named here rather than anywhere else, because this struct is where a reader
// meets it.
type Phase struct {
	Name   string `json:"name"`
	Engine string `json:"engine"`
	Model  string `json:"model,omitempty"`

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

	// Permissions is inert.
	// engine.Request has no field that could carry it, so no engine could
	// honour it even if Run passed it along: the built-in task flow ships
	// "permissions": ["repo"] while claudeArgs passes no permission flag at
	// all. The engine's own default is therefore the security posture, and
	// it is stated nowhere. Mapping these names onto a real engine's flags
	// decides what an agent is allowed to touch, so it belongs to the plan
	// that builds it, decided deliberately and reviewed as such.
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
	}
	return nil
}
