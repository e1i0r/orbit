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
func unconsumedNotes(s *store.Store, t Task) []string {
	path, err := s.EventsPath(t.Repo.Path, t.ID)
	if err != nil {
		return nil
	}

	events, err := record.Read(path)
	if err != nil {
		return nil
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

	return notes
}
