package task

// Taking a task back: the run stops and the task returns to the queue.

import (
	"context"
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// Requeue stops whatever is running on a task and puts it back in the queue.
//
// It is the gesture for a mistake of one's own — the brief was wrong, the
// engine was wrong, it was the wrong task — and that is why it is not
// Cancel. A cancelled run is a piece of work that is over, and the board
// files it under Done; this one is going to be started again as soon as
// somebody fixes what was wrong with it, and a row in Done is a row nobody
// opens again. Nor is it Direct, whose message is written as a note and
// handed to the next phase: a correction to the model is not the same thing
// as a correction to what the reader asked for.
//
// A task nothing is holding is requeued just the same, with no signal sent.
// Returning a task to the queue from needs-you or from done is the same act
// as returning it from a run, and a caller should not have to know which
// case it is in to say so.
//
// by is who took it back, in the words the reader sees, and why is their
// reason if they gave one.
func Requeue(ctx context.Context, s *store.Store, t Task, by, why string) error {
	pid, alive, err := Alive(s, t)
	if err != nil {
		return fmt.Errorf("check liveness for task %s: %w", t.ID, err)
	}

	if alive {
		if err := Cancel(s, t); err != nil {
			return err
		}

		// Waiting is not politeness, it is ordering. Cancel returns as soon
		// as the signal is sent and the run it asked to stop takes as long
		// to die as the engine it is waiting on; the run writes its own
		// task.cancelled on the way out. Written first, task.requeued would
		// be followed by that cancellation, and the fold takes the last
		// event as the state — the task the reader sent back to the queue
		// would land in Done anyway, which is the whole thing this avoids.
		if err := awaitStopped(ctx, s, t, stopWait, stopPoll); err != nil {
			return err
		}

		logger.Info("task/requeue", "%s: stopped process %d before requeuing", t.ID, pid)
	}

	e := record.Event{Kind: record.TaskRequeued, Text: strings.TrimSpace(why)}
	if by = strings.TrimSpace(by); by != "" {
		e.Data = map[string]string{"by": by}
	}

	return emit(s, t, e)
}
