package task

import (
	"context"
	"errors"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// cancellingEngine finishes its phase and then stops the run, so that the
// gate which follows is what the cancellation lands on. It wraps the fake
// rather than replacing it because everything else about the phase should
// happen exactly as it does in any other run.
type cancellingEngine struct {
	*engine.Fake
	cancel context.CancelFunc
}

func (e cancellingEngine) Run(ctx context.Context, req engine.Request) (engine.Result, error) {
	out, err := e.Fake.Run(ctx, req)
	e.cancel()

	return out, err
}

// commandGateFlow is one phase with one shell gate on it. The name says
// command because gatedFlow is already taken by a flow whose phase asks a
// person to wait, which is the other thing this package calls a gate.
func commandGateFlow(command string) flow.Flow {
	return flow.Flow{Name: "task", Phases: []flow.Phase{{
		Name:   "implement",
		Engine: "fake",
		Gates:  []flow.Gate{{Name: "build", Command: command}},
	}}}
}

// TestARunCancelledWhileAGateIsUpSaysItWasCancelled.
//
// The gate is a command exec kills when the run is stopped, and a killed
// command reports the way a command that exited non-zero reports. Nothing
// asked the context, so the record said gate.failed, phase.failed and
// task.failed — the build broke — about a run somebody had cancelled, and
// carried an exit status of 1 that no gate ever returned.
//
// The engine's own error path a hundred lines away has always asked the
// context first, and this is that same question.
func TestARunCancelledWhileAGateIsUpSaysItWasCancelled(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "retry the webhook on 5xx", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := cancellingEngine{engine.NewFake("wrote the retry"), cancel}

	runErr := Run(ctx, s, tk, commandGateFlow("sleep 30"), map[string]engine.Engine{"fake": eng}, nil)
	if !errors.Is(runErr, context.Canceled) {
		t.Fatalf("Run returned %v, want an error carrying context.Canceled", runErr)
	}

	events := eventsOf(t, s, tk)
	wantKinds(t, events,
		record.TaskCreated, record.TaskStarted, record.PhaseStarted,
		record.PhaseCancelled, record.TaskCancelled)

	// The phase keeps what the engine printed before the run was stopped,
	// which is the one thing a reader has to go on afterwards.
	if got := find(t, events, record.PhaseCancelled).Text; got != "wrote the retry" {
		t.Errorf("phase.cancelled text = %q, want what the engine printed before the gate was killed", got)
	}
}

// TestAGateThatFailsOnItsOwnStillSaysSo is the other half: with nothing
// cancelling anything, a gate that exits non-zero is still a failed run, and
// the check above must not swallow it.
func TestAGateThatFailsOnItsOwnStillSaysSo(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-2", "retry the webhook on 5xx", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := engine.NewFake("wrote the retry")

	runErr := Run(context.Background(), s, tk, commandGateFlow("exit 3"), map[string]engine.Engine{"fake": eng}, nil)
	if runErr == nil {
		t.Fatal("Run on a gate that exited 3 returned nil, want the failure")
	}

	// Three times, because the flow says nothing about attempts and the
	// default is three: the gate refuses, the phase is run again, and only
	// the attempt with nothing after it ends the phase and the run.
	events := eventsOf(t, s, tk)
	wantKinds(t, events,
		record.TaskCreated, record.TaskStarted,
		record.PhaseStarted, record.GateFailed, record.PhaseRetried,
		record.PhaseStarted, record.GateFailed, record.PhaseRetried,
		record.PhaseStarted, record.GateFailed, record.PhaseFailed, record.TaskFailed)

	for _, e := range events {
		if e.Kind != record.GateFailed {
			continue
		}

		if got := e.Data["exit"]; got != "3" {
			t.Errorf("gate.failed exit = %q, want the 3 the gate returned", got)
		}
	}
}
