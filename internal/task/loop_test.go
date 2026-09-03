package task

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// tddFlow is one phase and then a loop that goes round until the check
// passes, which is the shape the issue draws.
func tddFlow(check string, max int) flow.Flow {
	return flow.Flow{Name: "tdd", Phases: []flow.Phase{
		{Name: "implement", Engine: "fake"},
		{Name: "green", Loop: &flow.Loop{
			Phases: []flow.Phase{{Name: "fix", Engine: "fake"}},
			Until:  []flow.Gate{{Name: "unit", Command: check}},
			Max:    max,
		}},
	}}
}

// TestALoopGoesRoundUntilTheCheckPasses. The one thing that says the work is
// done is an exit code: a model asked whether its own work passes is being
// asked to mark its own paper.
func TestALoopGoesRoundUntilTheCheckPasses(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-29", "make the tests green", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("wrote a fix")

	if err := Run(context.Background(), s, tk, tddFlow(countingGate("3"), 5), map[string]engine.Engine{"fake": fake}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// One implement, then the loop's phase once per turn: the check fails
	// twice and passes on the third.
	if len(fake.Calls) != 4 {
		t.Errorf("the engine ran %d times, want 4 — implement, then three turns of the loop", len(fake.Calls))
	}

	kinds := kindsOf(mustEvents(t, s, tk))
	if got := count(kinds, record.LoopChecked); got != 3 {
		t.Errorf("the record holds %d checks, want one per turn: %v", got, kinds)
	}
}

// TestALoopStopsAtItsCap, and the task is stuck rather than failed: the
// history of the turns is what the reader picks it up with.
func TestALoopStopsAtItsCap(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-30", "a check nothing will satisfy", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("tried")

	err = Run(context.Background(), s, tk, tddFlow("echo 'FAIL: TestIdempotent'; exit 1", 2), map[string]engine.Engine{"fake": fake}, nil)
	if err == nil {
		t.Fatal("Run: want an error when the loop runs out of turns")
	}

	if len(fake.Calls) != 3 {
		t.Errorf("the engine ran %d times, want 3 — implement and the two turns the loop allows", len(fake.Calls))
	}

	events := mustEvents(t, s, tk)

	last := events[len(events)-1]
	if last.Kind != record.TaskStuck {
		t.Fatalf("the record ends in %q, want task.stuck: %v", last.Kind, kindsOf(events))
	}

	if !strings.Contains(last.Text, "FAIL: TestIdempotent") {
		t.Errorf("the task is stuck without the history of what failed:\n%s", last.Text)
	}
}

// TestEachTurnIsToldWhatTheCheckSaid. Retrying without the error is
// repeating blind, which is 3.1's rule applied to the outer loop.
func TestEachTurnIsToldWhatTheCheckSaid(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-31", "learn from the failure", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("tried")
	if err := Run(context.Background(), s, tk, tddFlow(countingGate("2"), 3), map[string]engine.Engine{"fake": fake}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) < 3 {
		t.Fatalf("the engine ran %d times, want implement and two turns", len(fake.Calls))
	}

	second := fake.Calls[2].Prompt
	if !strings.Contains(second, "undefined: Retry") || !strings.Contains(second, "unit") {
		t.Errorf("the second turn was not told what the check said:\n%s", second)
	}
}

// TestAPhaseWithNoLoopIsUntouched.
func TestAPhaseWithNoLoopIsUntouched(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-32", "a straight line", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("done")
	if err := Run(context.Background(), s, tk, twoPhases(), map[string]engine.Engine{"fake": fake}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 2 {
		t.Errorf("the engine ran %d times, want the two phases", len(fake.Calls))
	}
}
