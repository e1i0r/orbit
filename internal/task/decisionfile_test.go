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
	if err := Run(context.Background(), s, tk, f, fakes(engine.NewFake(scopedPlan)), nil); err != nil {
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
	engines := fakes(engine.NewFake(scopedPlan))

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

	if err := Run(context.Background(), s, tk, f, fakes(engine.NewFake(scopedPlan)), nil); err != nil {
		t.Fatalf("Run: %v — Orbit's own decision file was counted against the task's budget", err)
	}
}

// TestADecisionsFileIsNamedAfterWhatItDecided.
//
// The name is what a reader browsing the folder sees before they open
// anything, so it has to carry the decision and stop before it becomes the
// whole sentence: a directory of filenames a line long is a directory
// nobody reads.
func TestADecisionsFileIsNamedAfterWhatItDecided(t *testing.T) {
	for _, one := range []struct {
		text string
		want string
	}{
		{"Cents, not floats!", "cents-not-floats"},
		{"Retry the webhook on 5xx", "retry-the-webhook-on-5xx"},
		// Six words is where it stops, and a sentence of exactly six keeps
		// all of them: cutting one earlier loses the word that usually
		// carries the decision.
		{"one two three four five six", "one-two-three-four-five-six"},
		{"amounts are always in cents and never floats", "amounts-are-always-in-cents-and"},
		{"", ""},
		{"   ", ""},
		{"---", ""},
	} {
		if got := slug(one.text); got != one.want {
			t.Errorf("slug(%q) = %q, want %q", one.text, got, one.want)
		}
	}
}

// TestADecisionsCopyGoesBesideTheCodeItNames.
//
// One copy and not one per repository: a decision written into three
// checkouts is three files that will disagree the first time one of them is
// edited, and the scope is what says which of the three the reader will look
// in. A task that reaches into two repositories is the ordinary case for
// this — with one, every answer is the same answer.
func TestADecisionsCopyGoesBesideTheCodeItNames(t *testing.T) {
	s, repos := workspaceFixture(t, "api", "ledger")

	tk, err := Create(s, repos[0], "ACME-22", "decide something", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	second, err := Join(s, tk, repos[1])
	if err != nil {
		t.Fatalf("Join: %v", err)
	}

	// The file the decision names, in the second checkout and nowhere else.
	if err := os.MkdirAll(second, 0o750); err != nil {
		t.Fatalf("make the second worktree: %v", err)
	}

	if err := os.WriteFile(filepath.Join(second, "postings.go"), []byte("package ledger\n"), 0o600); err != nil {
		t.Fatalf("write the file it is about: %v", err)
	}

	where, found := worktreeFor(s, tk, decision{ID: "d1", Scope: "postings.go"})
	if !found {
		t.Fatal("a decision naming a file that is there found nowhere to go")
	}

	if where != second {
		t.Errorf("the copy goes to %q, want the checkout holding the file it names (%q)", where, second)
	}

	// A decision naming nothing that can be found falls back to where the
	// task is being worked, which is an answer and not a refusal.
	first, err := s.WorktreeDir(repos[0].Path, tk.ID)
	if err != nil {
		t.Fatalf("WorktreeDir: %v", err)
	}

	back, found := worktreeFor(s, tk, decision{ID: "d2", Scope: "nowhere.go"})
	if !found || back != first {
		t.Errorf("a decision naming nothing anybody has goes to %q, want %q", back, first)
	}

	// And one that names no place at all is the same case, rather than the
	// empty scope matching everything it is split into.
	bare, found := worktreeFor(s, tk, decision{ID: "d3"})
	if !found || bare != first {
		t.Errorf("a decision naming no place goes to %q, want %q", bare, first)
	}
}
