package cli

// Everything that can go wrong before gh is reached.
//
// A delivery is six steps and gh is the last of them. The five in front —
// the flags, the repository, the task, the base branch, the push — each have
// a way of failing, and what a reader is owed at each is a refusal that says
// which step it was and stops there.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheGhCommandsStopBeforeTheyReachGh. The three of them open the same
// things in the same order, so they fail the same way, and none of these
// refusals is allowed to be gh's — a command that ran gh against a
// repository it could not open would be asking GitHub about nothing.
func TestTheGhCommandsStopBeforeTheyReachGh(t *testing.T) {
	for _, tc := range []struct {
		name string
		args func(t *testing.T, dir string) []string
		says string
	}{
		{"a flag nobody defined", func(_ *testing.T, dir string) []string {
			return []string{"-nope", "-repo", dir, "PAY-1"}
		}, "not defined: -nope"},
		{"a directory that is not a repository", func(t *testing.T, _ string) []string {
			return []string{"-repo", t.TempDir(), "PAY-1"}
		}, "is not a repository"},
		{"a task identifier that cannot be a path", func(_ *testing.T, dir string) []string {
			return []string{"-repo", dir, "PAY 1\x00"}
		}, "control character"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := deliverable(t, "make the thing")
			argv := recordArgv(t, "echo https://github.test/pr/1")

			for _, command := range []string{"pr", "merge", "close-pr"} {
				code, _, errOut := run(t, append([]string{command}, tc.args(t, dir)...)...)
				if code == 0 {
					t.Errorf("%s exited 0 on %s", command, tc.name)
				}

				if !strings.Contains(errOut, tc.says) {
					t.Errorf("%s refused with %q, which does not say %q", command, errOut, tc.says)
				}
			}

			if _, err := os.Stat(argv); err == nil {
				t.Error("gh was run anyway, against a delivery that had already failed")
			}
		})
	}
}

// TestDeliveringATaskNobodyWroteSaysSo. The task is read for the text that
// becomes the title and the body of the pull request, so it is the one thing
// pr needs that merge and close-pr do not.
func TestDeliveringATaskNobodyWroteSaysSo(t *testing.T) {
	dir := deliverable(t, "make the thing")
	argv := recordArgv(t, "echo https://github.test/pr/1")

	code, _, errOut := run(t, "pr", "-repo", dir, "PAY-404")
	if code == 0 {
		t.Fatal("pr exited 0 for a task that was never written")
	}

	if !strings.Contains(errOut, "PAY-404") {
		t.Errorf("the refusal does not name the task asked for: %q", errOut)
	}

	if _, err := os.Stat(argv); err == nil {
		t.Error("gh was asked to open a pull request for a task that does not exist")
	}
}

// TestABaseBranchThatCannotBeFetchedDoesNotStopTheDelivery. Syncing the base
// branch in is a convenience: it saves a delivery from opening a pull
// request that conflicts on arrival. A remote that cannot answer for it is
// worth a line in the log and nothing more, because the work is committed
// and pushable either way — and stopping here would mean a base branch
// somebody deleted could block every delivery in the repository.
func TestABaseBranchThatCannotBeFetchedDoesNotStopTheDelivery(t *testing.T) {
	dir := deliverable(t, "make the thing")

	// The remote is still there to push to; what is gone is the branch the
	// sync would have fetched from it.
	inside(t, remoteOf(t, dir), "update-ref", "-d", "refs/heads/main")

	fakeGh(t, "echo https://github.test/pr/9")

	code, out, errOut := run(t, "pr", "-repo", dir, "PAY-1")
	if code != 0 {
		t.Fatalf("pr exited %d because the base branch could not be fetched: %s", code, errOut)
	}

	if !strings.Contains(out, "https://github.test/pr/9") {
		t.Errorf("the delivery went through and the reader was not told where: %q", out)
	}
}

// TestAWorktreeThatIsNotThereIsRefusedBeforeAnythingIsPushed. The checkout a
// delivery commits from is written by the run that did the work, and a task
// delivered from a machine that never ran it has no such directory.
func TestAWorktreeThatIsNotThereIsRefusedBeforeAnythingIsPushed(t *testing.T) {
	dir := deliverable(t, "make the thing")
	argv := recordArgv(t, "echo https://github.test/pr/1")

	if err := os.RemoveAll(worktreeOf(t, dir)); err != nil {
		t.Fatalf("take the worktree away: %v", err)
	}

	code, _, errOut := run(t, "pr", "-repo", dir, "PAY-1")
	if code == 0 {
		t.Fatal("pr exited 0 with no worktree to commit from")
	}

	if strings.TrimSpace(errOut) == "" {
		t.Error("pr refused without saying anything")
	}

	if _, err := os.Stat(argv); err == nil {
		t.Error("gh was asked to open a pull request for a branch with nothing on it")
	}
}

// TestACommitThatCannotBeMadeIsRefusedBeforeAnythingIsPushed. A worktree
// that is there is not the same as a worktree that can be committed to: an
// index.lock left behind by a git that was killed stops the staging while
// everything else still works.
//
// Which is what makes this the case worth writing. The branch pushes either
// way — it has existed on the remote since the worktree was made — so a
// delivery that carried on past the commit would open a pull request with
// none of the work in it, and say nothing about having done so.
func TestACommitThatCannotBeMadeIsRefusedBeforeAnythingIsPushed(t *testing.T) {
	dir := deliverable(t, "make the thing")
	argv := recordArgv(t, "echo https://github.test/pr/1")

	lock := filepath.Join(gitOut(t, worktreeOf(t, dir), "rev-parse", "--absolute-git-dir"), "index.lock")
	if err := os.WriteFile(lock, nil, 0o600); err != nil {
		t.Fatalf("leave a lock behind: %v", err)
	}

	code, _, errOut := run(t, "pr", "-repo", dir, "PAY-1")
	if code == 0 {
		t.Fatal("pr exited 0 with an index it could not write")
	}

	if strings.TrimSpace(errOut) == "" {
		t.Error("pr refused without saying anything")
	}

	if _, err := os.Stat(argv); err == nil {
		t.Error("gh was asked for a pull request with none of the work in it")
	}
}

// TestNoPullRequestIsOpenedForABranchThatWasNotPushed. The order is the
// point: gh is asked for a pull request on a branch that exists on the
// remote, and a push that failed means it does not. Running gh anyway opens
// a pull request against a branch nobody can see, or fails a second time and
// buries the reason the first one failed.
func TestNoPullRequestIsOpenedForABranchThatWasNotPushed(t *testing.T) {
	dir := deliverable(t, "make the thing")
	argv := recordArgv(t, "echo https://github.test/pr/1")

	inside(t, dir, "remote", "set-url", "acme", filepath.Join(t.TempDir(), "gone"))

	code, _, errOut := run(t, "pr", "-repo", dir, "PAY-1")
	if code == 0 {
		t.Fatal("pr exited 0 with a remote that is not there")
	}

	if !strings.Contains(errOut, "acme") {
		t.Errorf("the refusal does not name the remote it could not reach: %q", errOut)
	}

	if _, err := os.Stat(argv); err == nil {
		t.Error("gh was asked for a pull request on a branch that was never pushed")
	}
}
