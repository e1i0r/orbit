package record

// ReadFrom's own tests. read_test.go pins Read's torn-line handling; this
// file pins that ReadFrom's offset never advances past a line that is not
// yet finished, and covers everything else offset tracking has to get
// right along the way.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFromOfAMissingFileIsEmptyNotAnError(t *testing.T) {
	got, next, err := ReadFrom(filepath.Join(t.TempDir(), "nothing.jsonl"), 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("got %d events from a file that does not exist", len(got))
	}

	if next != 0 {
		t.Errorf("next = %d, want 0", next)
	}
}

func TestReadFromSeesAFileThatDidNotExistAndThenDoes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")

	got, next, err := ReadFrom(path, 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(got) != 0 || next != 0 {
		t.Fatalf("got %d events, next %d, before the file existed — want 0, 0", len(got), next)
	}

	if err := Append(path, Event{Kind: "task.created"}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, next, err = ReadFrom(path, next)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d events after the file was created, want 1", len(got))
	}

	if got[0].Kind != "task.created" {
		t.Errorf("Kind = %q, want task.created", got[0].Kind)
	}

	if next == 0 {
		t.Error("next did not move past the event that was read")
	}
}

func TestReadFromReturnsOnlyWhatWasAppendedSinceOffset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := Append(path, Event{Kind: "task.created"}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	_, next, err := ReadFrom(path, 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if err := Append(path, Event{Kind: "task.updated"}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, newNext, err := ReadFrom(path, next)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d events, want 1 — only what was appended after the offset", len(got))
	}

	if got[0].Kind != "task.updated" {
		t.Errorf("Kind = %q, want task.updated", got[0].Kind)
	}

	if newNext <= next {
		t.Errorf("next = %d, want it to advance past %d", newNext, next)
	}
}

func TestReadFromReadsNothingWhenAlreadyCaughtUp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := Append(path, Event{Kind: "task.created"}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	_, next, err := ReadFrom(path, 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	got, same, err := ReadFrom(path, next)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("got %d events with nothing new appended, want 0", len(got))
	}

	if same != next {
		t.Errorf("next = %d, want unchanged %d", same, next)
	}
}

func TestReadFromHoldsTheOffsetBackAtATornLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if _, err := f.WriteString(`{"kind":"task.c`); err != nil {
		t.Fatalf("write half: %v", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	got, next, err := ReadFrom(path, 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("got %d events from a torn line, want 0", len(got))
	}

	if next != 0 {
		t.Errorf("next = %d, want 0 — a torn line must not move the offset", next)
	}

	f, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if _, err := f.WriteString(`reated","text":"first"}` + "\n"); err != nil {
		t.Fatalf("write rest: %v", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	got, next, err = ReadFrom(path, next)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d events once the line was finished, want 1", len(got))
	}

	if got[0].Kind != "task.created" || got[0].Text != "first" {
		t.Errorf("event = %+v, want Kind task.created, Text first", got[0])
	}

	if next == 0 {
		t.Error("next did not move past the finished line")
	}
}

func TestReadFromWithholdsAValidLineThatHasNoTrailingNewlineYet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := os.WriteFile(path, []byte(`{"kind":"task.created"}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, next, err := ReadFrom(path, 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("got %d events for a line that parses but has not been terminated yet, want 0", len(got))
	}

	if next != 0 {
		t.Errorf("next = %d, want 0 — an unterminated line must not move the offset even if it happens to parse", next)
	}

	if err := os.WriteFile(path, []byte(`{"kind":"task.created"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write final newline: %v", err)
	}

	got, next, err = ReadFrom(path, next)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d events once terminated, want 1", len(got))
	}

	if next == 0 {
		t.Error("next did not move past the now-finished line")
	}
}

func TestReadFromMarksAMalformedTerminatedLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := Append(path, Event{Kind: "task.created"}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if _, err := f.WriteString("{this is not json}\n"); err != nil {
		t.Fatalf("write malformed: %v", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	got, next, err := ReadFrom(path, 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d events, want 2 — the malformed but terminated line must leave a mark, not vanish", len(got))
	}

	if got[1].Kind != Unreadable {
		t.Errorf("event 1 = %q, want %q", got[1].Kind, Unreadable)
	}

	if next == 0 {
		t.Error("next did not move past the malformed but terminated line")
	}
}

func TestReadFromRefusesALineOverTheCapWithoutAPartialRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := Append(path, Event{Kind: "task.created"}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if _, err := f.WriteString(strings.Repeat("x", MaxLine+1) + "\n"); err != nil {
		t.Fatalf("write oversized: %v", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	got, next, err := ReadFrom(path, 0)
	if err == nil {
		t.Fatal("ReadFrom accepted a line over MaxLine")
	}

	if got != nil {
		t.Errorf("got %d events on a scanner error, want nil, not a partial read", len(got))
	}

	if next != 0 {
		t.Errorf("next = %d, want 0 on a scanner error", next)
	}
}

func TestReadFromStartsOverWhenTheLogWasReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := Append(path, Event{Kind: "task.created"}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	if err := Append(path, Event{Kind: "task.updated"}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	_, next, err := ReadFrom(path, 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	// The log is replaced with something shorter than where the reader had
	// gotten to — a brand new file landing at the same path.
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}

	if err := Append(path, Event{Kind: "task.created"}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, newNext, err := ReadFrom(path, next)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d events from the replaced log, want 1", len(got))
	}

	if got[0].Kind != "task.created" {
		t.Errorf("Kind = %q, want task.created", got[0].Kind)
	}

	if newNext >= next {
		t.Errorf("next = %d, want less than %d — the caller must be able to see the offset moved backwards", newNext, next)
	}
}
