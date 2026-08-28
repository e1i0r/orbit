package task

import (
	"context"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

func TestLastSessionReturnsEmptyWhenNilOrNotResumable(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "SESS-1", "session test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sess := lastSession(nil, tk, &engine.Fake{}); sess != "" {
		t.Errorf("lastSession on nil store = %q, want empty", sess)
	}
	if sess := lastSession(s, tk, nil); sess != "" {
		t.Errorf("lastSession on nil engine = %q, want empty", sess)
	}
	if sess := lastSession(s, tk, &engine.Fake{Resumable: false}); sess != "" {
		t.Errorf("lastSession on non-resumable fake = %q, want empty", sess)
	}
}

func TestLastSessionFindsMostRecentSessionID(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "SESS-2", "session test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	fake := &engine.Fake{Resumable: true}

	// No session events yet
	if sess := lastSession(s, tk, fake); sess != "" {
		t.Errorf("lastSession before runs = %q, want empty", sess)
	}

	// Emit PhaseFinished with session "sess-alpha"
	ev1 := record.Event{
		Kind:  record.PhaseFinished,
		Phase: "plan",
		Data:  map[string]string{"session": "sess-alpha"},
	}
	if err := emit(s, tk, ev1); err != nil {
		t.Fatalf("emit ev1: %v", err)
	}
	if sess := lastSession(s, tk, fake); sess != "sess-alpha" {
		t.Errorf("lastSession = %q, want sess-alpha", sess)
	}

	// Emit PhaseCancelled with session "sess-beta"
	ev2 := record.Event{
		Kind:  record.PhaseCancelled,
		Phase: "impl",
		Data:  map[string]string{"session": "sess-beta"},
	}
	if err := emit(s, tk, ev2); err != nil {
		t.Fatalf("emit ev2: %v", err)
	}
	if sess := lastSession(s, tk, fake); sess != "sess-beta" {
		t.Errorf("lastSession after cancel = %q, want sess-beta", sess)
	}
}

func TestRunPassesLastSessionToResumableEngine(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "SESS-3", "run session test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Emit an initial session
	ev := record.Event{
		Kind:  record.PhaseFinished,
		Phase: "setup",
		Data:  map[string]string{"session": "sess-existing"},
	}
	if err := emit(s, tk, ev); err != nil {
		t.Fatalf("emit ev: %v", err)
	}

	fake := &engine.Fake{
		Output:    "all done",
		SessionID: "sess-next",
		Resumable: true,
	}
	engines := map[string]engine.Engine{"fake": fake}
	f := flow.Flow{
		Name: "single",
		Phases: []flow.Phase{
			{Name: "step1", Engine: "fake"},
		},
	}

	if err := Run(context.Background(), s, tk, f, engines, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("fake.Calls length = %d, want 1", len(fake.Calls))
	}
	if fake.Calls[0].Resume != "sess-existing" {
		t.Errorf("fake.Calls[0].Resume = %q, want sess-existing", fake.Calls[0].Resume)
	}
}
