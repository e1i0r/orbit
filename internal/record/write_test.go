package record

// Writing a whole log at once: the same bytes Append leaves, put down in one
// go, which is what an export is.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// three is a log with something of every field in it, so that a comparison
// of bytes is a comparison of the whole encoding and not of the two fields
// that happen to be set.
func three() []Event {
	at := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)

	return []Event{
		{At: at, Kind: "task.created", Text: "retry the webhook on 5xx"},
		{At: at.Add(time.Second), Kind: "phase.started", Phase: "implement"},
		{At: at.Add(2 * time.Second), Kind: "phase.finished", Phase: "implement", Data: map[string]string{"exit": "0"}},
	}
}

// TestWriteLeavesTheBytesAppendWouldHave. The promise of an export is that
// what comes out is what used to be on disk, and the only way to hold two
// encoders to one answer is to compare the files they produce.
func TestWriteLeavesTheBytesAppendWouldHave(t *testing.T) {
	dir := t.TempDir()
	appended := filepath.Join(dir, "appended.jsonl")
	written := filepath.Join(dir, "written.jsonl")

	events := three()
	for _, e := range events {
		if err := Append(appended, e); err != nil {
			t.Fatalf("append %s: %v", e.Kind, err)
		}
	}

	if err := Write(written, events); err != nil {
		t.Fatalf("write: %v", err)
	}

	want, err := os.ReadFile(appended)
	if err != nil {
		t.Fatalf("read the appended log: %v", err)
	}

	got, err := os.ReadFile(written)
	if err != nil {
		t.Fatalf("read the written log: %v", err)
	}

	if string(got) != string(want) {
		t.Errorf("written:\n%s\nappended:\n%s", got, want)
	}
}

// TestWrittenLogsReadBackAsThemselves. The file the export leaves is read by
// the migration, so the reader that matters is this package's own.
func TestWrittenLogsReadBackAsThemselves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")

	want := three()
	if err := Write(path, want); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("read %d events, wrote %d", len(got), len(want))
	}

	for i := range want {
		if !got[i].At.Equal(want[i].At) || got[i].Kind != want[i].Kind || got[i].Text != want[i].Text {
			t.Errorf("event %d read back as %+v, want %+v", i, got[i], want[i])
		}
	}

	if got[2].Data["exit"] != "0" {
		t.Errorf("the data of the last event is %v, want the map it was written with", got[2].Data)
	}
}

// TestWriteMakesTheDirectoryItWritesInto. An export names a directory that
// does not exist yet, and every task in it is a directory that does not
// exist yet either.
func TestWriteMakesTheDirectoryItWritesInto(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup", "tasks", "T-1", "events.jsonl")

	if err := Write(path, three()); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat what was written: %v", err)
	}
}

// TestAnEmptyLogIsAnEmptyFile. A task somebody wrote down and never ran has
// no events, and its file is what says the id is taken.
func TestAnEmptyLogIsAnEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")

	if err := Write(path, nil); err != nil {
		t.Fatalf("write: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if len(body) != 0 {
		t.Errorf("a log of no events is %q, want nothing at all", body)
	}
}

// TestWriteReplacesWhatWasThere. Twice into the same file is one log and not
// two: an export is the whole of a task's history, so a shorter one landing
// on a longer one must not leave the tail of the longer one behind it.
func TestWriteReplacesWhatWasThere(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")

	if err := Write(path, three()); err != nil {
		t.Fatalf("write three: %v", err)
	}

	if err := Write(path, three()[:1]); err != nil {
		t.Fatalf("write one: %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if len(got) != 1 {
		t.Errorf("the file holds %d events, want the one that was written last", len(got))
	}
}

// TestWriteSaysWhereItCouldNotWrite. A directory that cannot be made is the
// export failing before it started, and the path is the whole of what a
// reader needs to know about it.
func TestWriteSaysWhereItCouldNotWrite(t *testing.T) {
	dir := t.TempDir()
	blocked := filepath.Join(dir, "tasks")

	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("put a file where a directory has to go: %v", err)
	}

	path := filepath.Join(blocked, "T-1", "events.jsonl")

	err := Write(path, three())
	if err == nil {
		t.Fatal("writing under a file succeeded")
	}

	if !strings.Contains(err.Error(), "T-1") {
		t.Errorf("the failure is %q, want the directory it could not make in it", err)
	}
}

// TestWriteSaysWhenTheLogIsNotAFile. A directory where the log goes is the
// shape a state root takes when something else made the path first, and the
// write has to name it rather than report a log that is not there.
func TestWriteSaysWhenTheLogIsNotAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")

	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("put a directory where the log goes: %v", err)
	}

	err := Write(path, three())
	if err == nil {
		t.Fatal("writing a log over a directory succeeded")
	}

	if !strings.Contains(err.Error(), path) {
		t.Errorf("the failure is %q, want the path it could not open in it", err)
	}
}
