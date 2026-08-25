package task

// The two error returns of unconsumedNotes, both of which answer nil rather
// than propagating a fault — see its doc comment for why a phase starting
// must not be blocked by a record it cannot read.

import (
	"os"
	"strings"
	"testing"
)

func TestUnconsumedNotesErrorPaths(t *testing.T) {
	s, r := fixture(t)

	// 1. Bad id: EventsPath itself fails.
	bad := Task{ID: "has/slash", Repo: r}
	if notes := unconsumedNotes(s, bad); notes != nil {
		t.Errorf("unconsumedNotes on a bad id = %v, want nil", notes)
	}

	// 2. A log record.Read cannot parse at all: one line longer than
	// record.MaxLine, which trips the scanner rather than yielding a
	// record.unreadable event.
	tk, err := Create(s, r, "NOTES-ERR-1", "notes error test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	path, err := s.EventsPath(r.Path, tk.ID)
	if err != nil {
		t.Fatalf("EventsPath: %v", err)
	}
	oversized := strings.Repeat("x", 5<<20) // over record.MaxLine (4 MiB)
	if err := os.WriteFile(path, []byte(oversized+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if notes := unconsumedNotes(s, tk); notes != nil {
		t.Errorf("unconsumedNotes over an unreadable log = %v, want nil", notes)
	}
}
