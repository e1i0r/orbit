package cli

// Delivering a task, run against a gh that is a shell script.
//
// These three commands were the least covered in the package for the reason
// they are the hardest to test: each one ends in a call to a GitHub CLI that
// is not there, against a repository that does not exist, and everything
// they do before that — the sync, the commit, the push, the title — was
// reached by no test at all. A fake gh on $PATH and a bare repository on
// this disk make the whole path real except for GitHub itself.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// fakeGh puts a shell script named gh at the front of $PATH.
//
// A fake and not an interface, for the same reason internal/repo uses one:
// what is being tested is which of gh's two streams the answer was taken
// from, and an interface is exactly the thing that would hide it.
func fakeGh(t *testing.T, script string) {
	t.Helper()

	dir := t.TempDir()

	path := filepath.Join(dir, "gh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o700); err != nil {
		t.Fatalf("write the fake gh: %v", err)
	}

	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// deliverable is a workspace a task can actually be delivered from: a
// repository whose remote is a bare one on this disk, so the push that comes
// before gh is a real push, and a worktree with work in it waiting to be
// committed.
func deliverable(t *testing.T, text string) (dir string) {
	t.Helper()

	// The machine's own git configuration decides who a commit is by and
	// whether it is signed. Neither is an answer this test can afford to
	// take from whichever machine happens to run it.
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("ORBIT_HOME", filepath.Join(t.TempDir(), "orbit"))

	dir = withRemote(t, t.TempDir(), "payments")

	if code, _, errOut := run(t, "new", "-repo", dir, "-id", "PAY-1", text); code != 0 {
		t.Fatalf("orbit new exited %d: %s", code, errOut)
	}

	plantWorktree(t, dir, text)

	return dir
}

// withRemote is a repository at root/name whose remote is a bare repository
// on this disk, so a delivery from it makes a real push and stops only at
// GitHub.
func withRemote(t *testing.T, root, name string) string {
	t.Helper()

	remote := t.TempDir()
	inside(t, remote, "init", "--bare", "-q", "-b", "main")

	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("make the repository directory: %v", err)
	}

	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.name", "t"},
		{"config", "user.email", "t@t"},
		{"commit", "-q", "--allow-empty", "-m", "first"},
		{"remote", "add", "acme", remote},
		{"push", "-q", "acme", "HEAD:main"},
	} {
		inside(t, dir, args...)
	}

	return dir
}

// plantWorktree checks the task branch out where the delivery will look for
// it, and leaves a change in it. A delivery with nothing to commit commits
// nothing and pushes a branch with no work on it, which is not the case
// these tests are about.
func plantWorktree(t *testing.T, dir, text string) {
	t.Helper()

	_, r, err := openBoth(dir)
	if err != nil {
		t.Fatalf("open the workspace: %v", err)
	}

	wtDir := worktreeOf(t, dir)
	if err := r.AddWorktree(wtDir, "orbit/PAY-1"); err != nil {
		t.Fatalf("create the worktree: %v", err)
	}

	if err := os.WriteFile(filepath.Join(wtDir, "done.txt"), []byte(text+"\n"), 0o600); err != nil {
		t.Fatalf("write the work: %v", err)
	}
}

// worktreeOf is where the delivery of PAY-1 will look for its checkout.
func worktreeOf(t *testing.T, dir string) string {
	t.Helper()

	s, r, err := openBoth(dir)
	if err != nil {
		t.Fatalf("open the workspace: %v", err)
	}

	wtDir, err := s.WorktreeDir(r.Path, "PAY-1")
	if err != nil {
		t.Fatalf("locate the worktree: %v", err)
	}

	return wtDir
}

// gitOut runs one git command in a directory and answers what it printed.
func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v in %s: %v", args, dir, err)
	}

	return strings.TrimSpace(string(out))
}

// remoteOf is where the repository pushes to.
func remoteOf(t *testing.T, dir string) string {
	t.Helper()

	return gitOut(t, dir, "remote", "get-url", "acme")
}

// inside runs one git command in a directory.
func inside(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

// recordArgv is a fake gh that writes the arguments it was given, one to a
// line, and then says whatever the caller wanted said.
func recordArgv(t *testing.T, then string) (argv string) {
	t.Helper()

	argv = filepath.Join(t.TempDir(), "argv")
	fakeGh(t, "printf '%s\\n' \"$@\" > "+argv+"\n"+then)

	return argv
}

// gaveGh is the arguments gh was given, one to a line as it wrote them.
//
// A body written over several lines arrives as several arguments here. It is
// the last thing passed in every command these tests read, so the flags they
// look for are all above it.
func gaveGh(t *testing.T, argv string) []string {
	t.Helper()

	written, err := os.ReadFile(argv)
	if err != nil {
		t.Fatalf("read what gh was given: %v", err)
	}

	return strings.Split(strings.TrimSuffix(string(written), "\n"), "\n")
}

// flagValue is the argument gh was given after a flag.
func flagValue(t *testing.T, argv, flag string) string {
	t.Helper()

	args := gaveGh(t, argv)
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}

	t.Fatalf("gh was never given %s and a value after it:\n%s", flag, strings.Join(args, "\n"))

	return ""
}

