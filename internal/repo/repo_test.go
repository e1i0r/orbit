package repo

import (
	"errors"
	"os/exec"
	"path/filepath"
	"testing"
)

// makeRepo builds a real git repository in a temporary directory, with one
// commit on the named branch and a remote pointing nowhere. Nothing here
// touches the network.
func makeRepo(t *testing.T, dir, branch, remote string) {
	t.Helper()
	home := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
			// The developer's real ~/.gitconfig must never leak into the
			// test: a global commit.gpgsign or hooksPath would make this
			// suite pass or fail depending on whose machine runs it, and
			// would run that contributor's own hooks inside our suite.
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
			"HOME="+home,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", branch)
	run("commit", "-q", "--allow-empty", "-m", "first")
	if remote != "" {
		run("remote", "add", remote, "https://example.invalid/x.git")
	}
}

// isolateGitConfig keeps the developer's real ~/.gitconfig out of a test
// that calls the production git() helper itself rather than going through
// makeRepo's own isolated env. git() intentionally inherits the calling
// process's environment — that is correct in production, where a user's
// real config is load-bearing — so it is this process's environment that
// has to change for the duration of the test, mirroring the same three
// variables makeRepo sets for its own git commands above: a throwaway HOME,
// and both config files pointed at /dev/null so a contributor's global
// core.hooksPath cannot run their hooks inside our suite.
//
// t.Setenv reverts these when the test ends and fails the test if it is
// marked parallel; every test that calls this one runs serially.
func isolateGitConfig(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
}

func TestOpenReadsNameRemoteAndBase(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "payments")
	if err := mkdir(dir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	makeRepo(t, dir, "develop", "acme")

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if r.Name != "payments" {
		t.Errorf("Name = %q, want payments", r.Name)
	}
	if r.Remote != "acme" {
		t.Errorf("Remote = %q, want acme — the remote is not always called origin", r.Remote)
	}
	if r.Base != "develop" {
		t.Errorf("Base = %q, want develop", r.Base)
	}
}

func TestOpenPrefersOriginWhenThereAreSeveral(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app")
	if err := mkdir(dir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	makeRepo(t, dir, "main", "upstream")
	cmd := exec.Command("git", "remote", "add", "origin", "https://example.invalid/y.git")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("add origin: %v\n%s", err, out)
	}

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if r.Remote != "origin" {
		t.Errorf("Remote = %q, want origin", r.Remote)
	}
}

func TestOpenWithNoRemote(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "solo")
	if err := mkdir(dir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	makeRepo(t, dir, "main", "")

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if r.Remote != "" {
		t.Errorf("Remote = %q, want empty", r.Remote)
	}
}

func TestOpenRejectsSomethingThatIsNotARepository(t *testing.T) {
	if _, err := Open(t.TempDir()); err == nil {
		t.Error("Open succeeded on a plain directory")
	}
}

func TestDiscoverFindsEveryRepositoryBelow(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"one", "nested/two", "nested/deep/three"} {
		dir := filepath.Join(root, name)
		if err := mkdir(dir); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		makeRepo(t, dir, "main", "origin")
	}
	if err := mkdir(filepath.Join(root, "notarepo")); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	found, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(found) != 3 {
		names := make([]string, len(found))
		for i, r := range found {
			names[i] = r.Name
		}
		t.Fatalf("found %d repositories %v, want 3", len(found), names)
	}
}

func TestDiscoverDoesNotDescendIntoARepository(t *testing.T) {
	root := t.TempDir()
	outer := filepath.Join(root, "outer")
	if err := mkdir(outer); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	makeRepo(t, outer, "main", "origin")
	inner := filepath.Join(outer, "vendor", "inner")
	if err := mkdir(inner); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	makeRepo(t, inner, "main", "origin")

	found, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(found) != 1 {
		t.Errorf("found %d repositories, want 1 — a repository inside a repository is not a separate project", len(found))
	}
}

func TestGitErrorChainSurvives(t *testing.T) {
	dir := t.TempDir()
	_, err := git(dir, "rev-parse", "--show-toplevel")
	if err == nil {
		t.Fatal("git succeeded on a non-repository, expected failure")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Errorf("error chain was lost: %v cannot be recovered as *exec.ExitError", err)
	}
}
