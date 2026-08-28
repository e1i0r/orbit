package task

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// written creates a task, or fails the test trying.
func written(t *testing.T, s *store.Store, r repo.Repo) Task {
	t.Helper()

	tk, err := Create(s, r, "ACME-1", "retry the webhook on 5xx", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	return tk
}

// twoFlow is two phases, so a test can say what happened to the phase after
// the one the gate answered about.
func twoFlow() flow.Flow {
	return flow.Flow{Name: "task", Phases: []flow.Phase{
		{Name: "implement", Engine: "fake"},
		{Name: "second", Engine: "fake"},
	}}
}

// gatedFlow is a flow whose one phase asks to wait, which is the default the
// autopilot switch overrides.
func gatedFlow() flow.Flow {
	return flow.Flow{Name: "careful", Phases: []flow.Phase{
		{Name: "review", Engine: "fake", Wait: true},
	}}
}

// scriptedGate answers with a sequence written in advance and remembers what
// it was asked about. It is the double the three decisions are tested
// through, so that what Run does with an answer is asserted without any
// file, any clock and any poll.
type scriptedGate struct {
	answers []Go
	asked   []string
	n       int
}

func (g *scriptedGate) Before(_ context.Context, _ Task, p flow.Phase, _ int) (Go, error) {
	g.asked = append(g.asked, p.Name)
	if g.n >= len(g.answers) {
		return Continue, nil
	}

	answer := g.answers[g.n]
	g.n++

	return answer, nil
}

func TestAGateIsAskedAboutEveryPhaseAndContinueRunsIt(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)
	fake := engine.NewFake("wrote the retry")
	g := &scriptedGate{answers: []Go{Continue, Continue}}

	if err := Run(context.Background(), s, tk, twoFlow(), map[string]engine.Engine{"fake": fake}, g); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Every phase, not only the ones whose Wait says so: a live switch and a
	// word written mid-run can only be noticed by asking each time.
	if len(g.asked) != 2 || g.asked[0] != "implement" || g.asked[1] != "second" {
		t.Errorf("the gate was asked about %v, want [implement second]", g.asked)
	}

	if len(fake.Calls) != 2 {
		t.Errorf("the engine was called %d times, want 2", len(fake.Calls))
	}

	wantKinds(t, eventsOf(t, s, tk),
		record.TaskCreated, record.TaskStarted,
		record.PhaseStarted, record.PhaseFinished,
		record.PhaseStarted, record.PhaseFinished,
		record.TaskFinished)
}

func TestAGateThatSaysStopEndsTheRunAndNothingRunsAfterIt(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)
	fake := engine.NewFake("")
	g := &scriptedGate{answers: []Go{Stop}}

	err := Run(context.Background(), s, tk, twoFlow(), map[string]engine.Engine{"fake": fake}, g)
	if err == nil {
		t.Fatal("Run reported success after its gate stopped it — a run that was stopped did not finish")
	}

	if len(fake.Calls) != 0 {
		t.Errorf("the engine was called %d times after the gate said stop, want 0", len(fake.Calls))
	}

	if len(g.asked) != 1 {
		t.Errorf("the gate was asked about %v — the flow carried on past a stop", g.asked)
	}

	events := eventsOf(t, s, tk)
	wantKinds(t, events, record.TaskCreated, record.TaskStarted, record.TaskCancelled)

	for _, e := range events {
		if e.Kind == record.TaskFailed || e.Kind == record.PhaseFailed {
			t.Errorf("a run stopped at its gate was written down as %q: being stopped and breaking are different facts", e.Kind)
		}
	}
}

func TestAGateThatSaysSkipRecordsNothingForThatPhaseAndMovesOn(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)
	fake := engine.NewFake("did the second one")
	g := &scriptedGate{answers: []Go{Skip, Continue}}

	if err := Run(context.Background(), s, tk, twoFlow(), map[string]engine.Engine{"fake": fake}, g); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("the engine was called %d times, want 1 — the skipped phase ran", len(fake.Calls))
	}

	events := eventsOf(t, s, tk)
	wantKinds(t, events,
		record.TaskCreated, record.TaskStarted,
		record.PhaseStarted, record.PhaseFinished,
		record.TaskFinished)

	for _, e := range events {
		if e.Phase == "implement" {
			t.Errorf("the skipped phase left %q in the record; a phase that did not run has nothing to say", e.Kind)
		}
	}

	if got := find(t, events, record.PhaseStarted).Phase; got != "second" {
		t.Errorf("the phase that ran was %q, want second", got)
	}
}