// TestTheAnswerToADeliveryIsTheUrlAndNothingElse. gh prints a notice about
// its own next version to stderr whenever one exists, and a reader who is
// handed both streams is handed a link with a paragraph in front of it.
func TestTheAnswerToADeliveryIsTheUrlAndNothingElse(t *testing.T) {
	dir := deliverable(t, "make the thing")
	fakeGh(t, "echo 'A new release of gh is available: 2.40.0' >&2\necho https://github.test/acme/payments/pull/7")

	code, out, errOut := run(t, "pr", "-repo", dir, "PAY-1")
	if code != 0 {
		t.Fatalf("pr exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "https://github.test/acme/payments/pull/7") {
		t.Errorf("the pull request is not in what the reader was told: %q", out)
	}

	if strings.Contains(out, "A new release") {
		t.Errorf("gh's chatter came back inside the answer: %q", out)
	}
}

// TestADeliveryCarriesTheTaskAndItsBranchToGh. The pull request is opened
// for the branch the push just wrote, and it says which task it is for. Both
// are built here out of the task identifier, and a delivery that got either
// wrong would open a pull request against something else.
func TestADeliveryCarriesTheTaskAndItsBranchToGh(t *testing.T) {
	dir := deliverable(t, "make the thing")
	argv := recordArgv(t, "echo https://github.test/pr/1")

	if code, _, errOut := run(t, "pr", "-repo", dir, "PAY-1"); code != 0 {
		t.Fatalf("pr exited %d: %s", code, errOut)
	}

	if head := flagValue(t, argv, "--head"); head != "orbit/PAY-1" {
		t.Errorf("the pull request was opened for %q", head)
	}

	if base := flagValue(t, argv, "--base"); base != "main" {
		t.Errorf("the pull request was opened against %q", base)
	}

	if title := flagValue(t, argv, "--title"); !strings.HasPrefix(title, "PAY-1: ") {
		t.Errorf("the title does not say which task it is: %q", title)
	}
}

// TestATitleTooLongForTheFieldIsCutAtAWord. GitHub's title field holds 90
// characters and says nothing about the ones past it, so the cut is made
// here — at a space, and with three dots to say a cut was made. The three
// dots are part of what a reader sees, which is why the text is cut to 87.
func TestATitleTooLongForTheFieldIsCutAtAWord(t *testing.T) {
	dir := deliverable(t, "read the ledger and write the report and tell everybody who asked for it that it is ready to look at")
	argv := recordArgv(t, "echo https://github.test/pr/1")

	if code, _, errOut := run(t, "pr", "-repo", dir, "PAY-1"); code != 0 {
		t.Fatalf("pr exited %d: %s", code, errOut)
	}

	title := flagValue(t, argv, "--title")
	if n := utf8.RuneCountInString(title); n > 90 {
		t.Errorf("gh was given a title of %d characters: %q", n, title)
	}

	if !strings.HasSuffix(title, "...") {
		t.Errorf("the title was cut without saying so: %q", title)
	}

	if cut := strings.TrimSuffix(title, "..."); strings.HasSuffix(cut, " ") {
		t.Errorf("the title was cut after a space, so the dots hang off nothing: %q", title)
	}
}

// TestABranchThatWasPushedIsSaidToHaveBeenPushed. The push happens before gh
// does, so a gh that refuses leaves the work on the remote and the reader
// with something to do about it by hand. The sentence says so, and gh's own
// reason is still inside the error under it.
func TestABranchThatWasPushedIsSaidToHaveBeenPushed(t *testing.T) {
	dir := deliverable(t, "make the thing")
	fakeGh(t, "echo 'pull request create failed: HTTP 403' >&2\nexit 1")

	code, _, errOut := run(t, "pr", "-repo", dir, "PAY-1")
	if code == 0 {
		t.Fatal("pr exited 0 against a gh that refused")
	}

	for _, want := range []string{"orbit/PAY-1", "HTTP 403"} {
		if !strings.Contains(errOut, want) {
			t.Errorf("the refusal does not carry %q: %q", want, errOut)
		}
	}
}
