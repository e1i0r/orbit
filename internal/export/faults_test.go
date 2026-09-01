package export

// The ways an export does not happen: a record that will not answer, an id
// no state root would write, and a destination that cannot be looked into,
// made, or written to.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/db"
	"github.com/e1i0r/orbit/internal/record"
)

// TestARecordThatWillNotAnswerIsRefused. Nothing is written when the list of
// tasks itself could not be read: an export of a record nobody could ask is
// an empty directory that looks exactly like a backup.
func TestARecordThatWillNotAnswerIsRefused(t *testing.T) {
	s := root(t)

	from, err := db.Open(s.DBPath())
	if err != nil {
		t.Fatalf("open the record: %v", err)
	}

	if err := from.Close(); err != nil {
		t.Fatalf("close the record: %v", err)
	}

	to := root(t)

	if _, err := Records(from, to, ""); err == nil {
		t.Fatal("an export off a record that could not be read reported success")
	}

	if _, err := os.Stat(to.SupervisorLogPath()); err == nil {
		t.Error("the export wrote a thread out of a record it could not read")
	}
}

// TestAnIdNoStateRootWouldWriteIsRefused. The record does not police task
// ids — it holds what it was given, including whatever an older version or
// another client wrote — and the tree it is exported into is made of
// directories named after them. The store is what says no, and the export
// carries on to the tasks that have ordinary names.
func TestAnIdNoStateRootWouldWriteIsRefused(t *testing.T) {
	_, from := filled(t)

	at := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	if err := from.Append("../escape", record.Event{At: at, Kind: record.TaskCreated, Text: "out of the tree"}); err != nil {
		t.Fatalf("write down a task the store would refuse: %v", err)
	}

	to := root(t)

	out, err := Records(from, to, "")
	if err == nil {
		t.Fatal("an export of a task named out of the tree reported success")
	}

	if !strings.Contains(err.Error(), "escape") {
		t.Errorf("the failure is %q, want the id that was refused in it", err)
	}

	if out.Tasks != 2 {
		t.Errorf("the export wrote %+v, want the two tasks that could be named", out)
	}

	if _, err := os.Stat(filepath.Join(filepath.Dir(to.Root()), "escape")); err == nil {
		t.Error("the export wrote a directory outside the tree it was given")
	}
}

// TestADestinationThatCannotBeLookedIntoIsRefused. The question asked of a
// destination is whether it holds anything, and a directory that will not
// answer it is not an answer of "nothing".
func TestADestinationThatCannotBeLookedIntoIsRefused(t *testing.T) {
	s, _ := filled(t)

	dir := t.TempDir()
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatalf("make a directory nobody can read: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Errorf("give the directory back: %v", err)
		}
	})

	if _, err := Run(s, dir, ""); err == nil {
		t.Fatal("an export into a directory it could not look into reported success")
	}
}

// TestADestinationThatCannotBeMadeIsRefused. A path under a file is the
// typo an export is most likely to be given, and it has to be refused before
// anything is written.
func TestADestinationThatCannotBeMadeIsRefused(t *testing.T) {
	s, _ := filled(t)

	blocked := filepath.Join(t.TempDir(), "backup")
	if err := os.WriteFile(blocked, []byte("a file, not a directory"), 0o600); err != nil {
		t.Fatalf("put a file where a directory would go: %v", err)
	}

	if _, err := Run(s, filepath.Join(blocked, "today"), ""); err == nil {
		t.Fatal("an export under a file reported success")
	}
}

// TestARecordThatWillNotOpenIsRefused. Run reaches for the store's own
// handle, and a state root whose record cannot be opened has nothing to
// export — which is different from having nothing in it.
func TestARecordThatWillNotOpenIsRefused(t *testing.T) {
	s := root(t)

	if err := os.MkdirAll(s.DBPath(), 0o700); err != nil {
		t.Fatalf("put a directory where the record goes: %v", err)
	}

	if _, err := Run(s, filepath.Join(t.TempDir(), "backup"), ""); err == nil {
		t.Fatal("an export off a record that could not be opened reported success")
	}
}

// TestWhatCouldNotBeWrittenIsNamed. The marker and the thread are the two
// files that are not a task's log, and a failure to write either has to
// reach the reader: an export missing the link between a task and its
// repository restores as a board with nothing on it.
func TestWhatCouldNotBeWrittenIsNamed(t *testing.T) {
	_, from := filled(t)

	to := root(t)
	block(t, to.SupervisorLogPath())

	marker, err := to.TaskReposPath("T-1")
	if err != nil {
		t.Fatalf("marker path: %v", err)
	}

	block(t, marker)

	out, err := Records(from, to, "")
	if err == nil {
		t.Fatal("an export that wrote neither the marker nor the thread reported success")
	}

	for _, want := range []string{"repos", "supervisor.jsonl"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the failure is %q, want %q named in it", err, want)
		}
	}

	if out.Tasks != 2 || out.Messages != 0 {
		t.Errorf("the export wrote %+v, want both logs and no thread", out)
	}
}

// block puts a directory where a file has to go, which is the shortest way
// to make one write fail and leave every other write alone.
func block(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("put a directory where %s goes: %v", filepath.Base(path), err)
	}
}
