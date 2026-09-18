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

	err = Run(context.Background(), s, tk, budgetFlow(10), fakes(eng), nil)
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

	// And it says nothing about a plan, because this flow has none. A
	// sentence that names no files, and a field that holds none, send the
	// reader looking for a list that does not exist.
	if strings.Contains(last.Text, "not in the plan") {
		t.Errorf("a run under a flow with no plan was told %q", last.Text)
	}

	if unplanned, there := last.Data["unplanned"]; there {
		t.Errorf("a run under a flow with no plan carries unplanned=%q", unplanned)
	}
}

// TestAChangeExactlyAsBigAsWasAgreedIsWithinIt.
//
// The budget is a number somebody chose, and the reading of it has to be the
// one they made: a change of exactly that many lines is the change they
// agreed to read. One line too strict and the number that works is one
// larger than the number they typed, with nothing anywhere saying so.
func TestAChangeExactlyAsBigAsWasAgreedIsWithinIt(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-15", "a change right at the line", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := writingEngine{Fake: engine.NewFake("wrote exactly that much"), name: "exact.txt", lines: 10}

	if err := Run(context.Background(), s, tk, budgetFlow(10), fakes(eng), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(eng.Calls) != 2 {
		t.Errorf("a change of exactly the budget stopped the run after %d phases", len(eng.Calls))
	}
}

// TestAChangeOneLineOverIsOver, which is the other side of the same number.
func TestAChangeOneLineOverIsOver(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-16", "a change one line too far", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := writingEngine{Fake: engine.NewFake("wrote one line too many"), name: "over.txt", lines: 11}

	if err := Run(context.Background(), s, tk, budgetFlow(10), fakes(eng), nil); err == nil {
		t.Fatal("Run: want an error when the change is one line over its budget")
	}

	events, evErr := Events(s, tk)
	if evErr != nil {
		t.Fatalf("Events: %v", evErr)
	}

	last := events[len(events)-1]
	if last.Kind != record.TaskOverDiff || last.Data["lines"] != "11" {
		t.Errorf("the record ends in %q saying %q lines", last.Kind, last.Data["lines"])
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

	if err := Run(context.Background(), s, tk, budgetFlow(100), fakes(eng), nil); err != nil {
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
	if err := Run(context.Background(), s, tk, budgetFlow(0), fakes(eng), nil); err != nil {
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

	// And the sentence names it too: the field is for a program and the
	// text is what a person is shown when the run stops.
	if !strings.Contains(last.Text, "Changed and not in the plan: elsewhere.txt") {
		t.Errorf("task.over_diff reads %q", last.Text)
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

// TestABudgetCountsEveryCheckoutTheTaskHasWorkedIn.
//
// All of them and not only the one the run started in: a task that spans
// three repositories is one piece of work, and a budget that counted one of
// them would be three budgets nobody set.
//
// The record is asked first, because it is the only account of where the
// work actually went — and a task that has worked nowhere yet falls back to
// the checkout it was written against, which is where it is about to.
func TestABudgetCountsEveryCheckoutTheTaskHasWorkedIn(t *testing.T) {
	s, repos := workspaceFixture(t, "api", "ledger")

	tk, err := Create(s, repos[0], "ACME-18", "one piece of work", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := Join(s, tk, repos[1]); err != nil {
		t.Fatalf("Join: %v", err)
	}

	both := reposOf(s, tk)
	if len(both) != 2 {
		t.Fatalf("a task worked in two checkouts is measured over %d: %v", len(both), both)
	}

	seen := map[string]bool{}
	for _, p := range both {
		seen[p] = true
	}

	for _, r := range repos {
		if !seen[r.Path] {
			t.Errorf("%q is not among the checkouts the budget covers: %v", r.Path, both)
		}
	}

	// A task the record knows nothing about is the one it was written
	// against, which is the state every task passes through before its
	// first phase joins anything.
	fresh := reposOf(s, Task{ID: "NOT-WRITTEN-1", Repo: repos[0]})
	if len(fresh) != 1 || fresh[0] != repos[0].Path {
		t.Errorf("a task that has worked nowhere is measured over %v", fresh)
	}

	// And one written against nothing has nothing to measure, rather than a
	// path made of the empty string.
	if none := reposOf(s, Task{ID: "NOT-WRITTEN-2"}); len(none) != 0 {
		t.Errorf("a task with no checkout is measured over %v", none)
	}
}
