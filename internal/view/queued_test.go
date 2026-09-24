package view

import (
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// TestATaskWaitingInTheQueueIsNotToDo. It was asked to start, so it is not
// waiting on anybody; and autopilot, which takes from To Do, would ask again.
func TestATaskWaitingInTheQueueIsNotToDo(t *testing.T) {
	at := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)

	got := Fold([]record.Event{
		{At: at, Kind: record.TaskCreated, Text: "work"},
		{At: at.Add(time.Minute), Kind: record.TaskQueued},
	})

	if BandOf(got) != Running || got.Reason.Key != ReasonQueued {
		t.Errorf("a queued task folds to band %v reason %q, want Running and queued",
			BandOf(got), got.Reason.Key)
	}

	if !got.Since.Equal(at.Add(time.Minute)) {
		t.Errorf("Since is %v, want when it was queued: its place in line", got.Since)
	}
}

// TestAQueuedTaskForgetsWhatTheLastRunWasDoing. A run that failed halfway
// and was started again waits in the queue; the overview still counted the
// tool calls of the run that had ended as this attempt's.
func TestAQueuedTaskForgetsWhatTheLastRunWasDoing(t *testing.T) {
	at := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)

	got := Fold([]record.Event{
		{At: at, Kind: record.TaskCreated, Text: "work"},
		{At: at.Add(time.Minute), Kind: record.TaskStarted},
		{At: at.Add(2 * time.Minute), Kind: record.PhaseStarted, Phase: "implement"},
		{
			At: at.Add(3 * time.Minute), Kind: record.PhaseToolCall,
			Phase: "implement", Text: "Bash: make check",
		},
		{At: at.Add(4 * time.Minute), Kind: record.TaskFailed, Text: "engine gone"},
		{At: at.Add(5 * time.Minute), Kind: record.TaskQueued},
	})

	if got.CurrentAction != "" || got.ToolCallCount != 0 {
		t.Errorf("a queued task still says %q after %d tool calls, from the run that ended",
			got.CurrentAction, got.ToolCallCount)
	}
}
