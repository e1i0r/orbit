//go:build integration

package integration

// The `quick` flow: one phase, no gate, and the work has to be on disk when
// it is over.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestQuickWalksItsOnePhase.
func TestQuickWalksItsOnePhase(t *testing.T) {
	b := newBoard(t, map[string]any{
		"implement": []any{map[string]any{
			"write": map[string]string{
				"ledger.go": "package ledger\n\n// Total adds a charge and a refund.\nfunc Total(charge, refund int) int {\n\treturn charge + refund\n}\n",
			},
			"say":      "The refund was being subtracted twice.",
			"thoughts": []string{"reading ledger.go"},
			"cost":     0.25,
		}},
	})

	b.must(t, "new", "-repo", b.repo, "-id", "LED-1", "-flow", "quick", "fix the total")
	b.must(t, "run", "-repo", b.repo, "LED-1")

	events := b.record(t, "LED-1")

	for _, want := range []string{
		"task.created", "task.started", "phase.started", "phase.finished", "task.finished",
	} {
		if !holds(events, want) {
			t.Errorf("the record has no %s:\n%v", want, kinds(events))
		}
	}

	if n := count(events, "phase.started"); n != 1 {
		t.Errorf("the flow ran %d phases, want the one quick carries", n)
	}

	said := lastOf(events, "phase.finished")
	if !strings.Contains(said.Text, "subtracted twice") {
		t.Errorf("what the phase answered was not written down: %q", said.Text)
	}
}

// TestTheWorkIsOnDiskWhenTheRunEnds. A stand-in that only printed would
// leave the diff budget, the dependency gate and the story reading an empty
// change, and every test above them would pass on nothing.
func TestTheWorkIsOnDiskWhenTheRunEnds(t *testing.T) {
	b := newBoard(t, map[string]any{
		"implement": []any{map[string]any{
			"write": map[string]string{"NOTES.md": "the refund lands on the total twice\n"},
			"say":   "wrote it down",
		}},
	})

	b.must(t, "new", "-repo", b.repo, "-id", "LED-2", "-flow", "quick", "write down the bug")
	b.must(t, "run", "-repo", b.repo, "LED-2")

	found := worktreeFile(t, b, "NOTES.md")
	if !strings.Contains(found, "twice") {
		t.Errorf("the file the phase wrote holds %q", found)
	}
}

// lastOf is the last event of one kind, and the zero event when there is
// none — a test asserting on its fields fails on the emptiness.
func lastOf(events []event, kind string) event {
	var out event

	for _, e := range events {
		if e.Kind == kind {
			out = e
		}
	}

	return out
}

// worktreeFile reads one file out of the worktree a task was given, found by
// walking the state root rather than by rebuilding the hash of a path.
func worktreeFile(t *testing.T, b board, name string) string {
	t.Helper()

	var found string

	err := filepath.Walk(b.state, func(path string, info os.FileInfo, err error) error {
		if err == nil && found == "" && !info.IsDir() && filepath.Base(path) == name {
			found = path
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk the state root: %v", err)
	}

	if found == "" {
		t.Fatalf("no %s anywhere under the state root", name)
	}

	body, err := os.ReadFile(found)
	if err != nil {
		t.Fatalf("read %q: %v", found, err)
	}

	return string(body)
}
