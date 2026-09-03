package task

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

// judgingEngine answers one thing when it is asked to judge a change against
// a decision, and another when it is asked to do the work.
type judgingEngine struct {
	*engine.Fake

	verdict string
	writes  string
	asked   int
}

func (e *judgingEngine) Run(ctx context.Context, req engine.Request) (engine.Result, error) {
	if strings.Contains(req.Prompt, checkMarker) {
		e.asked++

		return engine.Result{Output: e.verdict}, nil
	}

	out, err := e.Fake.Run(ctx, req)
	if err != nil || e.writes == "" {
		return out, err
	}

	return out, os.WriteFile(filepath.Join(req.Dir, "kept.txt"), []byte("changed\n"), 0o600)
}

const planScoped = `Decided.

## Decisions

- id: keep-it-in-settings
  scope: kept.txt
  decision: The cap lives in the settings file, not in each flow.
`

func decisionFlow(allowContradictions bool) flow.Flow {
	return flow.Flow{Name: "task", AllowContradictions: allowContradictions, Phases: []flow.Phase{
		{Name: "1-plan", Engine: "fake"},
		{Name: "2-implement", Engine: "fake"},
	}}
}

// TestAChangeThatContradictsADecisionStopsTheRun. A decision nothing checks
// is a note; the check is what makes writing it down worth anything.
func TestAChangeThatContradictsADecisionStopsTheRun(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-24", "change what was decided", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := &judgingEngine{
		Fake:    engine.NewFake(planScoped),
		verdict: "verdict: contradicts keep-it-in-settings\nThe cap was moved into the flow file.",
		writes:  "kept.txt",
	}

	err = Run(context.Background(), s, tk, decisionFlow(false), map[string]engine.Engine{"fake": eng}, nil)
	if err == nil {
		t.Fatal("Run: want an error when the change contradicts a decision")
	}

	events, evErr := Events(s, tk)
	if evErr != nil {
		t.Fatalf("Events: %v", evErr)
	}

	last := events[len(events)-1]
	if last.Kind != record.TaskContradicts {
		t.Fatalf("the record ends in %q, want task.contradicts: %v", last.Kind, kindsOf(events))
	}

	if last.Data["decision"] != "keep-it-in-settings" {
		t.Errorf("task.contradicts names %q, want the decision the change went against", last.Data["decision"])
	}

	if !strings.Contains(last.Text, "moved into the flow file") {
		t.Errorf("task.contradicts does not carry why:\n%s", last.Text)
	}
}

// TestAChangeThatKeepsToTheDecisionRunsOn.
func TestAChangeThatKeepsToTheDecisionRunsOn(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-25", "change what was decided, consistently", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := &judgingEngine{
		Fake:    engine.NewFake(planScoped),
		verdict: "verdict: consistent\nThe cap is still read from the settings file.",
		writes:  "kept.txt",
	}

	if err := Run(context.Background(), s, tk, decisionFlow(false), map[string]engine.Engine{"fake": eng}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if eng.asked == 0 {
		t.Error("nothing was ever checked against the decision")
	}
}

// TestNothingIsAskedWhenNoDecisionGovernsTheChange. The check is a model
// call and a model call is money: a repository with no decision about the
// file being changed must not pay for one.
func TestNothingIsAskedWhenNoDecisionGovernsTheChange(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-26", "change something nobody decided about", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := &judgingEngine{
		Fake:    engine.NewFake("Nothing was decided here.\n"),
		verdict: "verdict: contradicts anything\nshould never be asked",
		writes:  "kept.txt",
	}

	if err := Run(context.Background(), s, tk, decisionFlow(false), map[string]engine.Engine{"fake": eng}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if eng.asked != 0 {
		t.Errorf("the check ran %d times over a change no decision governs", eng.asked)
	}
}

// TestAFlowMayAllowContradictions.
func TestAFlowMayAllowContradictions(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-27", "allowed to contradict", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eng := &judgingEngine{
		Fake:    engine.NewFake(planScoped),
		verdict: "verdict: contradicts keep-it-in-settings\nand nobody asked",
		writes:  "kept.txt",
	}

	if err := Run(context.Background(), s, tk, decisionFlow(true), map[string]engine.Engine{"fake": eng}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if eng.asked != 0 {
		t.Errorf("a flow that allows contradictions paid for %d checks", eng.asked)
	}
}

// TestAVerdictIsReadForItsDecisionAndItsReason. The answer is a model's, so
// the reading of it has to be forgiving about everything except the word
// that decides whether a run stops.
func TestAVerdictIsReadForItsDecisionAndItsReason(t *testing.T) {
	for _, tc := range []struct {
		answer string
		id     string
		why    string
	}{
		{"verdict: consistent", "", ""},
		{"Verdict: Consistent\nAll good.", "", ""},
		{"verdict: contradicts keep-it\nbecause it moved", "keep-it", "because it moved"},
		{"VERDICT: CONTRADICTS keep-it", "keep-it", ""},
		{"I could not tell.", "", ""},
	} {
		id, why := verdictIn(tc.answer)
		if id != tc.id || !strings.Contains(why, tc.why) {
			t.Errorf("%q reads as (%q, %q), want (%q, %q)", tc.answer, id, why, tc.id, tc.why)
		}
	}
}

// TestADecisionGovernsTheFilesUnderItsScope.
func TestADecisionGovernsTheFilesUnderItsScope(t *testing.T) {
	for _, tc := range []struct {
		scope string
		path  string
		want  bool
	}{
		{"internal/task/run.go", "internal/task/run.go", true},
		{"internal/task", "internal/task/run.go", true},
		{"internal/task/", "internal/task/run.go", true},
		{"internal/task", "internal/tasks/run.go", false},
		{"internal/task/run.go", "internal/task/gate.go", false},
		{"", "internal/task/run.go", false},
	} {
		if got := governs(tc.scope, tc.path); got != tc.want {
			t.Errorf("scope %q over %q = %v, want %v", tc.scope, tc.path, got, tc.want)
		}
	}
}
