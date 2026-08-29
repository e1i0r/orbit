package task

// User notes left for a task, and how phases consume them.

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// Note records a user note for the task.
//
// A note is recorded in the task's events log and is consumed by the next
// phase that starts. A note written while nothing is running is still
// recorded and will be read when a run starts.
func Note(s *store.Store, t Task, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("task %s: note text cannot be empty", t.ID)
	}

	return emit(s, t, record.Event{
		Kind: record.TaskNoted,
		Text: text,
	})
}

// unconsumedNotes returns the notes recorded since the last phase start.
//
// A log it cannot read is an error and not an empty list, and the difference
// matters more here than almost anywhere else in this package. These notes
// are the operator's own words, and the only thing they do is go into the
// prompt of the phase about to start. Swallowing the error handed that phase
// a prompt with the correction missing from it and told nobody — the model
// then does the thing it was told not to do, and the record shows a note
// that was written, a phase that started after it, and no reason at all why
// the note had no effect. That is the same call Supervise makes about a
// thread it cannot read, for the same reason: something that cannot see it
// is missing context speaks as though it has all of it.
func unconsumedNotes(s *store.Store, t Task) ([]string, error) {
	events, err := Events(s, t)
	if err != nil {
		return nil, err
	}

	var notes []string

	for _, e := range events {
		switch e.Kind {
		case record.PhaseStarted:
			notes = nil
		case record.TaskNoted:
			if strings.TrimSpace(e.Text) != "" {
				notes = append(notes, e.Text)
			}
		}
	}

	return notes, nil
}
