package repo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeGh puts a shell script named gh at the front of PATH, so that the
// delivery functions can be run against a GitHub CLI whose output is known.
//
// It is a fake and not a mock of an interface, because what is being tested
// is exactly the thing an interface would hide: which of the two streams the
// answer was taken from.
func fakeGh(t *testing.T, script string) {
	t.Helper()

	dir := t.TempDir()

	path := filepath.Join(dir, "gh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o600); err != nil {
		t.Fatalf("write the fake gh: %v", err)
	}

	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatalf("make the fake gh runnable: %v", err)
	}

	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestTheAnswerIsStdoutAndNothingElse. Every one of these commands is run for
// what it prints, and gh prints a notice to stderr whenever a newer version
// of itself exists. Folding the two together puts that notice inside the
// answer, which for CreatePR is inside a URL.
func TestTheAnswerIsStdoutAndNothingElse(t *testing.T) {
	out, err := run(t.TempDir(), "sh", "-c", "echo 'a new release of gh is available' >&2; echo https://example.test/pr/1")
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if out != "https://example.test/pr/1" {
		t.Errorf("the answer is %q, want the URL on stdout on its own", out)
	}
}

// TestWhatFailedSaysWhatWasOnStderr. The other half of the same rule: a
// message printed while failing is the only account of what went wrong, so it
// belongs in the error even though it is kept out of the answer.
func TestWhatFailedSaysWhatWasOnStderr(t *testing.T) {
	_, err := run(t.TempDir(), "sh", "-c", "echo 'no such pull request' >&2; exit 1")
	if err == nil {
		t.Fatal("a command that exited 1 came back without an error")
	}

	if !strings.Contains(err.Error(), "no such pull request") {
		t.Errorf("the error reads %q, want it to carry what stderr said", err)
	}
}

// TestGitIsNotSentSomewhereElseByTheEnvironment. cmd.Dir is not the whole
// answer to which repository git is talking to: GIT_DIR is read first. A
// process started from a git hook has one exported, and every command in
// this package would then run against a repository nobody named — the
// worktree left untouched, another repository committed to.
func TestGitIsNotSentSomewhereElseByTheEnvironment(t *testing.T) {
	here, elsewhere := t.TempDir(), t.TempDir()
	for _, dir := range []string{here, elsewhere} {
		if _, err := git(dir, "init"); err != nil {
			t.Fatalf("git init in %q: %v", dir, err)
		}
	}

	t.Setenv("GIT_DIR", filepath.Join(elsewhere, ".git"))
	t.Setenv("GIT_WORK_TREE", elsewhere)

	top, err := git(here, "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatalf("git rev-parse: %v", err)
	}

	want, err := filepath.EvalSymlinks(here)
	if err != nil {
		t.Fatalf("resolve %q: %v", here, err)
	}

	if top != want {
		t.Errorf("git answered for %q, want the directory it was given, %q", top, want)
	}
}

// TestGitIsNotLeftAskingATerminalNobodyIsAt. git does not read a password
// from stdin; it opens the terminal directly. Under the window that is a
// question nobody sees, waiting for an answer nobody can type, until the
// deadline ten minutes later.
func TestGitIsNotLeftAskingATerminalNobodyIsAt(t *testing.T) {
	out, err := run(t.TempDir(), "sh", "-c", `printf %s "$GIT_TERMINAL_PROMPT"`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if out != "0" {
		t.Errorf("GIT_TERMINAL_PROMPT is %q, want 0 so a missing credential fails instead of waiting", out)
	}
}

// TestACommandThatNeverFinishesIsGivenUpOn. A fetch against a host that
// stopped answering has no end of its own, and the task it belongs to says
// "delivering" for as long as the window is open.
func TestACommandThatNeverFinishesIsGivenUpOn(t *testing.T) {
	start := time.Now()

	_, err := runWithin(50*time.Millisecond, t.TempDir(), "sleep", "5")
	if err == nil {
		t.Fatal("a command that never finished came back without an error")
	}

	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("the wait lasted %s, want it ended at the deadline", took)
	}

	// "signal: killed" is what os/exec says, and it is a sentence about a
	// signal rather than about a command that did not finish.
	if !strings.Contains(err.Error(), "gave up waiting") {
		t.Errorf("the error reads %q, want it to say the wait was given up on", err)
	}
}

