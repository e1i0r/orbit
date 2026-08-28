package task

import (
	"fmt"
	"strings"
	"time"

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

// stopWait is how long Reopen gives a run it asked to stop before it gives up
// on restarting the task. Long enough for an engine to notice a cancelled
// context and unwind; short enough that a person is still watching.
const stopWait = 30 * time.Second

// Reopen applies a directive and starts a new run of the task once the old
// one is actually gone.
//
// Waiting is the whole of it. Cancel returns as soon as the signal is sent —
// it is documented to, and it has no way not to — while the run it asked to
// stop takes as long to die as the engine it is waiting on. Starting the
// next run before then put two of them on one worktree, one branch and one
// log, and the dying one wrote its task.cancelled *after* the new one's
// task.started: a record that says the task was cancelled while it is
// running, and a marker the dying run took with it on its way out.
//
// A run that will not stop inside the window is reported rather than
// restarted over the top of, and the message names the verb that ends it.
func Reopen(s *store.Store, t Task, by, message, flowName string, unread int) (int, error) {
	if err := Direct(s, t, by, message); err != nil {
		return 0, err
	}

	if err := awaitStopped(s, t, stopWait); err != nil {
		return 0, err
	}

	chosen := flowName
	if chosen == "" {
		chosen = t.Flow
	}

	return Start(s, t, chosen, unread)
}

// awaitStopped waits until nothing live holds the task.
//
// It polls, as the gate does, and for the same reason: there is no way to
// wait on a process that is not this one's child. The interval is short
// because the answer usually arrives on the first or second look — what is
// being waited on is a process unwinding, not a person deciding.
func awaitStopped(s *store.Store, t Task, within time.Duration) error {
	deadline := time.Now().Add(within)

	for {
		pid, alive, err := Alive(s, t)
		if err != nil {
			return err
		}

		if !alive {
			return nil
		}

		if !time.Now().Before(deadline) {
			return fmt.Errorf("task %s was asked to stop, but process %d is still running after %s; end it with `orbit cancel -now %s` and start it again", t.ID, pid, within, t.ID)
		}

		time.Sleep(50 * time.Millisecond)
	}
}
