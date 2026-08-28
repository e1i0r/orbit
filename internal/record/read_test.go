package record

// The reading side of the log, and in particular what Read does with a line
// it cannot parse: a truncated last line is an interrupted write and stays
// dropped, while damage anywhere else leaves a visible mark.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestReadOfAMissingFileIsEmptyNotAnError(t *testing.T) {
	got, err := Read(filepath.Join(t.TempDir(), "nothing.jsonl"))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d events from a file that does not exist", len(got))
	}
}

func TestReadSkipsTruncatedTrailingLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	// Write two good events using Append
	e1 := Event{Kind: "task.created", Text: "first"}
	e2 := Event{Kind: "task.updated", Text: "second"}
	if err := Append(path, e1); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := Append(path, e2); err != nil {
		t.Fatalf("Append: %v", err)
	}
	// Manually append a truncated JSON fragment with no trailing newline
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, err = f.WriteString(`{"kind":"task.failed"`)
	f.Close()
	if err != nil {
		t.Fatalf("write truncated: %v", err)
	}
	// Read should return the two good events and no error
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}
	if got[0].Kind != "task.created" || got[1].Kind != "task.updated" {
		t.Errorf("events = %+v, want first two with Kind task.created and task.updated", got)
	}
	for _, e := range got {
		if e.Kind == Unreadable {
			t.Error("a write interrupted mid-flight was reported as damage — it is the ordinary shape of a crash, not a corrupt record")
		}
	}
}

func TestReadMarksAMalformedLineInTheMiddle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	// Append a good event using Append
	e1 := Event{Kind: "task.created", Text: "first"}
	if err := Append(path, e1); err != nil {
		t.Fatalf("Append: %v", err)
	}
	// Manually append a malformed line
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, err = f.WriteString("{this is not json}\n")
	f.Close()
	if err != nil {
		t.Fatalf("write malformed: %v", err)
	}
	// Append another good event
	e2 := Event{Kind: "task.updated", Text: "second"}
	if err := Append(path, e2); err != nil {
		t.Fatalf("Append: %v", err)
	}
	// The corrupt line is neither hidden nor fatal: it is reported where
	// it sat, so the gap is visible instead of silently closed up.
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d events, want 3 — the corrupt line must leave a mark, not vanish", len(got))
	}
	if got[0].Kind != "task.created" || got[2].Kind != "task.updated" {
		t.Errorf("events = %+v, want the two good events either side", got)
	}
	if got[1].Kind != Unreadable {
		t.Fatalf("event 1 = %q, want %q in the dropped line's position", got[1].Kind, Unreadable)
	}
	// The mark says where to go and look, and the only useful test of that
	// is to go and look: the byte it names has to be the first byte of the
	// line nobody could parse.
	at, err := strconv.ParseInt(got[1].Data["byte"], 10, 64)
	if err != nil {
		t.Fatalf(`Data["byte"] = %q: %v`, got[1].Data["byte"], err)
	}
	whole, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if at < 0 || at >= int64(len(whole)) {
		t.Fatalf("the mark names byte %d of a %d byte log", at, len(whole))
	}
	line, _, _ := strings.Cut(string(whole[at:]), "\n")
	if line != "{this is not json}" {
		t.Errorf("byte %d begins %q, want the malformed line", at, line)
	}
}

func TestReadSkipsBlankLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	// Append a good event using Append
	e1 := Event{Kind: "task.created", Text: "first"}
	if err := Append(path, e1); err != nil {
		t.Fatalf("Append: %v", err)
	}
	// Manually append a blank line
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, err = f.WriteString("\n")
	f.Close()
	if err != nil {
		t.Fatalf("write blank: %v", err)
	}
	// Append another good event
	e2 := Event{Kind: "task.updated", Text: "second"}
	if err := Append(path, e2); err != nil {
		t.Fatalf("Append: %v", err)
	}
	// Read should return the two good events and no error
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}
	if got[0].Kind != "task.created" || got[1].Kind != "task.updated" {
		t.Errorf("events = %+v, want first and third with Kind task.created and task.updated", got)
	}
}
