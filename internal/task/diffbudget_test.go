package task

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// writingEngine writes a file of n lines into the worktree it is run in,
// which is what an engine does and the fake does not.
type writingEngine struct {
	*engine.Fake

	name  string
	lines int
}

func (e writingEngine) Run(ctx context.Context, req engine.Request) (engine.Result, error) {
	out, err := e.Fake.Run(ctx, req)
	if err != nil {
		return out, err
	}

	body := strings.Repeat("a line of it\n", e.lines)
	if wErr := os.WriteFile(filepath.Join(req.Dir, e.name), []byte(body), 0o600); wErr != nil {
		return out, wErr
	}

	return out, nil
}

func budgetFlow(lines int) flow.Flow {
	return flow.Flow{Name: "task", DiffBudget: lines, Phases: []flow.Phase{
		{Name: "implement", Engine: "fake"},
		{Name: "review", Engine: "fake"},
	}}
}

// TestARunStopsWhenTheChangeIsBiggerThanWasAgreed. The number is what a
// reader agreed to read; a change past it is a decision, and the phase after
// it would only make the decision bigger.
func TestARunStopsWhenTheChangeIsBiggerThanWasAgreed(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-10", "a change that runs away", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := writingEngine{Fake: engine.NewFake("wrote a lot"), name: "big.txt", lines: 40}

	err = Run(context.Background(), s, tk, budgetFlow(10), map[string]engine.Engine{"fake": eng}, nil)
	if err == nil {
		t.Fatal("Run: want an error when the change is over its budget")
	}

	if len(eng.Calls) != 1 {
		t.Errorf("the engine ran %d times, want 1 — the second phase would only add to a change already too big", len(eng.Calls))
	}

	events, evErr := Events(s, tk)
	if evErr != nil {
		t.Fatalf("Events: %v", evErr)
	}

	last := events[len(events)-1]
	if last.Kind != record.TaskOverDiff {
		t.Fatalf("the record ends in %q, want task.over_diff: %v", last.Kind, kindsOf(events))
	}

	if last.Data["lines"] != "40" || last.Data["budget"] != "10" {
		t.Errorf("task.over_diff says %q lines against %q, want 40 against 10", last.Data["lines"], last.Data["budget"])
	}
}

// TestAChangeInsideItsBudgetIsLeftAlone.
func TestAChangeInsideItsBudgetIsLeftAlone(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-11", "a small change", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := writingEngine{Fake: engine.NewFake("wrote a little"), name: "small.txt", lines: 3}

	if err := Run(context.Background(), s, tk, budgetFlow(100), map[string]engine.Engine{"fake": eng}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(eng.Calls) != 2 {
		t.Errorf("the engine ran %d times, want both phases", len(eng.Calls))
	}
}

// TestNoBudgetIsNoGate. Zero is the working zero of every setting in Orbit,
// and a flow that says nothing about size must not stop.
func TestNoBudgetIsNoGate(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-12", "a change nobody capped", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := writingEngine{Fake: engine.NewFake("wrote a lot"), name: "big.txt", lines: 500}
	if err := Run(context.Background(), s, tk, budgetFlow(0), map[string]engine.Engine{"fake": eng}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(eng.Calls) != 2 {
		t.Errorf("the engine ran %d times, want both phases", len(eng.Calls))
	}
}

// TestAFileThePlanNeverNamedStopsTheRun. The scope is the plan of this run,
// read back from the record: a file nobody planned for is the mistake this
// gate exists to catch, whatever its size.
func TestAFileThePlanNeverNamedStopsTheRun(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-13", "a change that wandered", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	planned := flow.Flow{Name: "task", DiffBudget: 1000, Phases: []flow.Phase{
		{Name: "1-plan", Engine: "planner"},
		{Name: "2-implement", Engine: "wanderer"},
		{Name: "3-review", Engine: "planner"},
	}}

	engines := map[string]engine.Engine{
		"planner":  engine.NewFake("I will change plan.txt and nothing else."),
		"wanderer": writingEngine{Fake: engine.NewFake("wrote elsewhere"), name: "elsewhere.txt", lines: 2},
	}

	err = Run(context.Background(), s, tk, planned, engines, nil)
	if err == nil {
		t.Fatal("Run: want an error when a file outside the plan is changed")
	}

	events, evErr := Events(s, tk)
	if evErr != nil {
		t.Fatalf("Events: %v", evErr)
	}

	last := events[len(events)-1]
	if last.Kind != record.TaskOverDiff {
		t.Fatalf("the record ends in %q, want task.over_diff: %v", last.Kind, kindsOf(events))
	}

	if last.Data["unplanned"] != "elsewhere.txt" {
		t.Errorf("task.over_diff names %q as unplanned, want elsewhere.txt", last.Data["unplanned"])
	}
}

// TestAFileThePlanNamedIsInScope, even when the plan says a great deal else
// around it: the reading is plain, and interpreting the paragraph is not
// Orbit's to do.
func TestAFileThePlanNamedIsInScope(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-14", "a change that stayed", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	planned := flow.Flow{Name: "task", DiffBudget: 1000, Phases: []flow.Phase{
		{Name: "1-plan", Engine: "planner"},
		{Name: "2-implement", Engine: "worker"},
	}}

	engines := map[string]engine.Engine{
		"planner": engine.NewFake("The work is in inside.txt, and nowhere else."),
		"worker":  writingEngine{Fake: engine.NewFake("wrote inside"), name: "inside.txt", lines: 2},
	}

	if err := Run(context.Background(), s, tk, planned, engines, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
}
