package task

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

func twoPhases() flow.Flow {
	return flow.Flow{Name: "task", Phases: []flow.Phase{
		{Name: "implement", Engine: "fake"},
		{Name: "review", Engine: "fake"},
	}}
}

// TestARunStopsWhenTheTaskHasSpentItsBudget. A budget nothing enforces is a
// number in a settings file. It is checked between phases, which is the one
// place a run can stop without throwing away work that was already paid for.
func TestARunStopsWhenTheTaskHasSpentItsBudget(t *testing.T) {
	s, r := fixture(t)

	if err := s.SaveSettings(store.Settings{BudgetTask: 0.20}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	tk, err := Create(s, r, "ACME-8", "an expensive task", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := costlyEngine{engine.NewFake("done")}

	err = Run(context.Background(), s, tk, twoPhases(), fakes(eng), nil)
	if err == nil {
		t.Fatal("Run: want an error when the task has spent its budget")
	}

	if len(eng.Calls) != 1 {
		t.Errorf("the engine ran %d times, want 1 — the second phase costs money the task does not have", len(eng.Calls))
	}

	events, evErr := Events(s, tk)
	if evErr != nil {
		t.Fatalf("Events: %v", evErr)
	}

	last := events[len(events)-1]
	if last.Kind != record.TaskOverBudget {
		t.Fatalf("the record ends in %q, want task.over_budget: %v", last.Kind, kindsOf(events))
	}

	if last.Data["spent"] != "0.25" || last.Data["budget"] != "0.2" {
		t.Errorf("task.over_budget says %q spent against %q, want 0.25 against 0.2", last.Data["spent"], last.Data["budget"])
	}

	if !strings.Contains(last.Text, "review") {
		t.Errorf("task.over_budget does not name the phase that was not run:\n%s", last.Text)
	}
}

// TestABudgetOfZeroIsNoBudget. Every field of the settings file has a
// working zero, and for a cap the working zero is "there is none": a fresh
// install must not stop a run before its first phase.
func TestABudgetOfZeroIsNoBudget(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-9", "a task with no cap", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := costlyEngine{engine.NewFake("done")}
	if err := Run(context.Background(), s, tk, twoPhases(), fakes(eng), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(eng.Calls) != 2 {
		t.Errorf("the engine ran %d times, want both phases", len(eng.Calls))
	}
}

// TestWhatATaskHasSpentIsEveryPhaseItPaidFor, refused attempts included:
// the money is gone whether or not the gate liked the work.
func TestWhatATaskHasSpentIsEveryPhaseItPaidFor(t *testing.T) {
	got := spentOn([]record.Event{
		{Kind: record.PhaseFinished, Data: map[string]string{"cost": "0.25"}},
		{Kind: record.PhaseRetried, Data: map[string]string{"cost": "0.25"}},
		{Kind: record.PhaseFailed, Data: map[string]string{"cost": "0.25"}},
		{Kind: record.PhaseCancelled, Data: map[string]string{"cost": "0.25"}},
		{Kind: record.PhaseStarted, Data: map[string]string{"cost": "99"}},
		{Kind: record.PhaseThought, Text: "free"},
	})

	if got != 1.0 {
		t.Errorf("spentOn = %v, want 1.0 — the four phase endings and nothing else", got)
	}
}

// TestATaskThatHasSpentExactlyItsBudgetHasSpentIt.
//
// The number is what a task may spend, so at exactly that number it has
// spent it and the phase after it is money past what anybody agreed to. A
// reading one cent looser is a cap that never quite holds, and a board that
// shows the budget beside the spend would say they were equal while the run
// carried on.
func TestATaskThatHasSpentExactlyItsBudgetHasSpentIt(t *testing.T) {
	s, r := fixture(t)

	// Exactly what one phase of costlyEngine costs.
	if err := s.SaveSettings(store.Settings{BudgetTask: 0.25}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	tk, err := Create(s, r, "ACME-17", "a task that spends exactly its budget", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := costlyEngine{engine.NewFake("done")}

	if runErr := Run(context.Background(), s, tk, twoPhases(), fakes(eng), nil); runErr == nil {
		t.Fatal("Run: want an error when the task has spent exactly its budget")
	}

	if len(eng.Calls) != 1 {
		t.Errorf("the engine ran %d times, want 1 — a task that has spent its budget has nothing left", len(eng.Calls))
	}

	spent, budget, over := overBudget(s, tk)
	if !over || spent != budget {
		t.Errorf("a task that spent %v of a budget of %v reads as over: %v", spent, budget, over)
	}
}

// TestTheBudgetIsAskedBetweenAttemptsToo.
//
// A phase whose gate refuses it is run again, and every attempt is charged
// for: three attempts at a phase that costs a quarter is seventy-five
// cents against a budget that stopped at twenty. The cap was asked where
// two phases meet and nowhere else, so a flow of one phase with attempts to
// spare walked straight past it — which is the same hole the loop's own
// check was written to close, in the other place a run goes round.
//
// Between attempts is as safe a place to stop as between phases: nothing is
// running, the gate has already said no, and what would happen next is a
// fresh call somebody has to pay for.
func TestTheBudgetIsAskedBetweenAttemptsToo(t *testing.T) {
	s, r := fixture(t)

	if err := s.SaveSettings(store.Settings{BudgetTask: 0.30}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	tk, err := Create(s, r, "ACME-42", "a phase that keeps being refused", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := costlyEngine{engine.NewFake("done")}

	// One phase, a gate nothing satisfies, and four attempts at it: the
	// third would take the task past the budget, so it never happens.
	f := flow.Flow{Name: "stubborn", Attempts: 4, Phases: []flow.Phase{{
		Name: "implement", Engine: "fake",
		Gates: []flow.Gate{{Name: "never", Command: "exit 1"}},
	}}}

	if err := Run(context.Background(), s, tk, f, fakes(eng), nil); err == nil {
		t.Fatal("Run: want an error when the task has spent its budget")
	}

	if len(eng.Calls) != 2 {
		t.Errorf("the engine ran %d times, want 2 — the third attempt costs money the task does not have",
			len(eng.Calls))
	}

	events := mustEvents(t, s, tk)

	last := events[len(events)-1]
	if last.Kind != record.TaskOverBudget {
		t.Fatalf("the record ends in %q, want task.over_budget: %v", last.Kind, kindsOf(events))
	}
}
