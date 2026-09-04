package repo

// Discovery's two blind spots: Orbit's own state root, which is full of
// checkouts that are not projects, and a repository standing on no branch
// at all. repo_test.go covers the ordinary walk.

import (
	"path/filepath"
	"testing"
)

func TestDiscoverSkipsOrbitsOwnStateRoot(t *testing.T) {
	root := t.TempDir()
	// A state root with an ordinary name. The old rule skipped anything
	// beginning with a dot, which worked only for as long as the root was
	// called ~/.orbit and $ORBIT_HOME was never pointed anywhere else.
	state := filepath.Join(root, "orbit-state")
	t.Setenv("ORBIT_HOME", state)

	worktree := filepath.Join(state, "worktrees", "02c3a714b58d", "ACME-1")
	if err := mkdir(worktree); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	makeRepo(t, worktree, "orbit/ACME-1", "origin")

	real := filepath.Join(root, "payments")
	if err := mkdir(real); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	makeRepo(t, real, "main", "origin")

	found, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if len(found) != 1 || found[0].Name != "payments" {
		names := make([]string, len(found))
		for i, r := range found {
			names[i] = r.Name
		}

		t.Fatalf("found %v, want only [payments] — Orbit's own throwaway "+
			"checkouts are not projects to start tasks against", names)
	}
}

func TestOpenReportsADetachedHeadAsNoBranch(t *testing.T) {
	// This test calls git() itself, below, to detach HEAD — outside
	// makeRepo's own isolated env — so it needs the same isolation at the
	// process level.
	isolateGitConfig(t)

	dir := filepath.Join(t.TempDir(), "src")
	if err := mkdir(dir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	makeRepo(t, dir, "main", "origin")

	head, err := git(dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}

	if _, err := git(dir, "checkout", "--detach", head); err != nil {
		t.Fatalf("detach: %v", err)
	}

	r, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v — a detached checkout is still a repository worth listing", err)
	}

	if r.Base == "HEAD" {
		t.Error(`Base = "HEAD" — that is git saying there is no branch, not a branch named HEAD`)
	}

	if r.Base != "" {
		t.Errorf("Base = %q, want empty: there is no branch here", r.Base)
	}
}

func TestDiscoverSkipsIgnoredDirs(t *testing.T) {
	root := t.TempDir()

	real := filepath.Join(root, "payments")
	if err := mkdir(real); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	makeRepo(t, real, "main", "origin")

	for _, ignored := range []string{
		"node_modules", "vendor", "__pycache__", "build", "dist", "target", "coverage",
	} {
		fake := filepath.Join(root, ignored, "child")
		if err := mkdir(fake); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		makeRepo(t, fake, "main", "origin")
	}

	found, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if len(found) != 1 || found[0].Name != "payments" {
		names := make([]string, len(found))
		for i, r := range found {
			names[i] = r.Name
		}

		t.Fatalf("found %v, want only [payments]", names)
	}
}
