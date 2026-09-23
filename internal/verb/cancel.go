package verb

// Cancelling a task: its run, or its place in the queue.

import (
	"github.com/e1i0r/orbit/internal/queue"
	"github.com/e1i0r/orbit/internal/task"
)

// cancelled asks the run to stop where it stands, and to write down that it
// was stopped — the signal rather than a word, because a cancel should not
// wait for a phase to end.
func cancelled(w World, in In) (Out, error) {
	t, err := found(w, in)
	if err != nil {
		return Out{}, err
	}

	left, err := queue.Leave(w.Store(), t)
	if err != nil {
		return Out{}, err
	}

	if left {
		return Out{Said: t.ID + " taken out of the queue before it started"}, nil
	}

	if err := task.Cancel(w.Store(), t); err != nil {
		return Out{}, err
	}

	return Out{Said: "asked the run of " + t.ID + " to stop"}, nil
}
