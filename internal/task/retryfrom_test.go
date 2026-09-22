package task

// A run that begins at one phase leaves the phases before it alone.

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// TestTheSpawnedRunCarriesTheStartingPhase: -from is passed only when a
// phase was named, the way -engine is.
func TestTheSpawnedRunCarriesTheStartingPhase(t *testing.T) {
	tk := Task{ID: "RET-1"}

	with := strings.Join(runCommand("orbit", "/root", tk, "careful", "claude", "review").Args, " ")
	if !strings.Contains(with, "-from review") {
		t.Errorf("a retry spawned %q, want -from review in it", with)
	}

	without := strings.Join(runCommand("orbit", "/root", tk, "careful", "claude", "").Args, " ")
	if strings.Contains(without, "-from") {
		t.Errorf("an ordinary run spawned %q, want no -from at all", without)
	}
}

// TestARetryRunsFromTheNamedPhaseOnward is the rule itself: what passed is
// not paid for twice, and what comes after the retried phase still runs.
func TestARetryRunsFromTheNamedPhaseOnward(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "RET-2", "three phases", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("done")

	three := flow.Flow{
		Name: "three",
		Phases: []flow.Phase{
			{Name: "implement", Engine: "fake", Model: "sonnet"},
			{Name: "review", Engine: "fake", Model: "opus"},
			{Name: "fix", Engine: "fake", Model: "sonnet"},
		},
	}

	if err := RunFrom(context.Background(), s, tk, three, fakes(fake), nil, "review"); err != nil {
		t.Fatalf("RunFrom: %v", err)
	}

	// Two engine calls, not three: implement was left alone.
	if len(fake.Calls) != 2 {
		t.Fatalf("a retry from review ran %d phases, want review and fix", len(fake.Calls))
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	var ran []string

	for _, e := range events {
		if e.Kind == record.PhaseStarted {
			ran = append(ran, e.Phase+"/"+e.Data["n"])
		}
	}

	// And the numbering is the flow's, not the retry's: review is still
	// the second phase of three, so the tree reads [2/3] and not [1/2].
	want := []string{"review/2", "fix/3"}
	if strings.Join(ran, " ") != strings.Join(want, " ") {
		t.Errorf("a retry from review ran %v, want %v", ran, want)
	}
}

// TestANamedPhaseThatIsNotInTheFlowRunsNothing: a retry cannot invent a
// phase, and silently running the whole flow would be the worst answer.
func TestANamedPhaseThatIsNotInTheFlowRunsNothing(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "RET-3", "two phases", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("done")

	two := flow.Flow{Name: "two", Phases: []flow.Phase{
		{Name: "implement", Engine: "fake", Model: "sonnet"},
		{Name: "review", Engine: "fake", Model: "opus"},
	}}

	if err := RunFrom(context.Background(), s, tk, two, fakes(fake), nil, "nonsense"); err != nil {
		t.Fatalf("RunFrom: %v", err)
	}

	if len(fake.Calls) != 0 {
		t.Errorf("a retry of a phase that is not in the flow ran %d phases, want none",
			len(fake.Calls))
	}
}
