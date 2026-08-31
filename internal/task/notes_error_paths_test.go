package task

// The two error returns of unconsumedNotes, both of which travel rather than
// being answered around.
//
// Returning nil and carrying on buys a phase that starts with the operator's
// correction missing from its prompt and nothing anywhere saying so, which
// is the failure this package refuses by name in Supervise.

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
)

// TestNotesThatCannotBeReadAreNotReportedAsNoNotes.
func TestNotesThatCannotBeReadAreNotReportedAsNoNotes(t *testing.T) {
	s, r := fixture(t)

	// 1. Bad id: EventsPath itself fails.
	bad := Task{ID: "has/slash", Repo: r}
	if notes, err := unconsumedNotes(s, bad); err == nil {
		t.Errorf("unconsumedNotes on a bad id = %v, nil — want the path error", notes)
	}

	// 2. A log record.Read cannot parse at all: one line longer than
	// record.MaxLine, which trips the scanner rather than yielding a
	// record.unreadable event.
	tk, err := Create(s, r, "NOTES-ERR-1", "notes error test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	path, err := s.EventsPath(tk.ID)
	if err != nil {
		t.Fatalf("EventsPath: %v", err)
	}

	oversized := strings.Repeat("x", 5<<20) // over record.MaxLine (4 MiB)
	if err := os.WriteFile(path, []byte(oversized+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if notes, err := unconsumedNotes(s, tk); err == nil {
		t.Errorf("unconsumedNotes over an unreadable log = %v, nil — want the read error", notes)
	}
}

// TestAPhaseIsNotRunWithoutTheNotesItWasMeantToCarry is the reason the error
// travels at all: a run whose corrections could not be read is a run about
// to do the thing the operator wrote a note to stop.
func TestAPhaseIsNotRunWithoutTheNotesItWasMeantToCarry(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "NOTES-RUN-1", "notes on an unreadable log", "quick")
	if err != nil {
		t.Fatal(err)
	}

	path, err := s.EventsPath(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	// One line over record.MaxLine, which trips the scanner rather than
	// yielding a record.unreadable event. Appending still works, so the run
	// gets as far as the phase loop and stops there.
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 5<<20)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	f := flow.Flow{Name: "quick", Phases: []flow.Phase{{Name: "phase-1", Engine: "fake"}}}
	engines := map[string]engine.Engine{"fake": engine.NewFake("out")}

	err = Run(context.Background(), s, tk, f, engines, nil)
	if err == nil {
		t.Fatal("a phase ran with the operator notes silently missing from its prompt")
	}

	if !strings.Contains(err.Error(), "phase-1") {
		t.Errorf("the failure is %q, and it does not name the phase that did not run", err)
	}
}
