package cli

// What the supervisor does for a delivery verb, written onto its task.

import (
	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/ui"
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
