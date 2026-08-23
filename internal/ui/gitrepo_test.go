package ui

// gitrepo_test.go is the one fixture in this package that is a real
// repository on disk, and it exists for one assertion: that the diff tab
// shows the task's worktree and not the reader's own checkout.
//
// It is written here rather than borrowed because internal/repo's makeRepo
// is unexported and lives in another package's test binary. Ten lines
// duplicated across a package boundary is the ordinary Go answer; exporting
// a test helper into production code to save them is the worse trade.
//
// Nothing here reaches the network, the real HOME, or $ORBIT_HOME. The
// remote is never added, both git config files are pointed at /dev/null, and
// TestMain has already moved HOME to a temporary directory.

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// gitRepo builds a repository with one commit on main and returns its path.
func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	write(t, filepath.Join(dir, "retry.go"), "package retry\n\nfunc send() {}\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "first")
	return dir
}

// worktreeOf cuts a branch of its own for one task, the way the store does.
func worktreeOf(t *testing.T, repoPath, id string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), id)
	git(t, repoPath, "worktree", "add", "-q", "-b", id, dir, "main")
	return dir
}

// git runs one git command in one directory, with the contributor's own
// configuration kept out of it. A global commit.gpgsign or core.hooksPath
// would otherwise decide whether this suite passes, and would run that
// contributor's hooks inside it.
func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestTheDiffIsTheWorktreesAndNotTheRepositorys is the defect this task was
// dispatched with, written down as a test.
//
// Before the diff tab existed, this command ran git in view.Task.RepoPath —
// the repository the reader has open in their own editor. Both directories
// have uncommitted changes here, they say different things, and the pane may
// only ever show one of them: an agent's work under the reader's own heading
// is a screen that lies quietly, which is the expensive kind.
func TestTheDiffIsTheWorktreesAndNotTheRepositorys(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	repoPath := gitRepo(t)
	tree := worktreeOf(t, repoPath, "ACME-2662")
	write(t, filepath.Join(repoPath, "retry.go"), "package retry\n\nfunc send() { theReadersOwnEdit() }\n")
	write(t, filepath.Join(tree, "retry.go"), "package retry\n\nfunc send() { backoff() }\n")

	r := &fakeReader{worktree: tree}
	msg, ok := diffOf(r, view.Task{ID: "ACME-2662", RepoPath: repoPath}, baseRef{})().(diffMsg)
	if !ok {
		t.Fatal("diffOf did not answer with a diff")
	}
	if msg.Err != nil {
		t.Fatalf("diff the worktree: %v", msg.Err)
	}
	if !strings.Contains(msg.Text, "backoff()") {
		t.Errorf("the diff says:\n%s\nwant the change made in the worktree", msg.Text)
	}
	if strings.Contains(msg.Text, "theReadersOwnEdit") {
		t.Errorf("the diff says:\n%s\nwant the reader's own checkout left out of it", msg.Text)
	}
	if msg.Tree != tree {
		t.Errorf("the diff came back from %q, want %q", msg.Tree, tree)
	}
	// The two answers that ride back with the text, against a repository
	// where both are knowable: this one is on main, so the comparison had a
	// base and the flag that says otherwise must not be set. Both tests of
	// the flag until now built a diffMsg by hand, which says nothing about
	// the code that computes it — a gitDiff that always answered true, or
	// always false, passed them.
	if msg.NoBase {
		t.Error("the diff says it had no base to measure against, in a repository whose branch is main")
	}
	// And the base it used comes back known, because that is what stops the
	// next rescan two seconds from now from looking it up again.
	if !msg.Base.known || msg.Base.name != "main" || msg.Base.timedOut {
		t.Errorf("the diff came back with base %+v, want main, looked up and answered", msg.Base)
	}
}

// TestNoBaseIsSaidOnlyWhenGitActuallySaidIt is the other direction of the
// same flag, and the line between the two facts it used to conflate.
//
// A repository with nothing to measure against really has no base, and the
// strip says so. A base lookup that ran out of time also comes back empty,
// and the strip must not say so about it: the diff on screen is the same
// plain working tree either way and is true either way, but one of the two
// is a fact about the repository and the other is a fact about how long this
// program was prepared to wait.
func TestNoBaseIsSaidOnlyWhenGitActuallySaidIt(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	repoPath := gitRepo(t)
	tree := worktreeOf(t, repoPath, "ACME-2662")
	write(t, filepath.Join(tree, "retry.go"), "package retry\n\nfunc send() { backoff() }\n")
	task := view.Task{ID: "ACME-2662", RepoPath: repoPath}

	absent := diffOf(&fakeReader{worktree: tree}, task, baseRef{known: true})().(diffMsg)
	if absent.Err != nil || !absent.NoBase {
		t.Errorf("a diff with no base to measure against said NoBase=%v (err %v), want it said plainly",
			absent.NoBase, absent.Err)
	}
	silent := diffOf(&fakeReader{worktree: tree}, task, baseRef{known: true, timedOut: true})().(diffMsg)
	if silent.Err != nil || silent.NoBase {
		t.Errorf("a base that timed out was reported as no base at all (err %v), want the claim withheld", silent.Err)
	}
	if !strings.Contains(silent.Text, "backoff()") {
		t.Errorf("the fallback diff says:\n%s\nwant the worktree's own change in it", silent.Text)
	}
}

