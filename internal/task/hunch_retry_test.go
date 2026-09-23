package task

// What the decision engine is shown when a run picks up partway through
// its flow.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestARetryFromAPhaseIsShownTheWorkBeforeIt. ORB-121 was moved back to
// review twelve times and jev was asked each time about a run that had said
// nothing: what implement reported was forgotten at every new attempt,
// though no attempt ran implement again. A phase's words stand until that
// phase runs again, whatever the attempts in between wrote about where
// they began.
func TestARetryFromAPhaseIsShownTheWorkBeforeIt(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-80", "make the endpoint idempotent", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	said := func(events ...record.Event) string {
		t.Helper()

		for _, e := range events {
			if err := emit(s, tk, e); err != nil {
				t.Fatalf("emit: %v", err)
			}
		}

		return lastSaid(s, tk)
	}

	// An attempt that says nothing of where it began, as every one before
	// the from field did.
	if got := said(
		record.Event{Kind: record.TaskStarted},
		record.Event{Kind: record.PhaseStarted, Phase: "implement"},
		record.Event{Kind: record.PhaseFinished, Phase: "implement", Text: "make check is green"},
		record.Event{Kind: record.TaskStarted},
	); got != "make check is green" {
		t.Errorf("a new attempt at review was shown %q, want what implement said", got)
	}

	if got := said(record.Event{Kind: record.PhaseStarted, Phase: "implement"}); got != "" {
		t.Errorf("implement running again was shown %q, the words of the run it replaces", got)
	}
}
