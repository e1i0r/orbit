package task

// What was done to a task from outside a run: a tool call over MCP, an
// interactive session opened on it from the cockpit.

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// Dialogue records that something outside a run acted on the task.
//
// It is not a note, and that difference is the whole reason it exists. A
// note is handed to the next phase that starts — unconsumedNotes reads
// exactly record.TaskNoted — so writing "a model paused this over mcp" as
// a note would put it in the engine's prompt, where it reads as an
// instruction to pause. This kind is written for the reader and for nobody
// else: the cockpit's notes tab draws it beside the notes, and no phase is
// ever told it.
//
// by is what acted, in the words the reader sees: "mcp" for a tool call,
// the engine's own name for a session somebody opened by hand.
func Dialogue(s *store.Store, t Task, by, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("task %s: dialogue text cannot be empty", t.ID)
	}
	e := record.Event{Kind: record.TaskDialogue, Text: text}
	if by = strings.TrimSpace(by); by != "" {
		e.Data = map[string]string{"by": by}
	}
	return emit(s, t, e)
}
