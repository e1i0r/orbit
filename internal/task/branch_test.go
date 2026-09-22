package task

// A task's branch: its own, recorded, and gone when the task is.

import (
	"os/exec"
	"strings"
	"testing"
)

// TestEachTaskGetsABranchNobodyElseHas.
func TestEachTaskGetsABranchNobodyElseHas(t *testing.T) {
	first, second := newBranch("FRA-128"), newBranch("FRA-128")

	if first == second {
		t.Errorf("two tasks written under FRA-128 got the same branch %q; the second would "+
			"check out the first one's commits", first)
	}

	for _, name := range []string{first, second} {
		if !strings.HasPrefix(name, "orbit/FRA-128-") {
			t.Errorf("branch %q does not read as FRA-128's; a reader listing branches has to "+
				"be able to tell whose it is", name)
		}
	}
}

// TestATaskWrittenBeforeSuffixesKeepsItsBranch.
//
// Those tasks have branches pushed and pull requests open against them. A
// new name would orphan both, so an empty BranchName answers what Orbit
// has always answered.
func TestATaskWrittenBeforeSuffixesKeepsItsBranch(t *testing.T) {
	if got := Branch(Task{ID: "FRA-9"}); got != "orbit/FRA-9" {
		t.Errorf("a task with nothing recorded is on %q, want orbit/FRA-9", got)
	}

	if got := Branch(Task{ID: "FRA-9", BranchName: "orbit/FRA-9-ab12"}); got != "orbit/FRA-9-ab12" {
		t.Errorf("a task that recorded its branch is on %q, want the recorded one", got)
	}
}

// TestTheBranchSurvivesARoundTripThroughTheRecord: every attempt at one
// task has to agree about it, or a resume starts a second branch and
// loses the first one's work.
func TestTheBranchSurvivesARoundTripThroughTheRecord(t *testing.T) {
	s, r := fixture(t)

	made, err := Create(s, r, "BR-1", "a task", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if made.BranchName == "" {
		t.Fatal("Create wrote no branch name down")
	}

	read, err := Load(s, r, "BR-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if Branch(read) != Branch(made) {
		t.Errorf("Load reads the branch as %q and Create made it %q", Branch(read), Branch(made))
	}
}

// TestWritingAnIDAgainGetsADifferentBranch is the whole point: FRA-128 was
// deleted and written again, and the new task woke up on the old one's
// three commits.
func TestWritingAnIDAgainGetsADifferentBranch(t *testing.T) {
	s, r := fixture(t)

	first, err := Create(s, r, "BR-2", "first go", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Delete(s, first); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	second, err := Create(s, r, "BR-2", "second go", "quick")
	if err != nil {
		t.Fatalf("Create again: %v", err)
	}

	if Branch(second) == Branch(first) {
		t.Errorf("the task written again is on %q, the same branch the deleted one used",
			Branch(second))
	}
}

// TestDeletingATaskTakesItsBranchWithIt, against real git.
//
// "Delete the task" meant the record and the worktree, and the branch was
// left on disk with every commit the run made on it. Elio, after the third
// time: if I delete a task, I want to delete all — worktree, branch and
// task.
func TestDeletingATaskTakesItsBranchWithIt(t *testing.T) {
	s, r := fixture(t)

	made, err := Create(s, r, "BR-3", "a task with work on it", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Give it a worktree, which is what a run does first.
	if _, err := Join(s, made, r); err != nil {
		t.Fatalf("Join: %v", err)
	}

	if !branchIsThere(t, r.Path, Branch(made)) {
		t.Fatalf("the worktree did not leave a branch %q to delete", Branch(made))
	}

	if err := Delete(s, made); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if branchIsThere(t, r.Path, Branch(made)) {
		t.Errorf("the branch %q outlived the task it was made for", Branch(made))
	}
}

// branchIsThere asks git, rather than asking Orbit whether Orbit did what
// it said. The claim under test is about the repository on disk.
func branchIsThere(t *testing.T, path, branch string) bool {
	t.Helper()

	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Dir = path

	return cmd.Run() == nil
}
