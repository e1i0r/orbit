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

	if got := count(kinds, record.TaskFailed); got != 1 {
		t.Errorf("task.failed = %d, want 1: %v", got, kinds)
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
