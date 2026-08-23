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
// Two of the five are not read by anything yet. They are documented as inert
// rather than deleted: the design defines the five-field object, so removing
// them would put the code out of step with the authority — but a field that
// looks live and is not is a lie, and one of these two is the security
// posture. Both gaps are written down in NEXT.md.
type Phase struct {
	Name   string `json:"name"`
	Engine string `json:"engine"`
	Model  string `json:"model,omitempty"`

	// Wait is inert. task.Run walks every phase straight through and never
	// consults it, so no phase can stop for a human yet. What is missing is
	// a Run that can pause and a window to release it from — plan 2.
	// WithAutopilot already clears this field, which is the whole of what
	// the autopilot switch will mean once something reads it.
	Wait bool `json:"wait,omitempty"`

	// Permissions is inert, and it is the more serious of the two.
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

// WithAutopilot returns a copy that stops for nobody.
//
// This is the whole of what the autopilot switch does. Night is not a second
// system: it is this function applied to an ordinary flow.
//
// Copying the phase slice is not enough on its own. A copied Phase still
// points at the same Permissions backing array as the phase it came from, so
// writing through the copy would reach into the original — a flow the caller
// still holds and believes is untouched.
func (f Flow) WithAutopilot() Flow {
	phases := make([]Phase, len(f.Phases))
	copy(phases, f.Phases)
	for i := range phases {
		phases[i].Wait = false
		if phases[i].Permissions != nil {
			perms := make([]string, len(phases[i].Permissions))
			copy(perms, phases[i].Permissions)
			phases[i].Permissions = perms
		}
	}
	return Flow{Name: f.Name, Phases: phases}
}