// TestABaseAlreadyKnownIsNotLookedUpAgain is the once-per-open rule, proved
// by the only means a test has of seeing a subprocess that did not run.
//
// The task is pointed at a directory that is not a repository at all, so a
// lookup would come back with nothing and the diff would fall back to the
// plain working tree and say so. The base handed in is main, which the
// worktree really does have, so a diff that used it compares against it and
// says nothing about a fallback. The flag tells the two apart, and with it
// the difference between asking git three times every two seconds and
// asking it three times when the reader opens the view.
func TestABaseAlreadyKnownIsNotLookedUpAgain(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	tree := worktreeOf(t, gitRepo(t), "ACME-2662")
	write(t, filepath.Join(tree, "retry.go"), "package retry\n\nfunc send() { backoff() }\n")

	task := view.Task{ID: "ACME-2662", RepoPath: t.TempDir()}
	msg := diffOf(&fakeReader{worktree: tree}, task, baseRef{name: "main", known: true})().(diffMsg)
	if msg.Err != nil {
		t.Fatalf("diff against a base handed in: %v", msg.Err)
	}
	if msg.NoBase {
		t.Error("the diff fell back to the plain working tree, so it looked the base up again instead of using the one it was given")
	}
}

// TestADiffWithoutAWorktreeSaysSo covers the other end: a task whose
// worktree the port cannot find is a sentence in the pane, never a blank one
// that reads as "no changes".
func TestADiffWithoutAWorktreeSaysSo(t *testing.T) {
	r := &fakeReader{treeErr: os.ErrNotExist}
	msg, ok := diffOf(r, view.Task{ID: "ACME-2662", RepoPath: "/nowhere"}, baseRef{})().(diffMsg)
	if !ok {
		t.Fatal("diffOf did not answer with a diff")
	}
	if msg.Err == nil {
		t.Fatal("a worktree that could not be found came back as a diff")
	}
}

// fakeGit puts a script standing in for git first on PATH: whatever it is
// asked, it does the one thing this suite needs a hung git to do, which is
// not answer. A lock left over from a crashed process, a filesystem that
// stopped responding, a pager misconfigured into waiting — all of them look
// like this from the caller's side, and none of them is worth reproducing
// to prove the timeout that has to catch all three.
func fakeGit(t *testing.T, seconds int) {
	t.Helper()
	bin := t.TempDir()
	script := filepath.Join(bin, "git")
	write(t, script, fmt.Sprintf("#!/bin/sh\nsleep %d\n", seconds))
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatalf("chmod fake git: %v", err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
}

// TestAHungGitTimesOutRatherThanHangingForever is I2's second requirement:
// runGitDiff's own bound has to actually kill a git that does not answer,
// and say that is what happened rather than repeat exec's own words for a
// process that, as far as this program can prove, is still running.
//
// gitDiffTimeout is shrunk for the run and put back after, so the test
// proves the bound is hit without the suite waiting out the real one.
func TestAHungGitTimesOutRatherThanHangingForever(t *testing.T) {
	old := gitDiffTimeout
	gitDiffTimeout = 50 * time.Millisecond
	t.Cleanup(func() { gitDiffTimeout = old })
	fakeGit(t, 5)

	_, err := runGitDiff(t.TempDir(), "diff")
	if !errors.Is(err, errGitTimedOut) {
		t.Fatalf("a hung git came back as %v, want the timeout error", err)
	}
}

// TestABaseThatDoesNotAnswerFallsBackWithoutHanging is boundedBaseOf's own
// half of the same requirement: what it is bounding is internal/repo's git,
// which takes no context and cannot be killed, so what has to be proven is
// the wait giving up on time — not the process, which this test leaves
// running in the background the way production code already discloses it
// will.
func TestABaseThatDoesNotAnswerFallsBackWithoutHanging(t *testing.T) {
	old := baseTimeout
	baseTimeout = 20 * time.Millisecond
	t.Cleanup(func() { baseTimeout = old })
	fakeGit(t, 5)

	start := time.Now()
	base := boundedBaseOf(t.TempDir())
	if base.name != "" || !base.timedOut {
		t.Errorf("boundedBaseOf against a hung git answered %+v, want empty and marked as having given up", base)
	}
	if !base.known {
		t.Error("boundedBaseOf came back unknown, so the next tick would ask a hung git all over again")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("boundedBaseOf took %v to give up, want it bounded near baseTimeout", elapsed)
	}
}
