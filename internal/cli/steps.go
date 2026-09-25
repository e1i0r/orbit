package cli

// What the supervisor does for a delivery verb, written onto its task.

import (
	"regexp"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/ui"
	"github.com/e1i0r/orbit/internal/view"
)

// stepsOnto writes each tool call the supervisor makes onto the task its
// errand is about, and nothing for a line somebody typed. A step that cannot
// be written is logged and the work goes on: the record is how the reader
// watches it, not what it depends on.
func stepsOnto(s *store.Store, about ui.Errand) func(engine.StreamEvent) {
	if about.Verb == "" {
		return nil
	}

	return func(ev engine.StreamEvent) {
		if ev.Type != "tool_call" {
			return
		}

		err := task.DeliveryStep(s, subject(about.Task), about.Verb, ev.ToolCall.Name, ev.ToolCall.Args)
		if err != nil {
			logger.Error("cli/deliver", "%s: a step of %s could not be written: %v", about.Task.ID, about.Verb, err)
		}
	}
}

// autopilotVerb is what the record calls an autopilot pass on a task, the
// same word the window put out on it.
const autopilotVerb = "AUTOPILOT"

// autopilotSteps writes each tool call an autopilot pass makes onto the
// tasks it is about: the ones its arguments name, or every task of the pass
// when they name none of them — reading the board is a step towards each.
func autopilotSteps(s *store.Store, tasks []view.Task) func(engine.StreamEvent) {
	return func(ev engine.StreamEvent) {
		if ev.Type != "tool_call" {
			return
		}

		for _, t := range namedIn(ev.ToolCall.Args, tasks) {
			err := task.DeliveryStep(s, subject(t), autopilotVerb, ev.ToolCall.Name, ev.ToolCall.Args)
			if err != nil {
				logger.Error("cli/autopilot", "%s: a step of the autopilot pass could not be written: %v", t.ID, err)
			}
		}
	}
}

// namedIn is the tasks whose id the arguments name, or all of them when
// they name none.
func namedIn(args string, tasks []view.Task) []view.Task {
	var named []view.Task

	for _, t := range tasks {
		// Whole ids only: ACME-1 is not named by an argument about ACME-12.
		if regexp.MustCompile(`\b` + regexp.QuoteMeta(t.ID) + `\b`).MatchString(args) {
			named = append(named, t)
		}
	}

	if len(named) == 0 {
		return tasks
	}

	return named
}
