package cli

// repos() branches TestReposListsWhatIsUnderTheRoot (cli_test.go) and the
// "repos" cases in cli_invocations_test.go never reach: a bad flag, no
// argument at all (which asks os.Getwd rather than taking fs.Arg(0)), a
// repository with no remote configured, and one whose checkout is detached
// rather than on a branch.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReposEarlyExitAndNoArgument(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	// 1. A flag parse failure.
	if code, _, errOut := run(t, "repos", "-nosuchflag"); code == 0 {
		t.Error("repos with an unknown flag exited 0")
	} else if errOut == "" {
		t.Error("repos failed silently on a bad flag")
	}

	// 2. No argument at all: root comes from os.Getwd rather than fs.Arg(0).
	// Wherever the test process's working directory is, repos must still
	// exit cleanly — it says either a list of repositories or that there
	// are none under it.
	if code, _, errOut := run(t, "repos"); code != 0 {
		t.Errorf("repos with no argument exited %d: %s", code, errOut)
	}
}

// noRemoteDetachedRepo makes a repository with no remote at all, on a
// detached HEAD, so repos() takes both the "—" remote column and the
// "detached" base column.
func noRemoteDetachedRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	home := t.TempDir()
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"HOME="+home,
	)
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"commit", "-q", "--allow-empty", "-m", "first"},
		{"checkout", "-q", "--detach"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func TestReposShowsNoRemoteAndDetachedAsPlaceholders(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ORBIT_HOME", t.TempDir())
	dir := filepath.Join(root, "orphan")
	noRemoteDetachedRepo(t, dir)

	code, out, errOut := run(t, "repos", root)
	if code != 0 {
		t.Fatalf("repos exited %d: %s", code, errOut)
	}
	if !strings.Contains(out, "—") {
		t.Errorf("repos did not print the no-remote placeholder:\n%s", out)
	}
	if !strings.Contains(out, "detached") {
		t.Errorf("repos did not print the detached-HEAD placeholder:\n%s", out)
	}
}
