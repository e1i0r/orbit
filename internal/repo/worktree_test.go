package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddWorktreeCheckoutsANewBranch(t *testing.T) {
	// AddWorktree below runs `git worktree add`, which checks the new
	// worktree out and so can run a contributor's own post-checkout hook;
	// this test also calls git() itself to read the branch back. Both need
	// the developer's real git config kept out of the test process.
	isolateGitConfig(t)

	dir := filepath.Join(t.TempDir(), "src")
	if err := mkdir(dir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	makeRepo(t, dir, "main", "origin")

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	wt := filepath.Join(t.TempDir(), "wt")
	if err := r.AddWorktree(wt, "orbit/ACME-1"); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}

	if _, err := os.Stat(filepath.Join(wt, ".git")); err != nil {
		t.Fatalf("worktree has no .git: %v", err)
	}

	branch, err := git(wt, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		t.Fatalf("read branch: %v", err)
	}

	if branch != "orbit/ACME-1" {
		t.Errorf("branch = %q, want orbit/ACME-1", branch)
	}
}

func TestAddWorktreeLeavesTheBaseBranchAlone(t *testing.T) {
	isolateGitConfig(t)

	dir := filepath.Join(t.TempDir(), "src")
	if err := mkdir(dir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	makeRepo(t, dir, "main", "origin")

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "orbit/ACME-1"); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}

	branch, err := git(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		t.Fatalf("read branch: %v", err)
	}

	if branch != "main" {
		t.Errorf("the source checkout moved to %q — a task must never touch the base branch", branch)
	}
}

func TestRemoveWorktreeCleansUpBothSides(t *testing.T) {
	isolateGitConfig(t)

	dir := filepath.Join(t.TempDir(), "src")
	if err := mkdir(dir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	makeRepo(t, dir, "main", "origin")

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	wt := filepath.Join(t.TempDir(), "wt")
	if err := r.AddWorktree(wt, "orbit/ACME-1"); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}

	if err := r.RemoveWorktree(wt); err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}

	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("the worktree directory survived: %v", err)
	}

	list, err := git(dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list: %v", err)
	}

	if len(splitLines(list)) != 1 {
		t.Errorf("git still lists %d worktrees:\n%s", len(splitLines(list)), list)
	}
}

func TestAddWorktreeReusesABranchThatOutlivedItsWorktree(t *testing.T) {
	isolateGitConfig(t)

	dir := filepath.Join(t.TempDir(), "src")
	if err := mkdir(dir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	makeRepo(t, dir, "main", "origin")

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	first := filepath.Join(t.TempDir(), "wt")
	if err := r.AddWorktree(first, "orbit/ACME-1"); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}
	// The worktree goes; the branch stays, as it does after every removal.
	if err := r.RemoveWorktree(first); err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}

	second := filepath.Join(t.TempDir(), "wt2")
	if err := r.AddWorktree(second, "orbit/ACME-1"); err != nil {
		t.Fatalf("a re-run of the same task could not have a worktree: %v", err)
	}

	branch, err := git(second, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		t.Fatalf("read branch: %v", err)
	}

	if branch != "orbit/ACME-1" {
		t.Errorf("branch = %q, want the task's own branch orbit/ACME-1", branch)
	}
}

func TestAddWorktreeRecoversFromADirectoryDeletedByHand(t *testing.T) {
	// This test never calls git() itself, but AddWorktree below does, and
	// a worktree checkout can run a contributor's own post-checkout hook
	// just as readily whether the test that triggered it reads git's
	// output back or not.
	isolateGitConfig(t)

	dir := filepath.Join(t.TempDir(), "src")
	if err := mkdir(dir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	makeRepo(t, dir, "main", "origin")

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	wt := filepath.Join(t.TempDir(), "wt")
	if err := r.AddWorktree(wt, "orbit/ACME-1"); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}
	// A person deletes the checkout. git keeps its own note about it, and
	// the branch is still there. This is an ordinary thing to do.
	if err := os.RemoveAll(wt); err != nil {
		t.Fatalf("remove the worktree by hand: %v", err)
	}

	if err := r.AddWorktree(wt, "orbit/ACME-1"); err != nil {
		t.Fatalf("the task could never be run again: %v", err)
	}

	if _, err := os.Stat(filepath.Join(wt, ".git")); err != nil {
		t.Errorf("the worktree was not checked out again: %v", err)
	}
}

func TestAddWorktreeSaysBothWhatFailedAndWhere(t *testing.T) {
	isolateGitConfig(t)

	dir := filepath.Join(t.TempDir(), "src")
	if err := mkdir(dir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	makeRepo(t, dir, "main", "origin")

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	r.Base = "no-such-branch"
	wt := filepath.Join(t.TempDir(), "wt")

	err = r.AddWorktree(wt, "orbit/ACME-1")
	if err == nil {
		t.Fatal("AddWorktree succeeded from a base branch that does not exist")
	}
	// Both facts, in Orbit's own words. The git command line quoted
	// further along happens to contain the directory too, so asserting on
	// the whole string would pass without the sentence saying anything.
	want := fmt.Sprintf("create a worktree for %q at %q", "orbit/ACME-1", wt)
	if !strings.Contains(err.Error(), want) {
		t.Errorf("the error reads %v\n  want it to open with %s", err, want)
	}
}

func TestAddWorktreeRefusesADetachedRepository(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "src")
	if err := mkdir(dir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	makeRepo(t, dir, "main", "origin")

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	r.Base = ""

	err = r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "orbit/ACME-1")
	if err == nil {
		t.Fatal("AddWorktree tried to cut a worktree from a branch that does not exist")
	}

	if !strings.Contains(err.Error(), "not on a branch") {
		t.Errorf("the error does not say what is wrong: %v", err)
	}
}
