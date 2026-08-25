package ui

// pane_refused_flow_artifacts_coverage_test.go covers the three remaining
// panes with real branches left untested: refusedLines' denial list,
// flowLines' per-phase execution states, and artifactsLines' four groups.

import (
	"errors"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

func TestRefusedLinesWithAndWithoutDenials(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	m.logErr = errors.New("record damaged")
	if joined := strings.Join(m.refusedLines(), "\n"); !strings.Contains(joined, "record damaged") {
		t.Errorf("refusedLines with logErr = %q, want it to name the error", joined)
	}
	m.logErr = nil

	if joined := strings.Join(m.refusedLines(), "\n"); !strings.Contains(joined, "no commands or actions were denied") {
		t.Errorf("refusedLines with no entries = %q, want the empty sentence", joined)
	}

	m.entries = []view.Entry{
		{Kind: "phase.refused", Tool: "psql", Text: "psql -c 'drop table tasks'"},
		{Kind: "phase.refused", Text: "rm -rf /"}, // no tool named: falls back to "command"
	}
	joined := strings.Join(m.refusedLines(), "\n")
	for _, want := range []string{"psql", "drop table", "command", "rm -rf"} {
		if !strings.Contains(joined, want) {
			t.Errorf("refusedLines with denials = %q, want it to mention %q", joined, want)
		}
	}
}

// TestFlowLinesEveryPhaseState drives findPhaseExec (and so flowLines)
// through a failed phase, a cancelled one, one waiting at a gate, one
// finished, and one in flight, plus the flow-not-found error branch.
func TestFlowLinesEveryPhaseState(t *testing.T) {
	// The fixture's ACME-2662 walks the "careful" flow, whose three phases
	// are implement, review and fix — real names, so findPhaseExec's fold
	// and flowLines' per-phase icon actually see something to key off.
	m := openOn(t, "ACME-2662")
	m.entries = []view.Entry{
		{Kind: "phase.started", Phase: "implement", Attempt: 1, Engine: "claude", Model: "opus"},
		{Kind: "phase.finished", Phase: "implement", Attempt: 1, Cost: 0.5, Text: "wrote the code"},
		{Kind: "phase.started", Phase: "review", Attempt: 1, Engine: "claude", Model: "opus"},
		{Kind: "phase.failed", Phase: "review", Attempt: 1, Cause: "tests failed", Exit: "1"},
		{Kind: "phase.cancelled", Phase: "fix", Attempt: 1, Text: "stopped by the reader"},
	}
	joined := strings.Join(m.flowLines(), "\n")
	for _, want := range []string{"implement", "review", "fix", "tests failed", "wrote the code", "stopped by the reader"} {
		if !strings.Contains(joined, want) {
			t.Errorf("flowLines = %q, want it to mention %q", joined, want)
		}
	}

	// A phase waiting at a gate, and one still in flight because the task's
	// own Band and Phase point at it with no entries of its own yet.
	m.entries = []view.Entry{{Kind: "phase.waiting", Phase: "implement", Attempt: 1, Cause: "needs sign-off"}}
	tk, ok := m.task(m.detail)
	if !ok {
		t.Fatal("fixture task not found")
	}
	tk.Band, tk.Phase = view.Running, "review"
	m.board.Tasks = replaceTask(m.board.Tasks, tk)
	joined = strings.Join(m.flowLines(), "\n")
	for _, want := range []string{"waiting at gate", "in progress"} {
		if !strings.Contains(joined, want) {
			t.Errorf("flowLines waiting/in-flight = %q, want it to mention %q", joined, want)
		}
	}

	// A flow that does not resolve is a sentence, not a crash.
	tk.Flow = "not-a-real-flow"
	m.board.Tasks = replaceTask(m.board.Tasks, tk)
	joined = strings.Join(m.flowLines(), "\n")
	if !strings.Contains(joined, "not-a-real-flow") {
		t.Errorf("flowLines with an unresolved flow = %q, want it to name the flow", joined)
	}

	// The task itself is gone from the board.
	m2 := openOn(t, "no-such-task")
	if joined := strings.Join(m2.flowLines(), "\n"); !strings.Contains(joined, "no longer on the board") {
		t.Errorf("flowLines for a gone task = %q, want the gone sentence", joined)
	}
}

// replaceTask swaps in t for the task of the same id, so a test can change
// one field of the fixture board without rebuilding it whole.
func replaceTask(tasks []view.Task, t view.Task) []view.Task {
	out := make([]view.Task, len(tasks))
	for i, existing := range tasks {
		if existing.ID == t.ID {
			out[i] = t
			continue
		}
		out[i] = existing
	}
	return out
}

func TestArtifactsLinesGroupsAndFormatBytes(t *testing.T) {
	m := openOn(t, "ACME-2662")
	m.diffKnown = true
	m.diff = "diff --git a/retry.go b/retry.go\n--- a/retry.go\n+++ b/retry.go\n"
	m.entries = []view.Entry{
		{Kind: "phase.finished", Phase: "implement", Text: "wrote retry.go"},
		{Kind: "gate.failed", Gate: "vet", Cause: "unreachable code"},
	}
	tk, ok := m.task(m.detail)
	if !ok {
		t.Fatal("fixture task not found")
	}
	tk.Flow, tk.Cost = "careful", 1.25
	m.board.Tasks = replaceTask(m.board.Tasks, tk)

	joined := strings.Join(m.artifactsLines(), "\n")
	for _, want := range []string{
		"what it produced", "the checks & gates", "gates.reason",
		"what it was asked for", "task.env", "run accounting", "cost.tsv",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("artifactsLines = %q, want it to mention %q", joined, want)
		}
	}

	// A gone task is a sentence, not a crash.
	m2 := openOn(t, "no-such-task")
	if joined := strings.Join(m2.artifactsLines(), "\n"); !strings.Contains(joined, "no longer on the board") {
		t.Errorf("artifactsLines for a gone task = %q, want the gone sentence", joined)
	}
}

func TestFormatBytesBothSides(t *testing.T) {
	if got := formatBytes(0); got != "1 B" {
		t.Errorf("formatBytes(0) = %q, want %q (clamped to at least one byte)", got, "1 B")
	}
	if got := formatBytes(512); got != "512 B" {
		t.Errorf("formatBytes(512) = %q, want %q", got, "512 B")
	}
	if got := formatBytes(2048); got != "2 k" {
		t.Errorf("formatBytes(2048) = %q, want %q", got, "2 k")
	}
}
