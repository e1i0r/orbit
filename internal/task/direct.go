package task

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/store"
)

// Direct halts any in-flight execution on the task, leaves both a dialogue
// entry and a corrective note in the record, and prepares it to be resumed.
//
// If the task is running, it cancels the active run via SIGTERM so that the
// engine's current session and worktree are preserved on exit.
// If the task is not running (e.g., in ToDo, NeedsYou, or Done), the dialogue
// and note are recorded, ready for the next run.
//
// by identifies who directed the task ("supervisor", "operator", "mcp").
// message is the directive or corrective feedback.
func Direct(s *store.Store, t Task, by, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return fmt.Errorf("task %s: directive message cannot be empty", t.ID)
	}
	if by == "" {
		by = "supervisor"
	}
	if err := Dialogue(s, t, by, message); err != nil {
		return fmt.Errorf("task %s dialogue: %w", t.ID, err)
	}
	noteText := fmt.Sprintf("[%s] %s", by, message)
	if err := Note(s, t, noteText); err != nil {
		return fmt.Errorf("task %s note: %w", t.ID, err)
	}
	pid, ok, err := Alive(s, t)
	if err != nil {
		return fmt.Errorf("check liveness for task %s: %w", t.ID, err)
	}
	if ok && pid > 0 {
		if err := Cancel(s, t); err != nil {
			return fmt.Errorf("stop task %s: %w", t.ID, err)
		}
	}
	return nil
}

// Reopen applies a directive and immediately starts a new run of the task.
func Reopen(s *store.Store, t Task, by, message, flowName string, unread int) (int, error) {
	if err := Direct(s, t, by, message); err != nil {
		return 0, err
	}
	chosen := flowName
	if chosen == "" {
		chosen = t.Flow
	}
	return Start(s, t, chosen, unread)
}
