package task

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
)

const scopedPlan = `Picked an approach.

## Decisions

- id: exponential-backoff
  scope: kept.txt
  decision: Retry a failed phase with exponential backoff and jitter.
  rejected: A fixed five-second delay.
`

// TestADecisionIsWrittenBesideTheCodeItGoverns. The event is the decision's
// home and the file is a copy of it — but the copy is what survives outside
// Orbit, and a reader who has the repository and not the state root is the
// one this is for.
func TestADecisionIsWrittenBesideTheCodeItGoverns(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-21", "decide something", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	f := flow.Flow{Name: "task", Phases: []flow.Phase{{Name: "1-plan", Engine: "fake"}}}
	if err := Run(context.Background(), s, tk, f, map[string]engine.Engine{"fake": engine.NewFake(scopedPlan)}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	wt, err := s.WorktreeDir(r.Path, tk.ID)
	if err != nil {
		t.Fatalf("WorktreeDir: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(wt, ".orbit", "decisions", "exponential-backoff.md"))
	if err != nil {
		t.Fatalf("read the decision file: %v", err)
	}

	text := string(body)
	for _, want := range []string{"ACME-21", "kept.txt", "exponential backoff", "five-second delay", "1-plan"} {
		if !strings.Contains(text, want) {
			t.Errorf("the decision file does not carry %q:\n%s", want, text)
		}
	}
}

// TestADecisionFileIsRewrittenAndNotRepeated. A plan that ran twice decided
// the same thing twice, and two files saying it are a repository arguing
// with itself.
func TestADecisionFileIsRewrittenAndNotRepeated(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-22", "decide it twice", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	f := flow.Flow{Name: "task", Phases: []flow.Phase{{Name: "1-plan", Engine: "fake"}}}
	engines := map[string]engine.Engine{"fake": engine.NewFake(scopedPlan)}

	for range 2 {
		if err := Run(context.Background(), s, tk, f, engines, nil); err != nil {
			t.Fatalf("Run: %v", err)
		}
	}

	wt, err := s.WorktreeDir(r.Path, tk.ID)
	if err != nil {
		t.Fatalf("WorktreeDir: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(wt, ".orbit", "decisions"))
	if err != nil {
		t.Fatalf("read the decisions directory: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("the second run left %d decision files, want the one rewritten", len(entries))
	}
}

// TestOrbitsOwnFilesAreNotTheTasksDiff. Orbit writes the decision into the
// worktree, and a gate of Orbit's that counted it would be Orbit refusing a
// change it made itself.
func TestOrbitsOwnFilesAreNotTheTasksDiff(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-23", "a plan and nothing else", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// A budget of one line: the decision file is longer than that, and the
	// plan changed nothing else at all.
	f := flow.Flow{Name: "task", DiffBudget: 1, Phases: []flow.Phase{
		{Name: "1-plan", Engine: "fake"},
		{Name: "2-implement", Engine: "fake"},
	}}

	if err := Run(context.Background(), s, tk, f, map[string]engine.Engine{"fake": engine.NewFake(scopedPlan)}, nil); err != nil {
		t.Fatalf("Run: %v — Orbit's own decision file was counted against the task's budget", err)
	}
}
