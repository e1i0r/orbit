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

// kindsFor is every kind the task's record holds, in order.
func kindsFor(t *testing.T, s *store.Store, tk Task) []string {
	t.Helper()

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	return kindsOf(events)
}

// countingGate fails until it has been run n times, and says so on stdout
// the way a build does.
func countingGate(n string) string {
	return "c=$(cat .attempts 2>/dev/null || echo 0); c=$((c+1)); printf %s \"$c\" > .attempts; " +
		"if [ \"$c\" -ge " + n + " ]; then echo build ok; else echo 'undefined: Retry'; exit 1; fi"
}

func retryFlow(gate string, attempts int) flow.Flow {
	return flow.Flow{Name: "task", Attempts: attempts, Phases: []flow.Phase{
		{Name: "implement", Engine: "fake", Gates: []flow.Gate{{Name: "build", Command: gate}}},
	}}
}

func count(kinds []string, kind string) int {
	n := 0

	for _, k := range kinds {
		if k == kind {
			n++
		}
	}

	return n
}

func TestPhaseIsRunAgainWhenItsGateFails(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "make the build green", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("wrote it")
	engines := map[string]engine.Engine{"fake": fake}

	if err := Run(context.Background(), s, tk, retryFlow(countingGate("3"), 0), engines, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 3 {
		t.Fatalf("the engine ran %d times, want 3 — the phase is run again for each gate that refused it", len(fake.Calls))
	}

	kinds := kindsFor(t, s, tk)
	if got := count(kinds, record.PhaseRetried); got != 2 {
		t.Errorf("phase.retried = %d, want 2: %v", got, kinds)
	}

	if got := count(kinds, record.TaskFinished); got != 1 {
		t.Errorf("task.finished = %d, want 1: %v", got, kinds)
	}
}

func TestAttemptsRunOutAndTheRunFails(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-2", "a gate nothing will satisfy", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("tried")
	engines := map[string]engine.Engine{"fake": fake}

	err = Run(context.Background(), s, tk, retryFlow("echo nope; exit 1", 0), engines, nil)
	if err == nil {
		t.Fatal("Run: want an error when the attempts run out")
	}

	if len(fake.Calls) != flow.DefaultAttempts {
		t.Fatalf("the engine ran %d times, want %d — the default cap", len(fake.Calls), flow.DefaultAttempts)
	}

	kinds := kindsFor(t, s, tk)
	if got := count(kinds, record.GateFailed); got != flow.DefaultAttempts {
		t.Errorf("gate.failed = %d, want %d: %v", got, flow.DefaultAttempts, kinds)
	}

	if got := count(kinds, record.PhaseFailed); got != 1 {
		t.Errorf("phase.failed = %d, want 1 — only the last attempt ends the phase: %v", got, kinds)
	}

	// task.stuck and not task.failed: the run did not break, it ran out of
	// the attempts it was allowed. TestAttemptsRunOutAndTheTaskIsStuck is
	// about what that event carries.
	if got := count(kinds, record.TaskStuck); got != 1 {
		t.Errorf("task.stuck = %d, want 1: %v", got, kinds)
	}
}

func TestAFlowMaySayHowManyAttempts(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-3", "one shot only", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("tried")
	engines := map[string]engine.Engine{"fake": fake}

	if err := Run(context.Background(), s, tk, retryFlow("exit 1", 1), engines, nil); err == nil {
		t.Fatal("Run: want an error when the one attempt the flow allows fails")
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("the engine ran %d times, want 1 — the flow allows one attempt", len(fake.Calls))
	}
}

func TestTheNextAttemptIsToldWhatFailedAndWhatItAnswered(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-4", "fix the build", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("I renamed the symbol")
	engines := map[string]engine.Engine{"fake": fake}

	if err := Run(context.Background(), s, tk, retryFlow(countingGate("3"), 0), engines, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) < 3 {
		t.Fatalf("the engine ran %d times, want 3", len(fake.Calls))
	}

	second := fake.Calls[1].Prompt
	for _, want := range []string{"Attempt 1", "build", "undefined: Retry", "I renamed the symbol"} {
		if !strings.Contains(second, want) {
			t.Errorf("the second attempt's prompt does not carry %q:\n%s", want, second)
		}
	}

	third := fake.Calls[2].Prompt
	if !strings.Contains(third, "Attempt 2") || !strings.Contains(third, "Attempt 1") {
		t.Errorf("the third attempt's prompt does not carry both attempts before it:\n%s", third)
	}
}

func TestAPhaseWithNoGateIsRunOnce(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-5", "nothing to verify", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("done")
	if err := Run(context.Background(), s, tk, oneFlow(), map[string]engine.Engine{"fake": fake}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("the engine ran %d times, want 1", len(fake.Calls))
	}
}

// costlyEngine answers like the fake and puts a price on it, which the fake
// itself does not: what a refused attempt spent is the point of the test
// below, and an engine that spends nothing cannot make it.
type costlyEngine struct{ *engine.Fake }

func (e costlyEngine) Run(ctx context.Context, req engine.Request) (engine.Result, error) {
	out, err := e.Fake.Run(ctx, req)
	out.Cost = 0.25
	out.Usage = engine.Usage{Input: 10, Output: 5}

	return out, err
}

// TestARefusedAttemptSaysWhatItSpent. A phase that ran for twenty minutes
// and was then refused by its gate was paid for. The event that ends the
// attempt is the only place that spend can be written down — nothing else
// ends it — so a total taken from the record without it is a number a reader
// acts on and is wrong about.
func TestARefusedAttemptSaysWhatItSpent(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-6", "the gate refuses once", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := costlyEngine{engine.NewFake("tried")}

	if err := Run(context.Background(), s, tk, retryFlow(countingGate("2"), 0), map[string]engine.Engine{"fake": eng}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	for _, e := range events {
		if e.Kind != record.PhaseRetried {
			continue
		}

		if e.Data["cost"] != "0.25" {
			t.Errorf("phase.retried says the attempt cost %q, want 0.25", e.Data["cost"])
		}

		if e.Data["tokens_in"] != "10" || e.Data["tokens_out"] != "5" {
			t.Errorf("phase.retried counts %q in and %q out, want 10 and 5", e.Data["tokens_in"], e.Data["tokens_out"])
		}

		if e.Data["gate"] != "build" || e.Data["attempt"] != "1" {
			t.Errorf("phase.retried names gate %q on attempt %q, want build on 1", e.Data["gate"], e.Data["attempt"])
		}

		return
	}

	t.Error("no phase.retried was written")
}

// TestAttemptsRunOutAndTheTaskIsStuck. A task that spent every attempt it
// was allowed is not one failed run: it is the end of retrying, and retrying
// is what already happened. The record has to say that in its own word, with
// the attempts summarised on the event, so that whoever picks it up reads
// what was tried rather than reconstructing it from the log.
func TestAttemptsRunOutAndTheTaskIsStuck(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-7", "a gate nothing will satisfy", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fake := engine.NewFake("tried")

	if err := Run(context.Background(), s, tk, retryFlow("echo 'undefined: Retry'; exit 2", 0), map[string]engine.Engine{"fake": fake}, nil); err == nil {
		t.Fatal("Run: want an error when the attempts run out")
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	kinds := kindsOf(events)
	if count(kinds, record.TaskFailed) != 0 {
		t.Errorf("the record says the task failed; a task out of attempts is stuck: %v", kinds)
	}

	last := events[len(events)-1]
	if last.Kind != record.TaskStuck {
		t.Fatalf("the record ends in %q, want task.stuck: %v", last.Kind, kinds)
	}

	if last.Data["attempts"] != "3" {
		t.Errorf("task.stuck says %q attempts were spent, want 3", last.Data["attempts"])
	}

	for _, want := range []string{"implement", "build", "3"} {
		if !strings.Contains(last.Text, want) {
			t.Errorf("task.stuck does not say %q in the line a human reads:\n%s", want, last.Text)
		}
	}

	if !strings.Contains(last.Text, "undefined: Retry") {
		t.Errorf("task.stuck carries no summary of what the gate said:\n%s", last.Text)
	}
}

// TestAStuckTaskIsNotARunToReconcile. Reconcile appends task.abandoned to a
// log that never got a terminal event. A stuck task got one — turning it
// into an abandoned one would lose the only word that says retrying is over.
func TestAStuckTaskIsNotARunToReconcile(t *testing.T) {
	if inFlight([]record.Event{
		{Kind: record.TaskStarted},
		{Kind: record.TaskStuck},
	}) {
		t.Error("a log ending in task.stuck reads as a run still in flight")
	}
}
