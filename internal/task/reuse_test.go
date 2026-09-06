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

// TestADirectoryLeftBehindByAnOlderDeleteIsNotATask. Deleting used to take
// the row off the board and leave the directory, so writing the id down
// again answered "already exists" about something nothing on screen could
// show. Every state root that ever ran that version still has those
// directories, and the record is what says which of them are stale.
func TestADirectoryLeftBehindByAnOlderDeleteIsNotATask(t *testing.T) {
	s, r := fixture(t)

	first, err := Create(s, r, "ACME-2", "the first life", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Deleted the way the older version did it: the event, and the
	// directory left where it was.
	if err := emit(s, first, record.Event{Kind: record.TaskDeleted}); err != nil {
		t.Fatalf("delete: %v", err)
	}

	dir, err := s.TaskDir("ACME-2")
	if err != nil {
		t.Fatalf("task dir: %v", err)
	}

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("the fixture did not leave a directory behind: %v", err)
	}

	again, err := Create(s, r, "ACME-2", "the second life", "")
	if err != nil {
		t.Fatalf("the leftover directory refused the id: %v", err)
	}

	if again.Text != "the second life" {
		t.Errorf("the new task reads %q", again.Text)
	}
}

// TestATaskThatIsStillThereStillRefusesItsId, which is the whole point of
// the check: two tasks with one name would share a directory and interleave
// their events.
func TestATaskThatIsStillThereStillRefusesItsId(t *testing.T) {
	s, r := fixture(t)

	if _, err := Create(s, r, "ACME-3", "the only life", ""); err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := Create(s, r, "ACME-3", "another", ""); !errors.Is(err, ErrExists) {
		t.Errorf("writing over a live task answered %v", err)
	}
}
