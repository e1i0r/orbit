package task

// What bounds a loop: the size of the change it has already made, and the
// number of turns it is allowed whoever is running them.

import (
	"context"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// TestTheDiffBudgetIsAskedBetweenTurnsToo.
//
// A loop is where a change grows: three turns at a phase that writes is
// three times whatever one turn writes. The size was measured where two
// phases meet, so a loop of three turns could go four hundred lines past a
// budget of five and only be caught on the way out — after paying for
// every turn that made the decision bigger.
func TestTheDiffBudgetIsAskedBetweenTurnsToo(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-43", "a loop that keeps writing", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	wrote := writingEngine{Fake: engine.NewFake("wrote a fix"), name: "big.txt", lines: 40}

	// Five lines allowed, a check nothing satisfies, and three turns to
	// blow it in: the first turn writes forty lines, and the second one
	// never starts.
	f := flow.Flow{Name: "fed", DiffBudget: 5, Phases: []flow.Phase{{
		Name: "green", Loop: &flow.Loop{
			Phases: []flow.Phase{{Name: "fix", Engine: "fake"}},
			Until:  []flow.Gate{{Name: "unit", Command: "exit 1"}},
			Max:    3,
		},
	}}}

	if err := Run(context.Background(), s, tk, f, fakes(wrote), nil); err == nil {
		t.Fatal("Run: want an error when the change is bigger than was agreed")
	}

	if len(wrote.Calls) != 1 {
		t.Errorf("the loop went round %d times, want 1 — the change was already too big", len(wrote.Calls))
	}

	events := mustEvents(t, s, tk)

	last := events[len(events)-1]
	if last.Kind != record.TaskOverDiff {
		t.Fatalf("the record ends in %q, want task.over_diff: %v", last.Kind, kindsOf(events))
	}
}

// dryAfter works for the first n calls and has nothing left after that,
// which is what running out in the middle of a loop looks like.
type dryAfter struct {
	*engine.Fake

	n    int
	runs int
}

func (e *dryAfter) Run(ctx context.Context, req engine.Request) (engine.Result, error) {
	e.runs++
	if e.runs > e.n {
		return engine.Result{Output: ranDryOut}, errRanOut
	}

	return e.Fake.Run(ctx, req)
}

func (e *dryAfter) RanOut(out engine.Result, err error) bool {
	return engine.NewClaude().RanOut(out, err)
}

// TestALoopKeepsItsCapAcrossARelay.
//
// max is how many turns the flow allows, and it was how many turns each
// engine allowed: an engine that ran out in the middle handed the task on,
// the loop started again from turn one, and a loop written as three turns
// went round three more — paying for every one of them, and for as many
// engines as the machine has.
func TestALoopKeepsItsCapAcrossARelay(t *testing.T) {
	s, tk := nowhere(t, "ACME-44", "a loop that changes hands")
	autopilotOn(t, s, true)

	const max = 3

	// claude does one turn and then has nothing left; codex would do as
	// many as it is given.
	engines := map[string]engine.Engine{
		"claude": under{&dryAfter{Fake: engine.NewFake("wrote a fix"), n: 1}, "claude"},
		"codex":  under{engine.NewFake("wrote a fix"), "codex"},
	}

	f := flow.Flow{Name: "fed", Phases: []flow.Phase{{
		Name: "green", Loop: &flow.Loop{
			Phases: []flow.Phase{{Name: "fix", Engine: "claude"}},
			Until:  []flow.Gate{{Name: "unit", Command: "exit 1"}},
			Max:    max,
		},
	}}}

	if err := Run(context.Background(), s, tk, f, engines, nil, onlyFree("codex", 0)); err == nil {
		t.Fatal("Run: want an error when the loop runs out of turns")
	}

	events := eventsOf(t, s, tk)

	turns := 0

	for _, e := range events {
		if e.Kind == record.LoopChecked {
			turns++
		}
	}

	if turns > max {
		t.Errorf("the loop went round %d times against a cap of %d: %v", turns, max, kindsFor(t, s, tk))
	}

	last := events[len(events)-1]
	if last.Kind != record.TaskStuck {
		t.Errorf("the record ends in %q, want task.stuck: %v", last.Kind, kindsFor(t, s, tk))
	}
}
