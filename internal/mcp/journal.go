package mcp

// What a tool call leaves behind on the task it acted on.

import (
	"fmt"

	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
)

// journalBy is what the record says acted. It is the server and not the
// model behind it: this process is told which client connected, never which
// model is driving it, and inventing a name for one would be evidence about
// a party nobody identified.
const journalBy = "mcp"

// journal writes down that a model changed something through this server.
//
// Only the calls that change something are written. A tool call that read
// the board is not a thing that happened to a task, and a record that grew
// a line every time a supervisor looked at it would bury the four lines
// that say what it did — in a file the next run is also folded from.
//
// It is a dialogue event and not a note for the reason task.Dialogue gives:
// a note is handed to the next phase, and "asked this task to pause" is not
// an instruction anybody meant to give an engine.
//
// The answer is the clause the tool adds to its own sentence, empty when
// the trace was written. The act has already happened by the time this
// runs, so a failure here cannot turn the call into a refusal — the task
// really is cancelled — but it must not be silent either, because the model
// is about to tell somebody the cockpit will show this.
func journal(s *store.Store, t task.Task, format string, args ...any) string {
	if err := task.Dialogue(s, t, journalBy, fmt.Sprintf(format, args...)); err != nil {
		return fmt.Sprintf(" (this call is not in the task's history: %v)", err)
	}
	return ""
}
