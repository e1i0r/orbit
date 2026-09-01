package task

// Taking a task off the board without unwriting what it did.

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// TestDeleteTakesTheTaskOffTheListingAndKeepsWhatItDid is the soft delete in
// one pair of assertions. List is the enumeration the board, `orbit list`
// and the reconcile sweep all read, so a task that is out of it is out of
// all three; Events is the account of what an engine was asked and what it
// spent, and tidying a board does not ask for that to be destroyed.
func TestDeleteTakesTheTaskOffTheListingAndKeepsWhatItDid(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DEL-1", "tidy this one away", "quick")
	if err != nil {
		t.Fatal(err)
	}

	if err := Delete(s, tk); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	ids, err := List(s, r)
	if err != nil {
		t.Fatalf("list the tasks of %s: %v", r.Name, err)
	}

	if slices.Contains(ids, "DEL-1") {
		t.Errorf("the deleted task is still listed: %v", ids)
	}

	last := lastEvent(t, s, tk)
	if last.Kind != "task.deleted" {
		t.Errorf("the record ends with %q, want task.deleted", last.Kind)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("read the record of the deleted task: %v", err)
	}

	if len(events) < 2 {
		t.Errorf("the deleted task has %d events, want the delete and the create it was written with", len(events))
	}
}

// TestDeleteGivesTheCheckoutBackToGit. The worktree is the half that goes
// for real, and it goes through git: a checkout removed by hand leaves an
// entry under .git/worktrees that only `git worktree prune` clears, in a
// repository Orbit does not own.
func TestDeleteGivesTheCheckoutBackToGit(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DEL-2", "this one ran", "quick")
	if err != nil {
		t.Fatal(err)
	}

	wtDir, err := s.WorktreeDir(r.Path, tk.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := r.AddWorktree(wtDir, "orbit/DEL-2"); err != nil {
		t.Fatalf("add a worktree: %v", err)
	}

	if err := Delete(s, tk); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := os.Stat(wtDir); err == nil {
		t.Errorf("the checkout at %q is still there", wtDir)
	}

	if listed := worktrees(t, r.Path); strings.Contains(listed, wtDir) {
		t.Errorf("git still lists the worktree of the deleted task:\n%s", listed)
	}
}

// TestDeletingATaskWithNoCheckoutIsNotAFailure. A task that was never
// started has no worktree, and git refuses to remove one that is missing:
// the row still has to leave the board.
func TestDeletingATaskWithNoCheckoutIsNotAFailure(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "DEL-3", "never started", "quick")
	if err != nil {
		t.Fatal(err)
	}

	if err := Delete(s, tk); err != nil {
		t.Fatalf("Delete on a task that never ran: %v", err)
	}

	if last := lastEvent(t, s, tk); last.Kind != "task.deleted" {
		t.Errorf("the record ends with %q, want task.deleted", last.Kind)
	}
}

// TestADeleteThatCouldNotHappenSaysSo. An id the store refuses fails on both
// halves — no directory to name, no record to write to — and the two come
// back joined rather than the first one silencing the second.
func TestADeleteThatCouldNotHappenSaysSo(t *testing.T) {
	s, r := fixture(t)

	err := Delete(s, Task{ID: "../etc", Repo: r})
	if err == nil {
		t.Fatal("a delete the store refused came back as a success")
	}

	if !strings.Contains(err.Error(), "../etc") {
		t.Errorf("the failure says %q, and the reader has to know which task it was about", err)
	}
}

// worktrees is what git says the repository has, which is the half of a
// worktree that lives inside the repository rather than in the state root.
func worktrees(t *testing.T, repoPath string) string {
	t.Helper()

	cmd := exec.Command("git", "-C", repoPath, "worktree", "list")
	cmd.Env = append(cmd.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git worktree list in %q: %v\n%s", repoPath, err, out)
	}

	return string(out)
}
