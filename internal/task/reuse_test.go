package task

// Writing down a task under an id that was deleted.
//
// The id comes from a tracker and the tracker hands the same one back. What
// went wrong before was worse than a refusal: the board hid the task, the
// disk still had its directory, and `orbit new` answered "already exists"
// about something nothing on screen could show.

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestADeletedIdCanBeWrittenDownAgain, and what comes back is a new task
// rather than the old one wearing its history.
func TestADeletedIdCanBeWrittenDownAgain(t *testing.T) {
	s, r := fixture(t)

	first, err := Create(s, r, "ACME-1", "the first life", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := emit(s, first, record.Event{Kind: record.TaskStarted}); err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := Delete(s, first); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// The directory goes with it: the text is in the record verbatim, and
	// what is left of it is what made `orbit new` refuse the id.
	dir, err := s.TaskDir("ACME-1")
	if err != nil {
		t.Fatalf("task dir: %v", err)
	}

	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the deleted task's directory is still there: %v", err)
	}

	again, err := Create(s, r, "ACME-1", "the second life", "")
	if err != nil {
		t.Fatalf("the id was not free: %v", err)
	}

	if again.Text != "the second life" {
		t.Errorf("the new task reads %q", again.Text)
	}

	// The record keeps both lives, because it keeps everything.
	events, err := Events(s, again)
	if err != nil {
		t.Fatalf("events: %v", err)
	}

	var created int

	for _, e := range events {
		if e.Kind == record.TaskCreated {
			created++
		}
	}

	if created != 2 {
		t.Errorf("the record holds %d creations, want both lives", created)
	}

	if !strings.Contains(events[0].Text, "the first life") {
		t.Errorf("the first life is gone from the record: %+v", events[0])
	}
}
