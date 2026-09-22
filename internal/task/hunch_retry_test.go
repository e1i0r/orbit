package task

// What the decision engine is shown when a run picks up partway through
// its flow.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestARetryFromAPhaseIsShownTheWorkBeforeIt. ORB-121 was moved back to
// review ten times and jev was asked each time about a run that had said
// nothing: the attempt began at the gate, and what implement reported was
// forgotten with the attempt before it. A retry from a phase keeps every
// phase before it, so what they said still stands. A run from the top
// replaces them, so it starts with nothing said.
func TestARetryFromAPhaseIsShownTheWorkBeforeIt(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-80", "make the endpoint idempotent", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	for _, e := range []record.Event{
		{Kind: record.TaskStarted},
		{Kind: record.PhaseFinished, Phase: "implement", Text: "make check is green"},
		{Kind: record.TaskStarted, Data: map[string]string{"from": "review"}},
	} {
		if err := emit(s, tk, e); err != nil {
			t.Fatalf("emit: %v", err)
		}
	}

	if got := lastSaid(s, tk); got != "make check is green" {
		t.Errorf("a retry from review was shown %q, want what implement said", got)
	}

	if err := emit(s, tk, record.Event{Kind: record.TaskStarted}); err != nil {
		t.Fatalf("emit: %v", err)
	}

	if got := lastSaid(s, tk); got != "" {
		t.Errorf("a run from the top was shown %q from the attempt it replaces", got)
	}
}