// TestAPullRequestUrlIsWhatGhPrintedOnStdout. gh's own chatter must not come
// back as the URL: the caller prints what it is given as "Pull Request
// created: %s".
func TestAPullRequestUrlIsWhatGhPrintedOnStdout(t *testing.T) {
	fakeGh(t, `echo 'A new release of gh is available: 2.40.0' >&2
echo https://github.com/acme/app/pull/7`)

	url, err := Repo{}.CreatePR(t.TempDir(), "title", "body", "orbit/TEST-1", "main")
	if err != nil {
		t.Fatalf("CreatePR: %v", err)
	}

	if url != "https://github.com/acme/app/pull/7" {
		t.Errorf("the pull request URL is %q, want the link with nothing in front of it", url)
	}
}

// TestAPullRequestThatIsAlreadyOpenIsLookedUp. Delivering a task twice is
// ordinary, and gh answers it by failing with a sentence that says a pull
// request exists and does not say where.
func TestAPullRequestThatIsAlreadyOpenIsLookedUp(t *testing.T) {
	fakeGh(t, `case "$2" in
view) echo https://github.com/acme/app/pull/7 ;;
*) echo 'a pull request for branch "orbit/TEST-1" already exists' >&2; exit 1 ;;
esac`)

	url, err := Repo{}.CreatePR(t.TempDir(), "title", "body", "orbit/TEST-1", "main")
	if err != nil {
		t.Fatalf("CreatePR: %v", err)
	}

	if url != "https://github.com/acme/app/pull/7" {
		t.Errorf("the pull request URL is %q, want the one gh pr view answered", url)
	}
}

// TestClosingLeavesTheCommentTheCallerGave. The sentence belongs to the
// caller: this package runs git and gh and writes no English of its own.
func TestClosingLeavesTheCommentTheCallerGave(t *testing.T) {
	dir := t.TempDir()
	argv := filepath.Join(dir, "argv")

	fakeGh(t, `printf '%s\n' "$@" > `+argv)

	if err := (Repo{}).ClosePR(dir, "orbit/TEST-1", "Closed from Orbit."); err != nil {
		t.Fatalf("ClosePR: %v", err)
	}

	got, err := os.ReadFile(argv)
	if err != nil {
		t.Fatalf("read what gh was given: %v", err)
	}

	if !strings.Contains(string(got), "--comment\nClosed from Orbit.\n") {
		t.Errorf("gh was given %q, want the caller's comment", got)
	}

	if err := (Repo{}).ClosePR(dir, "orbit/TEST-1", ""); err != nil {
		t.Fatalf("ClosePR with no comment: %v", err)
	}

	got, err = os.ReadFile(argv)
	if err != nil {
		t.Fatalf("read what gh was given: %v", err)
	}

	if strings.Contains(string(got), "--comment") {
		t.Errorf("gh was given %q, want no comment at all", got)
	}
}

// TestMergingAsksForASquashAndNoBranchLeftBehind. MergePR answers an error
// and nothing else, so the whole of what it does is the line it hands gh:
// one commit on the base branch, and the branch behind it gone. A flag
// dropped from that line is a delivery that merges some other way and says
// nothing about having done so.
func TestMergingAsksForASquashAndNoBranchLeftBehind(t *testing.T) {
	dir := t.TempDir()
	argv := filepath.Join(dir, "argv")

	fakeGh(t, `printf '%s\n' "$@" > `+argv)

	if err := (Repo{}).MergePR(dir, "orbit/TEST-1"); err != nil {
		t.Fatalf("MergePR: %v", err)
	}

	got, err := os.ReadFile(argv)
	if err != nil {
		t.Fatalf("read what gh was given: %v", err)
	}

	if want := "pr\nmerge\norbit/TEST-1\n--squash\n--delete-branch\n"; string(got) != want {
		t.Errorf("gh was given %q, want %q", got, want)
	}
}

// TestAMergeGhRefusedIsAnError. gh's confirmation goes to stderr and is not
// an answer, so nothing but the error is handed back — which makes the error
// the only thing that can say a merge did not happen.
func TestAMergeGhRefusedIsAnError(t *testing.T) {
	fakeGh(t, `echo 'pull request is not mergeable' >&2; exit 1`)

	err := (Repo{}).MergePR(t.TempDir(), "orbit/TEST-1")
	if err == nil {
		t.Fatal("a gh that refused was reported as a merge")
	}

	if !strings.Contains(err.Error(), "not mergeable") {
		t.Errorf("the error is %q, and does not carry gh's reason", err)
	}
}
