package migrate

// What a pass does when it meets something it cannot read. None of it stops
// the pass: a migration that gives up on the first damaged file leaves a
// state root half moved, which is the one shape nobody can reason about.

import (
	"os"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestADamagedLineIsCarriedAcrossAsOne. internal/record reads a line it
// cannot parse as a record.unreadable event and carries on; the migration
// inherits that, because the event saying a line could not be read is itself
// part of what happened.
func TestADamagedLineIsCarriedAcrossAsOne(t *testing.T) {
	s := root(t)

	log(t, s, "ACME-1", record.Event{Kind: record.TaskCreated, Text: "Retry the webhook"})

	path, err := s.EventsPath("ACME-1")
	if err != nil {
		t.Fatalf("events path: %v", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open the log: %v", err)
	}

	if _, err := f.WriteString("{this is not json}\n"); err != nil {
		t.Fatalf("write the damaged line: %v", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("close the log: %v", err)
	}

	if _, err := Run(s); err != nil {
		t.Fatalf("migrate a log with a damaged line: %v", err)
	}

	events, err := opened(t, s).Events("ACME-1")
	if err != nil {
		t.Fatalf("read the events: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("the record holds %d events, want the good one and the note about the bad one", len(events))
	}

	if events[1].Kind != record.Unreadable {
		t.Errorf("the damaged line came across as %q, want %q", events[1].Kind, record.Unreadable)
	}
}

// TestOneDamagedTaskDoesNotStopTheRest. The whole point of running this
// before every command is that it finishes: a pass that gives up on the
// first unreadable log leaves a state root half moved, which is the one
// shape nobody can reason about.
func TestOneDamagedTaskDoesNotStopTheRest(t *testing.T) {
	s := root(t)

	log(t, s, "ACME-1", record.Event{Kind: record.TaskCreated, Text: "the one that cannot be read"})
	log(t, s, "ACME-2", record.Event{Kind: record.TaskCreated, Text: "the one that can"})

	path, err := s.EventsPath("ACME-1")
	if err != nil {
		t.Fatalf("events path: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chmod(path, 0o600); err != nil {
			t.Errorf("give the log back its mode: %v", err)
		}
	})

	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("take the log's mode away: %v", err)
	}

	out, err := Run(s)
	if err == nil {
		t.Fatal("a log that could not be read was not reported")
	}

	if !strings.Contains(err.Error(), "ACME-1") {
		t.Errorf("the failure reads %q, want the task that could not be read named", err)
	}

	if out.Events != 1 {
		t.Errorf("the pass moved %d events, want the one from the task it could read", out.Events)
	}

	events, err := opened(t, s).Events("ACME-2")
	if err != nil {
		t.Fatalf("read the events: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("the readable task moved %d events, want its one", len(events))
	}
}

// TestASupervisorThreadThatCannotBeReadIsReported, and does not take the
// tasks with it.
func TestASupervisorThreadThatCannotBeReadIsReported(t *testing.T) {
	s := root(t)

	log(t, s, "ACME-1", record.Event{Kind: record.TaskCreated, Text: "Retry the webhook"})
	turns(t, s, record.Event{Kind: record.SupervisorMessage, Text: "start it"})

	path := s.SupervisorLogPath()

	t.Cleanup(func() {
		if err := os.Chmod(path, 0o600); err != nil {
			t.Errorf("give the thread back its mode: %v", err)
		}
	})

	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("take the thread's mode away: %v", err)
	}

	out, err := Run(s)
	if err == nil {
		t.Fatal("a supervisor thread that could not be read was not reported")
	}

	if out.Events != 1 {
		t.Errorf("the pass moved %d events, want the task's one", out.Events)
	}
}

// TestARecordThatWillNotOpenIsReported. The migration runs before every
// command, so it meets whatever is at that path.
func TestARecordThatWillNotOpenIsReported(t *testing.T) {
	s := root(t)

	if err := os.MkdirAll(s.DBPath(), 0o700); err != nil {
		t.Fatalf("put a directory where the record goes: %v", err)
	}

	if _, err := Run(s); err == nil {
		t.Error("a record that is a directory opened cleanly")
	}
}
