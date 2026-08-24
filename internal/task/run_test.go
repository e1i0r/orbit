package task

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
)

func oneFlow() flow.Flow {
	return flow.Flow{Name: "task", Phases: []flow.Phase{
		{Name: "implement", Engine: "fake", Model: "sonnet"},
	}}
}

func TestRunWalksEveryPhaseAndRecordsIt(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "retry the webhook on 5xx", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	fake := engine.NewFake("wrote the retry")
	engines := map[string]engine.Engine{"fake": fake}

	if err := Run(context.Background(), s, tk, oneFlow(), engines, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("the engine was called %d times, want 1", len(fake.Calls))
	}
	if fake.Calls[0].Model != "sonnet" {
		t.Errorf("model = %q, want sonnet", fake.Calls[0].Model)
	}
	if fake.Calls[0].Prompt == "" {
		t.Error("the engine was called with an empty prompt")
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	var kinds []string
	for _, e := range events {
		kinds = append(kinds, e.Kind)
	}
	want := []string{"task.created", "task.started", "phase.started", "phase.finished", "task.finished"}
	if len(kinds) != len(want) {
		t.Fatalf("events = %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Errorf("event %d = %q, want %q", i, kinds[i], want[i])
		}
	}
}

func TestRunRunsInsideAWorktreeOnItsOwnBranch(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	fake := engine.NewFake("ok")
	if err := Run(context.Background(), s, tk, oneFlow(), map[string]engine.Engine{"fake": fake}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	wt, err := s.WorktreeDir(r.Path, "ACME-1")
	if err != nil {
		t.Fatalf("WorktreeDir: %v", err)
	}
	if fake.Calls[0].Dir != wt {
		t.Errorf("the engine ran in %q, want the worktree %q", fake.Calls[0].Dir, wt)
	}
	if _, statErr := os.Stat(wt); statErr != nil {
		t.Errorf("the worktree was removed after a successful run: %v", statErr)
	}
	// The name of this test promises a branch, so ask git for one. Every
	// other assertion here is satisfied by a worktree checked out on the
	// repository's own HEAD, which would have the engine committing
	// straight onto main — the one outcome the branch exists to prevent.
	if got := headOf(t, wt); got != "orbit/ACME-1" {
		t.Errorf("the worktree is on %q, want orbit/ACME-1", got)
	}
}

// headOf is the branch a checkout is on, or "HEAD" when it is on none.
func headOf(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Env = append(cmd.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse in %q: %v", dir, err)
	}
	return strings.TrimSpace(string(out))
}

func TestRunRecordsAFailureAndStops(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	fake := engine.NewFake("")
	fake.Err = errors.New("the model fell over")
	f := flow.Flow{Name: "task", Phases: []flow.Phase{
		{Name: "implement", Engine: "fake"},
		{Name: "second", Engine: "fake"},
	}}

	if err := Run(context.Background(), s, tk, f, map[string]engine.Engine{"fake": fake}, nil); err == nil {
		t.Fatal("Run reported success after the engine failed")
	}
	if len(fake.Calls) != 1 {
		t.Errorf("the engine was called %d times — the flow carried on past a failure", len(fake.Calls))
	}
	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("events = %+v, want at least [phase.failed, task.failed] at the tail", events)
	}
	tail := events[len(events)-2:]
	wantTail := []string{"phase.failed", "task.failed"}
	for i, e := range tail {
		if e.Kind != wantTail[i] {
			t.Errorf("tail event %d = %q, want %q", i, e.Kind, wantTail[i])
		}
	}
	last := events[len(events)-1]
	if last.Text == "" {
		t.Error("the failure was recorded with no reason — evidence is never paraphrased away")
	}
}

func TestRunRecordsAPrepareFailureAndStops(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// A base branch that does not exist makes AddWorktree fail without
	// depending on the parked branch-collision bug: no directory and no
	// branch stand in the way, git simply has nothing to check the worktree
	// out from.
	tk.Repo.Base = "no-such-branch"

	fake := engine.NewFake("ok")
	if err := Run(context.Background(), s, tk, oneFlow(), map[string]engine.Engine{"fake": fake}, nil); err == nil {
		t.Fatal("Run reported success though the worktree could not be created")
	}
	if len(fake.Calls) != 0 {
		t.Errorf("the engine was called %d times though the worktree was never prepared", len(fake.Calls))
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	// task.started is in the middle because an attempt that could not make
	// a worktree is still an attempt, and the log says so. It is also what
	// keeps a re-run honest: the event clears the phase the attempt before
	// it died in, so the task.failed below is not read as a failure in that
	// phase.
	wantKinds(t, events, record.TaskCreated, record.TaskStarted, record.TaskFailed)
	if events[2].Text == "" {
		t.Error("the failure was recorded with no reason — evidence is never paraphrased away")
	}
}

func TestRunRejectsAnUnknownEngine(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	f := flow.Flow{Name: "task", Phases: []flow.Phase{{Name: "implement", Engine: "opencode"}}}
	if err := Run(context.Background(), s, tk, f, map[string]engine.Engine{"fake": engine.NewFake("")}, nil); err == nil {
		t.Error("Run accepted a phase naming an engine that is not configured")
	}
}

// resultEngine answers with a fixed engine.Result. It exists only to give a
// SessionID and Cost to an engine's answer, which engine.Fake cannot do.
type resultEngine struct {
	result engine.Result
}

func (e resultEngine) Name() string { return "fake" }

func (e resultEngine) CanResume() bool          { return false }
func (e resultEngine) Models() []engine.Choice  { return nil }
func (e resultEngine) Efforts() []engine.Choice { return nil }
func (e resultEngine) CanThink() bool           { return false }

func (e resultEngine) Run(context.Context, engine.Request) (engine.Result, error) {
	return e.result, nil
}

func TestRunRecordsSessionAndCostWhenTheEngineReportsThem(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	eng := resultEngine{result: engine.Result{Output: "done", SessionID: "sess-1", Cost: 0.42}}
	if err := Run(context.Background(), s, tk, oneFlow(), map[string]engine.Engine{"fake": eng}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	var finished record.Event
	found := false
	for _, e := range events {
		if e.Kind == "phase.finished" {
			finished, found = e, true
		}
	}
	if !found {
		t.Fatal("no phase.finished event was recorded")
	}
	if finished.Data["session"] != "sess-1" {
		t.Errorf(`Data["session"] = %q, want "sess-1"`, finished.Data["session"])
	}
	if finished.Data["cost"] != "0.42" {
		t.Errorf(`Data["cost"] = %q, want "0.42"`, finished.Data["cost"])
	}
}

func TestRunLeavesDataEmptyWhenTheEngineReportsNeither(t *testing.T) {
	s, r := fixture(t)
	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	fake := engine.NewFake("done")
	if err := Run(context.Background(), s, tk, oneFlow(), map[string]engine.Engine{"fake": fake}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	for _, e := range events {
		if e.Kind == "phase.finished" && e.Data != nil {
			t.Errorf("phase.finished has Data = %+v, want nil when the engine reports neither session nor cost", e.Data)
		}
	}
}
