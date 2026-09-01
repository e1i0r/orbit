package db

// Opening the record, and what happens when two processes want it at once.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// open is a record in a directory of this test's own, closed when the test
// ends. A close that fails is reported: an unclosed handle in a test is a
// file left locked for whatever runs next.
func open(t *testing.T) *DB {
	t.Helper()

	return openAt(t, filepath.Join(t.TempDir(), "orbit.db"))
}

// openAt is the same, at a path the caller chose — for the tests that need
// to open one file twice.
func openAt(t *testing.T, path string) *DB {
	t.Helper()

	d, err := Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}

	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Errorf("close %s: %v", path, err)
		}
	})

	return d
}

// clock hands out timestamps in the order they were asked for, so a test can
// say which event came first without sleeping.
func clock() func() time.Time {
	at := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)

	return func() time.Time {
		at = at.Add(time.Second)

		return at
	}
}

// TestOpenMakesTheFileAndItsDirectory. Nothing creates the state root before
// this: the first thing Orbit does on a new machine is write an event.
func TestOpenMakesTheFileAndItsDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "orbit.db")
	d := openAt(t, path)

	if d.Path() != path {
		t.Errorf("Path is %q, want %q", d.Path(), path)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat the record: %v", err)
	}

	// The record holds what every run of every engine printed. It is the
	// owner's to read and nobody else's.
	if perm := info.Mode().Perm(); perm != fileMode {
		t.Errorf("the record is mode %v, want %v", perm, fileMode)
	}
}

// TestTheCreatorAsksForWALAndNobodyElseDoes is the rule that cost a
// measurement to find. Setting journal_mode is a write, so a second process
// whose DSN asks for it cannot open the file at all while the first holds
// the lock. Only the process that made the file asks.
func TestTheCreatorAsksForWALAndNobodyElseDoes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orbit.db")
	first := openAt(t, path)

	var mode string
	if err := first.sql.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("read the journal mode: %v", err)
	}

	if mode != "wal" {
		t.Fatalf("the creator left the journal in %q, want wal", mode)
	}

	// The joining process, with the first one still holding its handle.
	second := openAt(t, path)

	if err := second.Append("ACME-1", record.Event{Kind: record.TaskCreated, Text: "Retry the webhook"}); err != nil {
		t.Fatalf("the second process could not write: %v", err)
	}

	events, err := first.Events("ACME-1")
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("the first process sees %d events, want the one the second wrote", len(events))
	}
}

// TestARecordNewerThanTheBinaryIsRefused. An older Orbit writing into a
// shape it does not know is how a column silently stops being filled, and
// the record is the one thing here that cannot be rebuilt.
func TestARecordNewerThanTheBinaryIsRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orbit.db")

	d, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if _, err := d.sql.Exec(`PRAGMA user_version = 99`); err != nil {
		t.Fatalf("stamp a later version: %v", err)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	again, err := Open(path)
	if err == nil {
		t.Fatalf("a record at version 99 opened; want a refusal, and closing it: %v", again.Close())
	}

	if got := err.Error(); !strings.Contains(got, "upgrade orbit") {
		t.Errorf("the refusal reads %q, want it to say what to do about it", got)
	}
}

// TestReopeningKeepsWhatWasWritten. Every orbit command is a process that
// starts, writes and dies, so the second open is the normal case and not an
// edge of it.
func TestReopeningKeepsWhatWasWritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orbit.db")

	d, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := d.Append("ACME-1", record.Event{Kind: record.TaskCreated, Text: "Retry the webhook"}); err != nil {
		t.Fatalf("append: %v", err)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	events, err := openAt(t, path).Events("ACME-1")
	if err != nil {
		t.Fatalf("read back after reopening: %v", err)
	}

	if len(events) != 1 || events[0].Text != "Retry the webhook" {
		t.Errorf("after reopening the record holds %v, want the one event", events)
	}
}

// TestOpeningSomethingThatIsNotARecordFails. A path pointing at a directory,
// or at a file of something else, is a mistake worth hearing about at the
// door rather than at the first write.
func TestOpeningSomethingThatIsNotARecordFails(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "not-a-record")
	if err := os.WriteFile(path, []byte("this is not a database\n"), 0o600); err != nil {
		t.Fatalf("write the file: %v", err)
	}

	d, err := Open(path)
	if err == nil {
		t.Fatalf("a text file opened as a record, and closing it: %v", d.Close())
	}
}

// TestTheSchemaVersionIsStamped. A file with no version is a file the next
// migration cannot place.
func TestTheSchemaVersionIsStamped(t *testing.T) {
	d := open(t)

	var found int
	if err := d.sql.QueryRow(`PRAGMA user_version`).Scan(&found); err != nil {
		t.Fatalf("read the version: %v", err)
	}

	if found != version {
		t.Errorf("the record is stamped %d, want %d", found, version)
	}
}

// TestForeignKeysAreOn. Every table here points at another one, and a
// foreign key nobody checks is a comment.
func TestForeignKeysAreOn(t *testing.T) {
	d := open(t)

	var on int
	if err := d.sql.QueryRow(`PRAGMA foreign_keys`).Scan(&on); err != nil {
		t.Fatalf("read the setting: %v", err)
	}

	if on != 1 {
		t.Error("foreign keys are off")
	}
}

// TestARefusalToTakeATurnIsTheOnlyThingRetried. Everything else fails the
// same way twice, and asking again only costs time.
func TestARefusalToTakeATurnIsTheOnlyThingRetried(t *testing.T) {
	for _, c := range []struct {
		text string
		want bool
	}{
		{"SQLITE_BUSY: database is busy", true},
		{"database is locked", true},
		{"no such column: banana", false},
		{"UNIQUE constraint failed", false},
	} {
		if got := refused(errText(c.text)); got != c.want {
			t.Errorf("refused(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}

// errText is an error that says exactly what it was given, which is what
// refused reads.
type errText string

func (e errText) Error() string { return string(e) }
