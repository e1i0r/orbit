// The gate parks a run in its own goroutine and waits there. These are the
// tests for what can still reach it while it is parked: the clock, and a word
// left on the task by somebody who was not watching. They live apart from
// gate_test.go because that file holds what the gate decides, and this one
// holds what happens to a run that has already stopped deciding.

package task

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/record"
)

// TestARunParkedAtAGateStillDiesWhenItsDeadlinePasses is the property the
// whole design of a blocking gate leans on: a run that has stopped for a
// human is still a run the operating system and the clock can reach. Without
// it, deleting the ctx.Done() arm of the gate's select leaves every other
// test in this file green while a paused task ignores SIGTERM and -timeout
// for ever, and its log ends on phase.waiting with nothing after it.
func TestARunParkedAtAGateStillDiesWhenItsDeadlinePasses(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)
	fake := engine.NewFake("reviewed")

	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		done := make(chan error, 1)
		go func() {
			done <- Run(ctx, s, tk, gatedFlow(), map[string]engine.Engine{"fake": fake}, FileGate(s, time.Second))
		}()
		synctest.Wait()

		// Parked, and nobody is coming.
		find(t, eventsOf(t, s, tk), record.PhaseWaiting)

		time.Sleep(3 * time.Second)
		err := <-done
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Run returned %v, want context.DeadlineExceeded — a run parked at a gate has to die when its deadline passes", err)
		}
	})

	events := eventsOf(t, s, tk)
	wantKinds(t, events,
		record.TaskCreated, record.TaskStarted,
		record.PhaseWaiting, record.TaskTimedOut)
	if len(fake.Calls) != 0 {
		t.Errorf("the engine was called %d times, want 0 — the phase was never let go", len(fake.Calls))
	}
}

// TestAStaleResumeDoesNotWaveThroughTheNextRunsFirstGate covers the other
// half of the same file: a word left on a task nobody is running. `orbit
// resume` is the only way to clear a stale pause, and it writes a word of its
// own, so the word a reader leaves behind must not become permission for a
// phase they never saw.
func TestAStaleResumeDoesNotWaveThroughTheNextRunsFirstGate(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)
	fake := engine.NewFake("reviewed")

	// Nothing is running: this is the word left over from clearing a pause.
	if err := Control(s, tk, "resume"); err != nil {
		t.Fatalf("Control: %v", err)
	}

	synctest.Test(t, func(t *testing.T) {
		done := make(chan error, 1)
		go func() {
			done <- Run(context.Background(), s, tk, gatedFlow(), map[string]engine.Engine{"fake": fake}, FileGate(s, time.Second))
		}()
		synctest.Wait()

		waiting := find(t, eventsOf(t, s, tk), record.PhaseWaiting)
		if got := waiting.Data["why"]; got != "flow" {
			t.Errorf("phase.waiting says why=%q, want flow — the stale word must not stand in for the flow's own ask", got)
		}
		if len(fake.Calls) != 0 {
			t.Fatalf("the engine was called %d times, want 0 — a leftover resume let the review phase run unasked", len(fake.Calls))
		}

		if err := Control(s, tk, "continue"); err != nil {
			t.Fatalf("Control: %v", err)
		}
		if err := <-done; err != nil {
			t.Fatalf("Run: %v", err)
		}
	})
}
