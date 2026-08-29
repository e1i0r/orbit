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
	"github.com/e1i0r/orbit/internal/record"
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
func (capableEngine) Locate() (string, error)  { return "capable", nil }
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
func (oversizedSessionEngine) Locate() (string, error)  { return "oversized", nil }
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

// TestRunOnATaskIdTheStateRootRefuses stops before anything is written at
// all: the marker goes down first, and an id that is not a single path
// element has nowhere to put one.
//
// This test used to say it covered the task.started write. It did not -- the
// hold above that write fails on the same id, so Run returned one line
// earlier and the branch the name promised was never entered. The one below
// enters it.
func TestRunOnATaskIdTheStateRootRefuses(t *testing.T) {
	s, r := fixture(t)
	bad := Task{ID: "has/slash", Repo: r}
	f := flow.Flow{Name: "f", Phases: []flow.Phase{{Name: "p", Engine: "fake"}}}

	engines := map[string]engine.Engine{"fake": engine.NewFake("out")}
	if err := Run(context.Background(), s, bad, f, engines, nil); err == nil {
		t.Error("Run walked a task whose id the state root cannot hold")
	}
}

// TestRunTaskStartedEmitFailure covers Run's very first write failing, with
// the marker already held.
//
// task.started carries the flow's name, and a flow name past record.MaxLine
// is the one part of that event a caller controls. What the run must not do
// is carry on: every later failure writes task.failed, and a task.failed on a
// log with no task.started tells every reader the phase from the *previous*
// attempt is still the current one.
func TestRunTaskStartedEmitFailure(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "RUN-TSTART-ERR-1", "task started emit error test", "quick")
	if err != nil {
		t.Fatal(err)
	}

	f := flow.Flow{
		Name:   strings.Repeat("F", 5<<20),
		Phases: []flow.Phase{{Name: "p", Engine: "fake"}},
	}

	engines := map[string]engine.Engine{"fake": engine.NewFake("out")}
	if err := Run(context.Background(), s, tk, f, engines, nil); err == nil {
		t.Error("Run walked on after the log refused its task.started")
	}

	got, err := Events(s, tk)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range got {
		if e.Kind == "task.failed" {
			t.Errorf("the log carries a task.failed with no task.started above it; kinds are %v", kindsOf(got))
		}
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

// unrecordableEngine answers with an output the record cannot hold.
//
// captured cuts an engine's answer to a megabyte, and record.MaxLine is four
// — but a megabyte of control characters is six megabytes once JSON has
// escaped every one of them to \u0000. So phase.finished is refused by the
// log while the short task.failed after it fits easily: one event that
// cannot be written, in a log that is otherwise perfectly writable.
type unrecordableEngine struct{}

func (unrecordableEngine) Name() string             { return "unrecordable" }
func (unrecordableEngine) CanResume() bool          { return false }
func (unrecordableEngine) CanThink() bool           { return false }
func (unrecordableEngine) Models() []engine.Choice  { return nil }
func (unrecordableEngine) Efforts() []engine.Choice { return nil }
func (unrecordableEngine) Locate() (string, error)  { return "unrecordable", nil }

func (unrecordableEngine) Run(_ context.Context, _ engine.Request) (engine.Result, error) {
	return engine.Result{Output: strings.Repeat("\x00", maxOutput)}, nil
}

// TestALogThatCannotBeWrittenIsStillClosed is the fix.
//
// The three emits on Run's success path returned their error bare. The
// deferred release then took the marker off cleanly, so the log was left
// ending at phase.started with no terminal event and no stale marker — the
// one state Reconcile cannot repair, because it acts on a marker that is
// still there. Every reader of that record shows the task as running, for
// ever, and nothing will ever say otherwise.
func TestALogThatCannotBeWrittenIsStillClosed(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "RUN-CLOSE-1", "an answer the record cannot hold", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	f := flow.Flow{Name: "one", Phases: []flow.Phase{{Name: "implement", Engine: "unrecordable"}}}

	engines := map[string]engine.Engine{"unrecordable": unrecordableEngine{}}
	if runErr := Run(context.Background(), s, tk, f, engines, nil); runErr == nil {
		t.Fatal("Run returned nil after the phase's ending was refused by the log")
	}

	got, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	if inFlight(got) {
		t.Errorf("the log is still open after the run returned; kinds are %v", kindsOf(got))
	}

	if _, alive, err := Alive(s, tk); err != nil || alive {
		t.Errorf("Alive = (%v, %v), want the marker released", alive, err)
	}
}

func kindsOf(events []record.Event) []string {
	out := make([]string, 0, len(events))
	for _, e := range events {
		out = append(out, e.Kind)
	}

	return out
}
