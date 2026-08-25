package task

// The branches of Run that a straightforward success or a straightforward
// failure never reaches: a model or effort that does match, an engine with
// no effort dial asked for one anyway, a gate that errors instead of merely
// stopping, and the two mid-run events whose own write can fail without the
// run itself having gone wrong.

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
)

// capableEngine is a fake with a model and an effort a phase can actually
// name, so the validation loop's "found" branches run instead of only their
// "not found" opposites.
type capableEngine struct{}

func (capableEngine) Name() string { return "capable" }
func (capableEngine) Run(ctx context.Context, req engine.Request) (engine.Result, error) {
	return engine.Result{Output: "ok"}, nil
}
func (capableEngine) Models() []engine.Choice  { return []engine.Choice{{ID: "model-a"}} }
func (capableEngine) Efforts() []engine.Choice { return []engine.Choice{{ID: "effort-a"}} }
func (capableEngine) CanThink() bool           { return true }
func (capableEngine) CanResume() bool          { return false }

func TestRunModelAndEffortMatchesRunTheEngine(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "RUN-MATCH-1", "model and effort match test", "quick")
	if err != nil {
		t.Fatal(err)
	}
	f := flow.Flow{Name: "match", Phases: []flow.Phase{
		{Name: "phase-1", Engine: "capable", Model: "model-a", Effort: "effort-a"},
	}}
	engines := map[string]engine.Engine{"capable": capableEngine{}}
	if err := Run(context.Background(), s, tk, f, engines, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRunEffortNamedOnAnEngineWithNoEffortDial(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "RUN-NO-EFFORT-1", "no effort dial test", "quick")
	if err != nil {
		t.Fatal(err)
	}
	f := flow.Flow{Name: "no-effort", Phases: []flow.Phase{
		{Name: "phase-1", Engine: "fake", Effort: "high"},
	}}
	engines := map[string]engine.Engine{"fake": engine.NewFake("out")}
	err = Run(context.Background(), s, tk, f, engines, nil)
	if err == nil || !strings.Contains(err.Error(), "has no effort dial") {
		t.Errorf("Run = %v, want a \"has no effort dial\" error", err)
	}
}

// erroringGate answers every phase with an error rather than a decision, the
// one shape ask must hand straight back to Run rather than treat as a stop.
type erroringGate struct{ err error }

func (g erroringGate) Before(context.Context, Task, flow.Phase, int) (Go, error) {
	return Continue, g.err
}

func TestRunGateErrorIsReportedAsFailure(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "RUN-GATE-ERR-1", "gate error test", "quick")
	if err != nil {
		t.Fatal(err)
	}
	f := flow.Flow{Name: "gate-err", Phases: []flow.Phase{
		{Name: "phase-1", Engine: "fake"},
	}}
	engines := map[string]engine.Engine{"fake": engine.NewFake("out")}
	g := erroringGate{err: errors.New("boom")}
	err = Run(context.Background(), s, tk, f, engines, g)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("Run = %v, want the gate's own error wrapped in", err)
	}
}

// TestRunPhaseStartedEmitFailure covers the phaseStart write failing: an
// absurdly long phase name pushes the event itself over record.MaxLine
// before anything is run.
func TestRunPhaseStartedEmitFailure(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "RUN-PSTART-ERR-1", "phase started emit error test", "quick")
	if err != nil {
		t.Fatal(err)
	}
	f := flow.Flow{Name: "oversized-name", Phases: []flow.Phase{
		{Name: strings.Repeat("N", 5<<20), Engine: "fake"},
	}}
	engines := map[string]engine.Engine{"fake": engine.NewFake("out")}
	if err := Run(context.Background(), s, tk, f, engines, nil); err == nil {
		t.Error("Run should have failed to record an oversized phase.started event")
	}
}

// oversizedSessionEngine returns a result whose session id alone is over
// record.MaxLine, which phaseEnd does not truncate the way it truncates
// output — so the phase itself succeeds and only the finishing write fails.
type oversizedSessionEngine struct{}

func (oversizedSessionEngine) Name() string { return "oversized-session" }
func (oversizedSessionEngine) Run(ctx context.Context, req engine.Request) (engine.Result, error) {
	return engine.Result{Output: "ok", SessionID: strings.Repeat("S", 5<<20)}, nil
}
func (oversizedSessionEngine) Models() []engine.Choice  { return nil }
func (oversizedSessionEngine) Efforts() []engine.Choice { return nil }
func (oversizedSessionEngine) CanThink() bool           { return false }
func (oversizedSessionEngine) CanResume() bool          { return false }

func TestRunPhaseFinishedEmitFailure(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "RUN-PFINISH-ERR-1", "phase finished emit error test", "quick")
	if err != nil {
		t.Fatal(err)
	}
	f := flow.Flow{Name: "oversized-session", Phases: []flow.Phase{
		{Name: "phase-1", Engine: "sess"},
	}}
	engines := map[string]engine.Engine{"sess": oversizedSessionEngine{}}
	if err := Run(context.Background(), s, tk, f, engines, nil); err == nil {
		t.Error("Run should have failed to record an oversized phase.finished event")
	}
}

// TestRunTaskStartedEmitFailure covers Run's very first write failing: a bad
// task id, before anything else about the run has happened.
func TestRunTaskStartedEmitFailure(t *testing.T) {
	s, r := fixture(t)
	bad := Task{ID: "has/slash", Repo: r}
	f := flow.Flow{Name: "f", Phases: []flow.Phase{{Name: "p", Engine: "fake"}}}
	engines := map[string]engine.Engine{"fake": engine.NewFake("out")}
	if err := Run(context.Background(), s, bad, f, engines, nil); err == nil {
		t.Error("Run should have failed to record task.started for a bad task id")
	}
}

// TestRunHoldFailureAfterTaskStarted covers hold's own failure once
// task.started already landed: the task directory keeps its existing files
// writable (so that first event still appends) but loses the ability to
// create the new "run" marker file the hold needs.
func TestRunHoldFailureAfterTaskStarted(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "RUN-HOLD-ERR-1", "run hold error test", "quick")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := s.TaskDir(r.Path, tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) //nolint:errcheck

	f := flow.Flow{Name: "f", Phases: []flow.Phase{{Name: "p", Engine: "fake"}}}
	engines := map[string]engine.Engine{"fake": engine.NewFake("out")}
	if err := Run(context.Background(), s, tk, f, engines, nil); err == nil {
		t.Error("Run should have failed when the run marker cannot be created")
	}
}