func TestANilGateNeverStops(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)
	fake := engine.NewFake("ok")

	// A run with no gate is a run nothing can hold: `orbit run` before this
	// task existed, and every test that is not about the gate.
	if err := Run(context.Background(), s, tk, gatedFlow(), map[string]engine.Engine{"fake": fake}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Errorf("the engine was called %d times, want 1 — a nil gate held a phase that asks to wait", len(fake.Calls))
	}
}

func TestAPhaseThatAsksToWaitStopsAndSaysTheFlowAskedForIt(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)
	fake := engine.NewFake("reviewed")

	synctest.Test(t, func(t *testing.T) {
		done := make(chan error, 1)
		go func() {
			done <- Run(context.Background(), s, tk, gatedFlow(), map[string]engine.Engine{"fake": fake}, FileGate(s, time.Second))
		}()
		// Wait returns once the run is parked on its poll, so what follows
		// is asserted against a run that has stopped and not against a race.
		synctest.Wait()

		waiting := find(t, eventsOf(t, s, tk), record.PhaseWaiting)
		if waiting.Phase != "review" {
			t.Errorf("phase.waiting names %q, want review", waiting.Phase)
		}

		if got := waiting.Data["why"]; got != "flow" {
			t.Errorf("phase.waiting says why=%q, want flow — a phase the flow held is one that needs you", got)
		}

		for _, e := range eventsOf(t, s, tk) {
			if e.Kind == record.PhaseStarted {
				t.Fatal("the phase ran while its gate was still waiting")
			}
		}

		if err := Control(s, tk, "continue"); err != nil {
			t.Fatalf("Control: %v", err)
		}

		if err := <-done; err != nil {
			t.Fatalf("Run: %v", err)
		}
	})

	events := eventsOf(t, s, tk)
	wantKinds(t, events,
		record.TaskCreated, record.TaskStarted,
		record.PhaseWaiting, record.PhaseResumed,
		record.PhaseStarted, record.PhaseFinished,
		record.TaskFinished)

	if got := find(t, events, record.PhaseResumed).Data["how"]; got != "continue" {
		t.Errorf("phase.resumed says how=%q, want continue — the record has to answer what the reader chose", got)
	}
}

func TestAutopilotLetsAPhaseThatAsksToWaitStraightThrough(t *testing.T) {
	s, r := fixture(t)

	tk := written(t, s, r)
	if err := s.SaveSettings(store.Settings{Autopilot: true, UnreadCap: 5}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	fake := engine.NewFake("reviewed")

	if err := Run(context.Background(), s, tk, gatedFlow(), map[string]engine.Engine{"fake": fake}, FileGate(s, time.Second)); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events := eventsOf(t, s, tk)
	for _, e := range events {
		if e.Kind == record.PhaseWaiting {
			t.Fatal("a phase stopped with autopilot on — the switch is what decides, and Wait is only its default")
		}
	}

	wantKinds(t, events,
		record.TaskCreated, record.TaskStarted,
		record.PhaseStarted, record.PhaseFinished,
		record.TaskFinished)
}

func TestFlippingAutopilotOnReleasesAPhaseThatIsAlreadyWaiting(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)
	fake := engine.NewFake("reviewed")

	synctest.Test(t, func(t *testing.T) {
		done := make(chan error, 1)
		go func() {
			done <- Run(context.Background(), s, tk, gatedFlow(), map[string]engine.Engine{"fake": fake}, FileGate(s, time.Second))
		}()

		synctest.Wait()

		// The switch is live in both directions or the header and the row
		// contradict each other: `autopilot ●` above a task sitting in
		// needs-you because its flow asked it to wait.
		if err := s.SaveSettings(store.Settings{Autopilot: true, UnreadCap: 5}); err != nil {
			t.Fatalf("SaveSettings: %v", err)
		}

		if err := <-done; err != nil {
			t.Fatalf("Run: %v", err)
		}
	})

	if got := find(t, eventsOf(t, s, tk), record.PhaseResumed).Data["how"]; got != "autopilot" {
		t.Errorf("phase.resumed says how=%q, want autopilot", got)
	}
}
