package task

// The two error returns of unconsumedNotes, both of which travel rather than
// being answered around.
//
// Returning nil and carrying on buys a phase that starts with the operator's
// correction missing from its prompt and nothing anywhere saying so, which
// is the failure this package refuses by name in Supervise.

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/store"
)

// TestNotesThatCannotBeReadAreNotReportedAsNoNotes.
func TestNotesThatCannotBeReadAreNotReportedAsNoNotes(t *testing.T) {
	s, r := fixture(t)

	// 1. Bad id: the record refuses to be asked about it at all.
	bad := Task{ID: "has/slash", Repo: r}
	if notes, err := unconsumedNotes(s, bad); err == nil {
		t.Errorf("unconsumedNotes on a bad id = %v, nil — want the path error", notes)
	}

	// 2. A record that cannot be reached, which is what a damaged row is not:
	// a row that will not parse comes back as record.unreadable and the notes
	// around it still read.
	tk, err := Create(s, r, "NOTES-ERR-1", "notes error test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	breakRecord(t, s)

	if notes, err := unconsumedNotes(s, tk); err == nil {
		t.Errorf("unconsumedNotes over an unreadable log = %v, nil — want the read error", notes)
	}
}

// TestAPhaseIsNotRunWithoutTheNotesItWasMeantToCarry is the reason the error
// travels at all: a run whose corrections could not be read is a run about
// to do the thing the operator wrote a note to stop.
func TestAPhaseIsNotRunWithoutTheNotesItWasMeantToCarry(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "NOTES-RUN-1", "notes on an unreadable record", "quick")
	if err != nil {
		t.Fatal(err)
	}

	// The record is taken away by the gate, which is the one moment that
	// isolates this: the run has started and been written down, the phase has
	// been let past, and the very next thing it does is read the notes. Taking
	// it away any earlier would fail the run over task.started instead, which
	// is a different branch and has its own test.
	gate := breakingGate{t: t, store: s}

	f := flow.Flow{Name: "quick", Phases: []flow.Phase{{Name: "phase-1", Engine: "fake"}}}
	engines := map[string]engine.Engine{"fake": engine.NewFake("out")}

	err = Run(context.Background(), s, tk, f, engines, gate)
	if err == nil {
		t.Fatal("a phase ran with the operator notes silently missing from its prompt")
	}

	if !strings.Contains(err.Error(), "phase-1") {
		t.Errorf("the failure is %q, and it does not name the phase that did not run", err)
	}
}

// breakingGate lets every phase past and takes the record away on the way.
type breakingGate struct {
	t     *testing.T
	store *store.Store
}

func (g breakingGate) Before(_ context.Context, _ Task, _ flow.Phase, _ int) (Go, error) {
	breakRecord(g.t, g.store)

	return Continue, nil
}
