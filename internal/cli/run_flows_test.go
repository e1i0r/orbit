package cli

// Covers runTask branches that TestRunNeedsAnID, TestRunFailsOnAnUnknownFlow
// and TestRunFailsOnATaskThatWasNeverCreated (cli_test.go) never reach: those
// three are built to return before task.Load ever succeeds (see the doc
// comment above them), so none of them walks as far as choosing a flow or
// calling task.Run at all.
//
// This file creates a real task and gives it a flow of its own, naming an
// engine no build of orbit has — "nonexistent-engine" rather than "claude" —
// so that task.Run fails fast on its own phase-validation loop, before any
// worktree is made and long before anything would exec a real binary. That
// keeps this deterministic and safe to run on a machine that happens to have
// claude/codex/opencode installed.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWalksAChosenFlowAndFailsInsideTaskRun(t *testing.T) {
	root, orbitHome := workspace(t)
	repoDir := filepath.Join(root, "payments")

	// A flow of its own, naming an engine this build cannot run, so
	// task.Run fails on the phase-validation loop rather than on anything
	// that would exec a real process.
	flowsDir := filepath.Join(orbitHome, "flows")
	if err := os.MkdirAll(flowsDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := `{"name":"nope","phases":[{"name":"implement","engine":"nonexistent-engine"}]}`
	if err := os.WriteFile(filepath.Join(flowsDir, "nope.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write flow: %v", err)
	}

	// 1. The task is written against that flow, so a run with no -flow
	// override walks chosen = t.Flow (run.go: `if chosen == "" { chosen =
	// t.Flow }`), a branch none of the cli_test.go run tests reach because
	// none of them gets past task.Load.
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-9", "-flow", "nope", "walk the custom flow"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	// 2. No -flow on the run: chosen starts "", falls back to t.Flow
	// ("nope"), flow.Resolve finds the user's own flow file, and task.Run is
	// reached and fails inside its phase-validation loop — covering the
	// runTask branches from flow.Resolve's success onward through the
	// task.Run error return, without spawning anything.
	code, _, errOut := run(t, "run", "-repo", repoDir, "-timeout", "50ms", "ACME-9")
	if code == 0 {
		t.Error("run with an unconfigured engine exited 0")
	}
	if !strings.Contains(errOut, "nonexistent-engine") {
		t.Errorf("the refusal does not name the engine:\n%s", errOut)
	}
}

// A run whose -flow names a real user flow directly (no fallback to t.Flow)
// still has to reach the same task.Run failure — asserted separately so the
// -flow override path and the *timeout > 0 branch are each exercised once on
// their own terms.
func TestRunWithExplicitFlowAndTimeoutReachesTaskRun(t *testing.T) {
	root, orbitHome := workspace(t)
	repoDir := filepath.Join(root, "payments")

	flowsDir := filepath.Join(orbitHome, "flows")
	if err := os.MkdirAll(flowsDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := `{"name":"nope2","phases":[{"name":"implement","engine":"nonexistent-engine"}]}`
	if err := os.WriteFile(filepath.Join(flowsDir, "nope2.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write flow: %v", err)
	}

	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-10", "plain task"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	code, _, errOut := run(t, "run", "-repo", repoDir, "-flow", "nope2", "-timeout", "1s", "ACME-10")
	if code == 0 {
		t.Error("run with an unconfigured engine exited 0")
	}
	if !strings.Contains(errOut, "nonexistent-engine") {
		t.Errorf("the refusal does not name the engine:\n%s", errOut)
	}
}
