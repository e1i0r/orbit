package view

// The delta spec as a reader sees it: what the change asks, what it
// promises, what it assumed, and what it decided against.
//
// It is the engine's claim about its own work and nothing verified it. The
// fold keeps it apart from everything else on an Entry for that reason: a
// pane drawing it has to say whose word it is, and a field buried among the
// facts of a phase would be read as one.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/record"
)

// Delta is those four, each of them the sentences the engine wrote under it.
type Delta struct {
	Needs      []string
	Guarantees []string
	Assumes    []string
	Instead    []string
}

// Any is whether the engine said anything at all.
func (d Delta) Any() bool {
	return len(d.Needs)+len(d.Guarantees)+len(d.Assumes)+len(d.Instead) > 0
}

// deltaOf reads a task.delta, and nothing for any other kind.
//
// Unlike the story, a delta with three of its four fields empty is a whole
// answer: a change that adds no precondition has nothing to say under needs,
// and a line invented to fill the shape would be worse than the gap.
func deltaOf(e record.Event) *Delta {
	if e.Kind != record.TaskDelta {
		return nil
	}

	d := Delta{
		Needs:      sentences(e.Data["needs"]),
		Guarantees: sentences(e.Data["guarantees"]),
		Assumes:    sentences(e.Data["assumes"]),
		Instead:    sentences(e.Data["instead"]),
	}

	if !d.Any() {
		return nil
	}

	return &d
}

// sentences splits what one field holds back into the lines it was written
// as, dropping the empties a record with a trailing newline would leave.
func sentences(field string) []string {
	var out []string

	for _, line := range strings.Split(field, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}

	return out
}
