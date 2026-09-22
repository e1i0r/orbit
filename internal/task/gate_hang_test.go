package task

// A gate is a shell line, and the work is whatever it spawns.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// TestAGateThatLeavesSomethingRunningDoesNotHangTheRun.
//
// CombinedOutput waits for the output pipe to close, and a background child
// holds it open after the shell itself has exited: a gate ending in
// `make serve &` — or any `go test` that leaves a helper behind — held the
// run for as long as that child lived, with the task reading as running and
// its slot taken. internal/repo learned this on its own checks; the gates a
// flow carries went on running the old way.
func TestAGateThatLeavesSomethingRunningDoesNotHangTheRun(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-40", "a gate that spawns something", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	f := flow.Flow{Name: "gated", Phases: []flow.Phase{{
		Name: "implement", Engine: "fake",
		Gates: []flow.Gate{{Name: "leaves one behind", Command: "sleep 30 & echo started"}},
	}}}

	done := make(chan error, 1)

	began := time.Now()

	go func() { done <- Run(context.Background(), s, tk, f, fakes(engine.NewFake("wrote it")), nil) }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}

		if took := time.Since(began); took > 20*time.Second {
			t.Errorf("the run waited %s on a gate whose shell had already exited", took)
		}

		// And the gate is passed rather than refused: exit zero is what
		// it answered, and the record says the reading was cut short by
		// what the gate left running.
		var passed *record.Event

		for _, e := range mustEvents(t, s, tk) {
			if e.Kind == record.GatePassed {
				passed = &e

				break
			}
		}

		if passed == nil {
			t.Fatalf("no gate passed: %v", kindsOf(mustEvents(t, s, tk)))
		}

		if passed.Data["left_running"] != "true" {
			t.Errorf("the gate left a child running and the record does not say so: %v", passed.Data)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the run is still waiting on a gate whose shell exited at once")
	}
}

// TestCancellingARunKillsWhatItsGateStarted. A gate's shell is killed when
// the run is cancelled and its children were not: `sh` died and the
// `go test` under it went on, in a worktree Orbit was finished with.
func TestCancellingARunKillsWhatItsGateStarted(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-41", "a gate whose child outlives it", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	mark := filepath.Join(t.TempDir(), "still-here")

	// The gate starts something slow and then blocks: the run is cancelled
	// while it is in there, and what the child does afterwards is the
	// question.
	f := flow.Flow{Name: "gated", Phases: []flow.Phase{{
		Name: "implement", Engine: "fake",
		Gates: []flow.Gate{{
			Name:    "starts a child",
			Command: "(sleep 2; touch " + mark + ") & sleep 20",
		}},
	}}}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- Run(ctx, s, tk, f, fakes(engine.NewFake("wrote it")), nil) }()

	time.Sleep(500 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("the run did not stop when it was cancelled")
	}

	// Long enough for the child to have touched the file, if it lived.
	time.Sleep(3 * time.Second)

	if _, err := os.Stat(mark); err == nil {
		t.Error("the run was cancelled and what its gate started carried on")
	}
}
