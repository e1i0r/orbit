package task

// However a run ends, the record says it ended.
//
// A task's log ends in a terminal event or a reader appends one, and until
// one is there the window draws the task as running: a row in the running
// band, a slot held against the limit, and a reader waiting on a process
// that is not there. Every way a run can stop is walked here and held to
// that one rule, because the ways multiply — a budget, a diff, a
// dependency, a contradiction, a gate, an engine with nothing left — and
// each of them was written on its own day.

import (
	"context"
	"errors"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// overWith are the kinds internal/view reads as a run that is over: every
// state it maps to the done band or the needs-you band. A kind missing
// here is a task the window would draw as running for ever, which is why
// the list is written out rather than derived.
var overWith = map[string]bool{
	record.TaskFinished:      true,
	record.TaskFailed:        true,
	record.TaskCancelled:     true,
	record.TaskTimedOut:      true,
	record.TaskStuck:         true,
	record.TaskOverBudget:    true,
	record.TaskOverDiff:      true,
	record.TaskNeedsEngine:   true,
	record.TaskNoEngine:      true,
	record.TaskAbandoned:     true,
	record.PhaseWaiting:      true,
	record.PhaseFailed:       true,
	record.PhaseDenied:       true,
	record.PhaseRanOut:       true,
	record.TaskRequeued:      true,
	record.TaskNewDependency: true,
	record.TaskContradicts:   true,
}

// TestHoweverARunEndsTheRecordSaysItEnded.
func TestHoweverARunEndsTheRecordSaysItEnded(t *testing.T) {
	for _, c := range []struct {
		what string
		// set up the world, and answer with the flow to walk and the
		// engines to walk it with.
		open func(*testing.T, *store.Store, Task) (flow.Flow, map[string]engine.Engine)
		// gate is what is asked before every phase, and nil for the runs
		// nothing holds.
		gate Gate
		// stopped is a run the reader cancelled before it began, which is
		// the shape of pressing cancel on a task whose phase has not
		// started yet.
		stopped bool
	}{
		{"every phase ran", func(_ *testing.T, _ *store.Store, _ Task) (flow.Flow, map[string]engine.Engine) {
			return twoPhases(), fakes(engine.NewFake("done"))
		}, nil, false},
		{"the engine broke", func(_ *testing.T, _ *store.Store, _ Task) (flow.Flow, map[string]engine.Engine) {
			broken := engine.NewFake("")
			broken.Err = errors.New("the model refused to start")

			return twoPhases(), fakes(broken)
		}, nil, false},
		{"the flow names an engine nobody has", func(_ *testing.T, _ *store.Store, _ Task) (flow.Flow, map[string]engine.Engine) {
			f := flow.Flow{Name: "odd", Phases: []flow.Phase{{Name: "implement", Engine: "nobody"}}}

			return f, fakes(engine.NewFake("done"))
		}, nil, false},
		{"a gate refused every attempt", func(_ *testing.T, _ *store.Store, _ Task) (flow.Flow, map[string]engine.Engine) {
			f := flow.Flow{Name: "gated", Attempts: 2, Phases: []flow.Phase{{
				Name: "implement", Engine: "fake",
				Gates: []flow.Gate{{Name: "never", Command: "exit 1"}},
			}}}

			return f, fakes(engine.NewFake("done"))
		}, nil, false},
		{"the loop never went green", func(_ *testing.T, _ *store.Store, _ Task) (flow.Flow, map[string]engine.Engine) {
			return tddFlow("exit 1", 2), fakes(engine.NewFake("tried"))
		}, nil, false},
		{"the task spent its budget", func(t *testing.T, s *store.Store, _ Task) (flow.Flow, map[string]engine.Engine) {
			t.Helper()

			if err := s.SaveSettings(store.Settings{BudgetTask: 0.10}); err != nil {
				t.Fatalf("SaveSettings: %v", err)
			}

			return twoPhases(), fakes(costlyEngine{engine.NewFake("done")})
		}, nil, false},
		{"the engine ran out and there is no other", func(_ *testing.T, _ *store.Store, _ Task) (flow.Flow, map[string]engine.Engine) {
			dry := engine.NewFake("You've reached your usage limit. Your limit resets at 4pm.")

			return twoPhases(), fakes(dry)
		}, nil, false},
		{"the change was bigger than was agreed", func(_ *testing.T, _ *store.Store, _ Task) (flow.Flow, map[string]engine.Engine) {
			wrote := writingEngine{Fake: engine.NewFake("done"), name: "big.txt", lines: 40}

			return budgetFlow(5), fakes(wrote)
		}, nil, false},
		{"a gate stopped it", func(_ *testing.T, _ *store.Store, _ Task) (flow.Flow, map[string]engine.Engine) {
			return twoPhases(), fakes(engine.NewFake("done"))
		}, &scriptedGate{answers: []Go{Stop}}, false},
		{"every phase was skipped", func(_ *testing.T, _ *store.Store, _ Task) (flow.Flow, map[string]engine.Engine) {
			return twoPhases(), fakes(engine.NewFake("done"))
		}, &scriptedGate{answers: []Go{Skip, Skip}}, false},
		{"the reader cancelled it", func(_ *testing.T, _ *store.Store, _ Task) (flow.Flow, map[string]engine.Engine) {
			return twoPhases(), fakes(engine.NewFake("done"))
		}, nil, true},
	} {
		t.Run(c.what, func(t *testing.T) {
			s, r := fixture(t)

			tk, err := Create(s, r, "ACME-50", "however this ends", "")
			if err != nil {
				t.Fatalf("Create: %v", err)
			}

			f, engines := c.open(t, s, tk)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if c.stopped {
				cancel()
			}

			// Whether Run answers with an error is not the question: a
			// run that failed is a run that ended, and both have to be
			// written down.
			runErr := Run(ctx, s, tk, f, engines, c.gate)

			events := mustEvents(t, s, tk)
			if len(events) == 0 {
				t.Fatalf("the record is empty after %v", runErr)
			}

			last := events[len(events)-1]
			if !overWith[last.Kind] {
				t.Errorf("the record ends in %q, which the window reads as a run still going: %v",
					last.Kind, kindsOf(events))
			}
		})
	}
}
