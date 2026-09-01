package migrate

// What a pass does when it meets something it cannot read. None of it stops
// the pass: a migration that gives up on the first damaged file leaves a
// state root half moved, which is the one shape nobody can reason about.

import (
	"os"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/db"
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

// TestAMarkerThatCannotBeReadDoesNotStopThePass. The link is carried across
// before the events, so a task whose marker file is unreadable is the first
// thing a pass meets — and it must cost that task its link and nothing else.
func TestAMarkerThatCannotBeReadDoesNotStopThePass(t *testing.T) {
	s := root(t)

	log(t, s, "ACME-1", record.Event{Kind: record.TaskCreated, Text: "the one whose marker is gone"})
	log(t, s, "ACME-2", record.Event{Kind: record.TaskCreated, Text: "the one that is fine"})

	marker, err := s.TaskReposPath("ACME-1")
	if err != nil {
		t.Fatalf("marker path: %v", err)
	}

	if err := os.Remove(marker); err != nil {
		t.Fatalf("remove the marker: %v", err)
	}

	// A directory where the file was: no permission bit to depend on, and
	// root would not notice one.
	if err := os.Mkdir(marker, 0o700); err != nil {
		t.Fatalf("put a directory where the marker goes: %v", err)
	}

	out, err := Run(s)
	if err == nil {
		t.Fatal("a marker that could not be read was not reported")
	}

	if !strings.Contains(err.Error(), "ACME-1") {
		t.Errorf("the error does not name the task whose marker could not be read: %v", err)
	}

	if out.Events != 2 {
		t.Errorf("the pass moved %s, want both tasks' events — the damaged marker stopped the rest", out)
	}

	joined, err := opened(t, s).ReposOfTask("ACME-2")
	if err != nil {
		t.Fatalf("the repositories of ACME-2: %v", err)
	}

	if len(joined) != 1 {
		t.Errorf("ACME-2 belongs to %v, want the repository it was worked in", joined)
	}
}

// TestARecordThatWillNotBeWrittenToIsReported. The pass reads three things —
// the log, the marker, and what the record already holds — and writes what
// is missing. A record that answers every read and refuses every write is
// the shape that tells the two apart, and it is not hypothetical: a state
// root restored from a backup arrives with the mode it was archived with.
func TestARecordThatWillNotBeWrittenToIsReported(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: a read-only file is still writable, so there is no fault to make")
	}

	s := root(t)

	log(t, s, "ACME-1", record.Event{Kind: record.TaskCreated, Text: "Retry the webhook"})

	// Made, let go of, and made read-only: the mode is checked when the file
	// is opened, so a handle already held keeps the permission it opened
	// with whatever the bit says afterwards.
	made, err := db.Open(s.DBPath())
	if err != nil {
		t.Fatalf("make the record: %v", err)
	}

	if err := made.Close(); err != nil {
		t.Fatalf("let go of the record: %v", err)
	}

	if err := os.Chmod(s.DBPath(), 0o444); err != nil {
		t.Fatalf("make the record read-only: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chmod(s.DBPath(), 0o600); err != nil {
			t.Errorf("give the record its mode back: %v", err)
		}
	})

	out, err := Records(s, opened(t, s))
	if err == nil {
		t.Fatal("a record that would take nothing reported a clean pass")
	}

	if !strings.Contains(err.Error(), "ACME-1") {
		t.Errorf("the error does not name the task that could not be moved: %v", err)
	}

	if out.Moved() {
		t.Errorf("the pass says it moved %s into a record that would take nothing", out)
	}
}
