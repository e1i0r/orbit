package repo

// What a worktree was cut from, which is not what its repository is standing
// on now.

import (
	"strings"
	"testing"
)

// TestTheBaseIsTheOneTheWorktreeWasCutFrom. r.Base is wherever the
// repository's own checkout happens to be: a person who switches branches or
// detaches HEAD while a task runs was counting the task's change against a
// branch it never left, or against nothing at all — and an over-diff gate
// with nothing to compare against passes anything.
func TestTheBaseIsTheOneTheWorktreeWasCutFrom(t *testing.T) {
	r, wt := changed(t)

	if got := r.cutFrom(wt); got != "main" {
		t.Fatalf("the worktree says it was cut from %q, want main", got)
	}

	// The repository moves on, as a person working in it does.
	if _, err := git(r.Path, "checkout", "-b", "somewhere-else"); err != nil {
		t.Fatalf("checkout: %v", err)
	}

	moved := Repo{Path: r.Path, Name: r.Name, Base: "somewhere-else"}
	if got := moved.cutFrom(wt); got != "main" {
		t.Errorf("after the checkout moved, the worktree says %q, want main", got)
	}

	// And detached, where r.Base is empty and against() answered with
	// nothing at all.
	if _, err := git(r.Path, "checkout", "--detach"); err != nil {
		t.Fatalf("detach: %v", err)
	}

	detached := Repo{Path: r.Path, Name: r.Name}
	if got := detached.cutFrom(wt); got != "main" {
		t.Errorf("with the checkout detached, the worktree says %q, want main", got)
	}

	if args := detached.against(wt); len(args) != 2 || args[1] != "main" {
		t.Errorf("the diff is taken against %v, want a merge base with main", args)
	}
}

// TestAWorktreeFromBeforeThisFallsBackToTheRepository, which is what it was
// measured against for its whole life.
func TestAWorktreeFromBeforeThisFallsBackToTheRepository(t *testing.T) {
	r, wt := changed(t)

	branch, err := git(wt, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		t.Fatalf("branch: %v", err)
	}

	if _, err := git(r.Path, "config", "--unset", baseKey(strings.TrimSpace(branch))); err != nil {
		t.Fatalf("unset: %v", err)
	}

	if got := r.cutFrom(wt); got != r.Base {
		t.Errorf("a worktree with nothing written down says %q, want the repository's %q", got, r.Base)
	}
}
