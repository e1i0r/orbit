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
