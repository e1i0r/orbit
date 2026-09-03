package repo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWorktreeChangesCountsWhatWasWrittenAndWhatWasCommitted. A budget an
// engine can walk past by committing is not a budget, so both halves of the
// work are one count.
func TestWorktreeChangesCountsWhatWasWrittenAndWhatWasCommitted(t *testing.T) {
	isolateGitConfig(t)

	dir := filepath.Join(t.TempDir(), "src")
	if err := mkdir(dir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	makeRepo(t, dir, "main", "")

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	wt := filepath.Join(t.TempDir(), "wt")
	if err := r.AddWorktree(wt, "orbit/ACME-1"); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}

	// One file committed on the task's own branch.
	if err := os.WriteFile(filepath.Join(wt, "committed.txt"), []byte("a\nb\nc\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := r.CommitWorktree(wt, "the phase committed this"); err != nil {
		t.Fatalf("CommitWorktree: %v", err)
	}

	// One file left in the tree, never added.
	if err := os.WriteFile(filepath.Join(wt, "loose.txt"), []byte("d\ne\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	changes, err := r.WorktreeChanges(wt)
	if err != nil {
		t.Fatalf("WorktreeChanges: %v", err)
	}

	lines := 0
	seen := map[string]bool{}

	for _, c := range changes {
		seen[c.Path] = true
		lines += c.Lines()
	}

	if !seen["committed.txt"] || !seen["loose.txt"] {
		t.Errorf("the count names %v, want both the committed file and the loose one", seen)
	}

	if lines != 5 {
		t.Errorf("the worktree changed %d lines, want the 3 committed plus the 2 written", lines)
	}
}

// TestAWorktreeThatChangedNothingCountsNothing.
func TestAWorktreeThatChangedNothingCountsNothing(t *testing.T) {
	isolateGitConfig(t)

	dir := filepath.Join(t.TempDir(), "src")
	if err := mkdir(dir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	makeRepo(t, dir, "main", "")

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	wt := filepath.Join(t.TempDir(), "wt")
	if err := r.AddWorktree(wt, "orbit/ACME-2"); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}

	changes, err := r.WorktreeChanges(wt)
	if err != nil {
		t.Fatalf("WorktreeChanges: %v", err)
	}

	if len(changes) != 0 {
		t.Errorf("a worktree nobody wrote in reports %v", changes)
	}
}

// TestNumstatSkipsALineItCannotRead. numstat's shape is git's to change, and
// a line read wrong is a file counted under a name nobody wrote.
func TestNumstatSkipsALineItCannotRead(t *testing.T) {
	got := numstat(strings.Join([]string{
		"3\t1\tinternal/task/run.go",
		"not a numstat line",
		"-\t-\tassets/logo.png",
		"",
	}, "\n"))

	if len(got) != 2 {
		t.Fatalf("numstat read %d changes, want 2: %v", len(got), got)
	}

	if got[0].Lines() != 4 {
		t.Errorf("the go file counts %d lines, want 4", got[0].Lines())
	}

	if !got[1].Binary() || got[1].Lines() != 0 {
		t.Errorf("the png reads as %+v, want a binary file worth no lines", got[1])
	}
}
