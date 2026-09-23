package task

// A hold the decision engine put on a run under autopilot stays until a
// person lifts it.

import (
	"context"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/hunch"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// TestAutopilotDoesNotLiftTheEnginesHold. With autopilot and decisions on,
// the engine said a person is needed and the run stopped; the wait then
// read the switch, found it on, and let the run go one poll later. The
// engine was also asked a second time and a second decision written down.
// Now the run stays stopped, the engine is asked once, and resume lets it go.
func TestAutopilotDoesNotLiftTheEnginesHold(t *testing.T) {
	s, r := fixture(t)

	cfg := store.Settings{Autopilot: true, Decisions: store.DecisionsOn, DecisionFloor: 70}
	if err := s.SaveSettings(cfg); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	tk, err := Create(s, r, "ACME-80", "make the endpoint idempotent", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	port := &saidSo{verdict: sure(hunch.Human, 0.95)}
	f := flow.Flow{Name: "careful", Phases: []flow.Phase{{Name: "review", Engine: "fake", Wait: true}}}

	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()

	done := make(chan error, 1)

	go func() {
		done <- Run(ctx, s, tk, f, fakes(engine.NewFake("did the work")),
			FileGate(s, 10*time.Millisecond, port))
	}()

	for !parked(mustEvents(t, s, tk)) {
		if ctx.Err() != nil {
			t.Fatal("the run never stopped for the engine's hold")
		}

		time.Sleep(10 * time.Millisecond)
	}

	// Twenty polls of the gate with autopilot on.
	time.Sleep(200 * time.Millisecond)

	events := mustEvents(t, s, tk)
	if resumed, how := letGo(events); resumed {
		t.Fatalf("the engine's hold was lifted by %q", how)
	}

	decisions := 0

	for _, e := range events {
		if e.Kind == record.Decided {
			decisions++
		}
	}

	if len(port.asked) != 1 || decisions != 1 {
		t.Errorf("the engine was asked %d times and %d decisions written, want one of each",
			len(port.asked), decisions)
	}

	if err := Control(s, tk, wordResume); err != nil {
		t.Fatalf("resume: %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("Run: %v", err)
	}

	if resumed, how := letGo(mustEvents(t, s, tk)); !resumed || how != wordResume {
		t.Errorf("the run was let go by %q (resumed=%v), want the reader's resume", how, resumed)
	}
}
