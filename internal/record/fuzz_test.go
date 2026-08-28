package record

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestRecordAppendReadRoundTripProperty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	fixedTime := time.Date(2026, 8, 24, 15, 0, 0, 0, time.UTC)
	events := []Event{
		{At: fixedTime, Kind: TaskCreated, Text: "Initial task description"},
		{At: fixedTime.Add(time.Minute), Kind: PhaseStarted, Phase: "implement", Data: map[string]string{"engine": "claude", "model": "sonnet"}},
		{At: fixedTime.Add(2 * time.Minute), Kind: PhaseFinished, Phase: "implement", Text: "Code written", Data: map[string]string{"cost": "0.05"}},
		{At: fixedTime.Add(3 * time.Minute), Kind: TaskFinished},
	}

	for _, e := range events {
		if err := Append(path, e); err != nil {
			t.Fatalf("Append %s: %v", e.Kind, err)
		}
	}

	// 1. Read full log
	readEvents, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(readEvents) != len(events) {
		t.Fatalf("read %d events, want %d", len(readEvents), len(events))
	}

	for i, want := range events {
		got := readEvents[i]
		if got.Kind != want.Kind || got.Phase != want.Phase || got.Text != want.Text {
			t.Errorf("event %d = %+v, want %+v", i, got, want)
		}
		if len(want.Data) > 0 && !reflect.DeepEqual(got.Data, want.Data) {
			t.Errorf("event %d data = %+v, want %+v", i, got.Data, want.Data)
		}
	}

	// 2. Incremental ReadFrom
	firstChunk, offset, err := ReadFrom(path, 0)
	if err != nil || len(firstChunk) != 4 {
		t.Fatalf("ReadFrom offset 0 failed: %v, len=%d", err, len(firstChunk))
	}

	// Append one more event
	extraEvent := Event{At: fixedTime.Add(4 * time.Minute), Kind: TaskNoted, Text: "Followup note"}
	if err := Append(path, extraEvent); err != nil {
		t.Fatalf("Append extra: %v", err)
	}

	secondChunk, _, err := ReadFrom(path, offset)
	if err != nil || len(secondChunk) != 1 || secondChunk[0].Kind != TaskNoted {
		t.Fatalf("ReadFrom incremental failed: %v, chunk=%+v", err, secondChunk)
	}
}

func TestRecordUnreadableLineTracking(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.jsonl")

	corruptData := "{\"at\":\"2026-08-24T12:00:00Z\",\"kind\":\"task.created\"}\n{broken json\n{\"at\":\"2026-08-24T12:02:00Z\",\"kind\":\"task.finished\"}\n"
	if err := os.WriteFile(path, []byte(corruptData), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	events, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events including unreadable line, got %d", len(events))
	}

	if events[1].Kind != Unreadable || events[1].Data["byte"] != "52" {
		t.Errorf("expected unreadable line 2, got %+v", events[1])
	}
}

func TestRecordMaxLineAndEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	// 1. Zero time gets auto-populated
	if err := Append(path, Event{Kind: TaskCreated}); err != nil {
		t.Fatalf("Append with zero time: %v", err)
	}

	// 2. Line over MaxLine gets rejected
	hugeText := string(make([]byte, MaxLine+100))
	err := Append(path, Event{Kind: TaskCreated, Text: hugeText})
	if err == nil {
		t.Error("expected Append with huge text over MaxLine to fail")
	}

	// 3. Read empty file (0 bytes)
	emptyPath := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(emptyPath, nil, 0o644); err != nil {
		t.Fatalf("WriteFile empty: %v", err)
	}
	events, err := Read(emptyPath)
	if err != nil || len(events) != 0 {
		t.Errorf("Read empty file failed: %v, events=%+v", err, events)
	}
}

func FuzzRecordScanner(f *testing.F) {
	f.Add([]byte(`{"at":"2026-08-24T12:00:00Z","kind":"task.created"}` + "\n"))
	f.Add([]byte(`{"at":"invalid","kind":""}` + "\n"))
	f.Add([]byte("not json at all\n"))
	f.Add([]byte("\n\n\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = scanEvents(bytes.NewReader(data), 0) //nolint:errcheck // fuzz scanner against arbitrary data
	})
}
