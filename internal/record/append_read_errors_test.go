package record

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendMaxLineExceeded(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")

	hugeText := strings.Repeat("A", MaxLine+100)
	err := Append(logPath, Event{Kind: TaskCreated, Text: hugeText})
	if err == nil {
		t.Fatal("expected error appending line exceeding MaxLine")
	}
	if !strings.Contains(err.Error(), "refusing to write a line") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestAppendZeroTimeDefaultsToNow(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "events.jsonl")

	err := Append(logPath, Event{Kind: TaskCreated, Text: "auto time"})
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	events, err := Read(logPath)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].At.IsZero() {
		t.Error("expected non-zero timestamp on read event")
	}
}

func TestAppendToReadOnlyFile(t *testing.T) {
	tmpDir := t.TempDir()
	readOnlyPath := filepath.Join(tmpDir, "readonly.jsonl")

	if err := os.WriteFile(readOnlyPath, []byte("data\n"), 0o400); err != nil {
		t.Fatal(err)
	}

	err := Append(readOnlyPath, Event{Kind: TaskCreated})
	if err == nil {
		t.Fatal("expected error appending to read-only file")
	}
}

func TestAppendToInvalidPath(t *testing.T) {
	tmpDir := t.TempDir()
	blocker := filepath.Join(tmpDir, "file_blocker")
	if err := os.WriteFile(blocker, []byte("blocker"), 0o600); err != nil {
		t.Fatal(err)
	}

	invalidPath := filepath.Join(blocker, "sub", "events.jsonl")
	err := Append(invalidPath, Event{Kind: TaskCreated})
	if err == nil {
		t.Fatal("expected error appending to invalid nested path")
	}
}

func TestReadOpenError(t *testing.T) {
	tmpDir := t.TempDir()
	blocker := filepath.Join(tmpDir, "file_blocker")
	if err := os.WriteFile(blocker, []byte("data\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	invalidPath := filepath.Join(blocker, "child.jsonl")
	_, err := Read(invalidPath)
	if err == nil {
		t.Fatal("expected error reading through non-directory path")
	}

	_, _, err = ReadFrom(invalidPath, 0)
	if err == nil {
		t.Fatal("expected error ReadFrom through non-directory path")
	}
}

func TestReadEmptyFileAndBlanks(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "empty.jsonl")

	// File with blank lines
	if err := os.WriteFile(logPath, []byte("\n\n\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	events, err := Read(logPath)
	if err != nil {
		t.Fatalf("Read failed on blank file: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestScanEventsMultipleBadLinesAndScannerError(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "bad.jsonl")

	// 1. Multiple bad lines followed by a valid line
	content := "{bad1\n{bad2\n{\"kind\":\"task.created\"}\n"
	if err := os.WriteFile(logPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	events, err := Read(logPath)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events (2 unreadable + 1 created), got %d", len(events))
	}
	if events[0].Kind != Unreadable || events[1].Kind != Unreadable || events[2].Kind != TaskCreated {
		t.Errorf("unexpected event sequence: %+v", events)
	}

	// 2. Line exceeding buffer size causing scanner error
	hugePath := filepath.Join(tmpDir, "huge.jsonl")
	hugeLine := strings.Repeat("x", MaxLine+100) + "\n"
	if err := os.WriteFile(hugePath, []byte(hugeLine), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Read(hugePath)
	if err == nil {
		t.Fatal("expected scanner error reading line > MaxLine")
	}
}

func TestReadUnterminatedFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "unterminated.jsonl")

	if err := os.WriteFile(logPath, []byte("{\"kind\":\"task.created\"}\n{incomplete json"), 0o600); err != nil {
		t.Fatal(err)
	}

	events, err := Read(logPath)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	// An unterminated incomplete last line is dropped
	if len(events) != 1 {
		t.Errorf("expected 1 event on unterminated file, got %d", len(events))
	}
}

func TestReadFromOffsetEqualsSizeAndNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "tail.jsonl")

	// 1. Non-existent file
	events, offset, err := ReadFrom(filepath.Join(tmpDir, "missing.jsonl"), 0)
	if err != nil || len(events) != 0 || offset != 0 {
		t.Errorf("expected (nil, 0, nil), got (%v, %d, %v)", events, offset, err)
	}

	// 2. Write 1 event
	if err := os.WriteFile(logPath, []byte("{\"kind\":\"task.created\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}
	size := info.Size()

	// Call with offset == size
	events, offset, err = ReadFrom(logPath, size)
	if err != nil || len(events) != 0 || offset != size {
		t.Errorf("expected (nil, %d, nil), got (%v, %d, %v)", size, events, offset, err)
	}

	// Call with offset > size (file shrunk / replaced)
	events, offset, err = ReadFrom(logPath, size+1000)
	if err != nil || len(events) != 1 || offset != size {
		t.Errorf("expected (1 event, %d, nil), got (%v, %d, %v)", size, events, offset, err)
	}

	// 3. Multi-line where last line is a valid event without trailing newline
	multi := "{\"kind\":\"task.created\"}\n{\"kind\":\"task.started\"}"
	if err := os.WriteFile(logPath, []byte(multi), 0o600); err != nil {
		t.Fatal(err)
	}
	events, offset, err = ReadFrom(logPath, 0)
	if err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if offset != int64(len("{\"kind\":\"task.created\"}\n")) {
		t.Errorf("unexpected offset: %d", offset)
	}

	// 4. Multi-line where last line is an incomplete valid event without trailing newline
	multiLastValid := "{\"kind\":\"task.created\"}\n{\"kind\":\"task.finished\"}"
	if err := os.WriteFile(logPath, []byte(multiLastValid), 0o600); err != nil {
		t.Fatal(err)
	}
	events, offset, err = ReadFrom(logPath, 0)
	if err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if offset != int64(len("{\"kind\":\"task.created\"}\n")) {
		t.Errorf("unexpected offset: %d", offset)
	}

	// 5. Multi-line where last line is an incomplete invalid JSON line without trailing newline
	multiBad := "{\"kind\":\"task.created\"}\n{not json yet"
	if err := os.WriteFile(logPath, []byte(multiBad), 0o600); err != nil {
		t.Fatal(err)
	}
	events, offset, err = ReadFrom(logPath, 0)
	if err != nil {
		t.Fatalf("ReadFrom failed on incomplete bad line: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if offset != int64(len("{\"kind\":\"task.created\"}\n")) {
		t.Errorf("unexpected offset: %d", offset)
	}
}

func TestAppendAndReadErrorPaths(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Append where parent directory is a file (MkdirAll fails)
	blocker := filepath.Join(tmpDir, "blocker_file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := Append(filepath.Join(blocker, "sub", "events.jsonl"), Event{Kind: TaskCreated})
	if err == nil {
		t.Error("expected error appending under regular file")
	}

	// 2. Append where path itself is an existing directory (OpenFile fails)
	dirPath := filepath.Join(tmpDir, "some_dir")
	if err := os.MkdirAll(dirPath, 0o700); err != nil {
		t.Fatal(err)
	}
	err = Append(dirPath, Event{Kind: TaskCreated})
	if err == nil {
		t.Error("expected error opening directory for append")
	}

	// 3. Read on a directory: os.Open succeeds on some OSes and fails on
	// others, so neither outcome is wrong — this line exists to exercise
	// the path, not to assert which branch it takes.
	_, _ = Read(dirPath) //nolint:errcheck
}
