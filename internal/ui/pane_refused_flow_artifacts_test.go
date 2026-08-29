package ui

// The three remaining panes and the branches of each: refusedLines' denial
// list, flowLines' per-phase execution states, and what artifactsLines says
// about a directory it has read, has not read yet, and could not read.

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

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

	// Closed, a node is the branch it hangs off and where it got to. What it
	// was set up with and what it said is what opening one is for.
	shut := strings.Join(m.flowLines(), "\n")
	for _, want := range []string{"implement", "review", "fix", "completed", "failed", "cancelled"} {
		if !strings.Contains(shut, want) {
			t.Errorf("flowLines closed = %q, want it to mention %q", shut, want)
		}
	}

	if strings.Contains(shut, "wrote the code") {
		t.Errorf("a closed node printed what its phase said anyway: %q", shut)
	}

	m.expandedDetail = true

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

// TestArtifactsNamesWhatIsOnDisk. Every row of this tab is read: the names
// and the sizes come from the task's own directory and the changed files
// from the diff, and a listing that cannot be checked against the disk is
// worse than no listing — this is the one tab a reader goes to to check.
func TestArtifactsNamesWhatIsOnDisk(t *testing.T) {
	m := openOn(t, "ACME-2662")
	m.diffKnown = true
	m.diff = "diff --git a/retry.go b/retry.go\n--- a/retry.go\n+++ b/retry.go\n"
	m.filesKnown = true
	m.files = []view.File{
		{Name: "task.md", Size: 512},
		{Name: "events.jsonl", Size: 2048},
		{Name: "notes.bin", Size: 3},
	}

	joined := strings.Join(m.artifactsLines(), "\n")
	for _, want := range []string{
		"what orbit wrote down", "3 files",
		"task.md", "512 B", "the task as it was written",
		"events.jsonl", "2 k", "one line per event",
		"what the run changed", "1 file", "retry.go",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("artifactsLines = %q, want it to mention %q", joined, want)
		}
	}

	// A name this build does not know is listed and left at that. The name
	// and the size were read and are true; a sentence invented beside them
	// would be the one part of the row that is not.
	row := rowOf(strings.Split(joined, "\n"), "notes.bin")
	if row < 0 {
		t.Fatalf("the file this build knows nothing about is not listed:\n%s", joined)
	}

	if plain := strings.TrimSpace(ansi.Strip(strings.Split(joined, "\n")[row])); !strings.HasSuffix(plain, "3 B") {
		t.Errorf("the unknown file's row = %q, want it to end at its size", plain)
	}
}

// TestArtifactsSaysWhichAnswerItIsWaitingFor. Nothing read yet, nothing
// there, and a read that failed are three different facts, and a tab that
// drew all three as an empty section would assert the second one it never
// observed.
func TestArtifactsSaysWhichAnswerItIsWaitingFor(t *testing.T) {
	for _, c := range []struct {
		name  string
		on    func(m Model) Model
		wants string
	}{{
		name:  "before the first answer",
		on:    func(m Model) Model { return m },
		wants: "reading the task's directory",
	}, {
		name:  "after an empty one",
		on:    func(m Model) Model { m.filesKnown = true; return m },
		wants: "has not run",
	}, {
		name:  "after a failure",
		on:    func(m Model) Model { m.filesErr = errors.New("open: permission denied"); return m },
		wants: "permission denied",
	}} {
		t.Run(c.name, func(t *testing.T) {
			joined := strings.Join(c.on(openOn(t, "ACME-2662")).artifactsLines(), "\n")
			if !strings.Contains(joined, c.wants) {
				t.Errorf("artifactsLines = %q, want it to say %q", joined, c.wants)
			}
		})
	}

	// A gone task is a sentence, not a crash.
	m := openOn(t, "no-such-task")
	if joined := strings.Join(m.artifactsLines(), "\n"); !strings.Contains(joined, "no longer on the board") {
		t.Errorf("artifactsLines for a gone task = %q, want the gone sentence", joined)
	}
}

// TestFormatBytesNamesItsUnit. A size is read off the disk now, so an empty
// file is nought bytes and says so: the listing this replaces clamped every
// size up to one byte because none of them had been measured.
func TestFormatBytesNamesItsUnit(t *testing.T) {
	for _, c := range []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1 k"},
		{2048, "2 k"},
		{1024*1024 - 1, "1023 k"},
		{3 * 1024 * 1024, "3 M"},
	} {
		if got := formatBytes(c.bytes); got != c.want {
			t.Errorf("formatBytes(%d) = %q, want %q", c.bytes, got, c.want)
		}
	}
}
