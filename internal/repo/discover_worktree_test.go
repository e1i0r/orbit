package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func TestDiscoverEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Root containing a non-git file and hidden dir
	hiddenDir := filepath.Join(tmpDir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0o700); err != nil {
		t.Fatal(err)
	}
	dummyFile := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(dummyFile, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 2. Directory with fake/broken .git so Open fails and gets skipped
	brokenDir := filepath.Join(tmpDir, "broken_repo")
	if err := os.MkdirAll(brokenDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brokenDir, ".git"), []byte("not a git file"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 3. State root directory inside search path should be skipped
	stateDir := filepath.Join(tmpDir, "orbit_state")
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ORBIT_HOME", stateDir)

	// 4. Valid git repos found by Discover
	repo1 := filepath.Join(tmpDir, "repo1")
	if err := os.MkdirAll(repo1, 0o700); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, repo1, "init", "-q", "-b", "main")
	gitCmd(t, repo1, "config", "user.email", "test@orbit.local")
	gitCmd(t, repo1, "config", "user.name", "Orbit Tester")
	gitCmd(t, repo1, "commit", "-q", "--allow-empty", "-m", "init")

	// Nested repo inside repo1 (should be skipped because parent is a repo)
	nested := filepath.Join(repo1, "vendor", "nested_repo")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, nested, "init", "-q", "-b", "main")
	gitCmd(t, nested, "config", "user.email", "test@orbit.local")
	gitCmd(t, nested, "config", "user.name", "Orbit Tester")
	gitCmd(t, nested, "commit", "-q", "--allow-empty", "-m", "init")

	// 5. Unreadable directory (permission error ignored during walk)
	unreadableDir := filepath.Join(tmpDir, "unreadable_dir")
	if err := os.MkdirAll(unreadableDir, 0o700); err == nil {
		_ = os.Chmod(unreadableDir, 0o000)                    //nolint:errcheck
		defer func() { _ = os.Chmod(unreadableDir, 0o700) }() //nolint:errcheck
	}

	repos, err := Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if len(repos) != 1 || repos[0].Name != "repo1" {
		t.Errorf("expected 1 repo named repo1, got %+v", repos)
	}
}

func TestMkdirError(t *testing.T) {
	tmpDir := t.TempDir()
	blocker := filepath.Join(tmpDir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := mkdir(filepath.Join(blocker, "sub"))
	if err == nil {
		t.Fatal("expected error creating dir under regular file")
	}
}

func TestOpenNotAGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := Open(tmpDir)
	if err == nil {
		t.Fatal("expected error opening non-git directory")
	}
}

func TestOpenDetachedHeadAndRemotes(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0o700); err != nil {
		t.Fatal(err)
	}

	gitCmd(t, repoDir, "init", "-q", "-b", "main")
	gitCmd(t, repoDir, "config", "user.email", "test@orbit.local")
	gitCmd(t, repoDir, "config", "user.name", "Orbit Tester")
	gitCmd(t, repoDir, "commit", "-q", "--allow-empty", "-m", "init")

	// Add a non-origin remote (upstream) and fork
	gitCmd(t, repoDir, "remote", "add", "fork", "https://github.com/example/fork.git")
	gitCmd(t, repoDir, "remote", "add", "upstream", "https://github.com/example/repo.git")

	rNoOrigin, err := Open(repoDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if rNoOrigin.Remote != "fork" {
		t.Errorf("expected remote 'fork', got %q", rNoOrigin.Remote)
	}

	// Add origin remote
	gitCmd(t, repoDir, "remote", "add", "origin", "https://github.com/example/origin.git")

	r, err := Open(repoDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if r.Remote != "origin" {
		t.Errorf("expected remote 'origin', got %q", r.Remote)
	}
	if r.Base != "main" {
		t.Errorf("expected base 'main', got %q", r.Base)
	}

	// Detach HEAD
	gitCmd(t, repoDir, "checkout", "-q", "--detach", "HEAD")
	rDetached, err := Open(repoDir)
	if err != nil {
		t.Fatalf("Open detached failed: %v", err)
	}
	if rDetached.Base != "" {
		t.Errorf("expected empty base on detached HEAD, got %q", rDetached.Base)
	}
}

func TestAddWorktreeErrors(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0o700); err != nil {
		t.Fatal(err)
	}

	gitCmd(t, repoDir, "init", "-q", "-b", "main")
	gitCmd(t, repoDir, "config", "user.email", "test@orbit.local")
	gitCmd(t, repoDir, "config", "user.name", "Orbit Tester")
	gitCmd(t, repoDir, "commit", "-q", "--allow-empty", "-m", "init")

	r, err := Open(repoDir)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Missing base branch
	rNoBase := r
	rNoBase.Base = ""
	if err := rNoBase.AddWorktree("/tmp/wt", "task-1"); err == nil {
		t.Error("expected error on empty base branch")
	}

	// 2. Mkdir failure (parent is a file)
	blocker := filepath.Join(tmpDir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := r.AddWorktree(filepath.Join(blocker, "sub", "wt"), "task-2"); err == nil {
		t.Error("expected error creating worktree under regular file")
	}

	// 3. Git worktree command failure (base branch nonexistent)
	rBadBase := r
	rBadBase.Base = "nonexistent-base-branch"
	if err := rBadBase.AddWorktree(filepath.Join(tmpDir, "wt-bad"), "task-3"); err == nil {
		t.Error("expected error creating worktree with nonexistent base branch")
	}
}

func TestAddWorktreeExistingBranchReuse(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0o700); err != nil {
		t.Fatal(err)
	}

	gitCmd(t, repoDir, "init", "-q", "-b", "main")
	gitCmd(t, repoDir, "config", "user.email", "test@orbit.local")
	gitCmd(t, repoDir, "config", "user.name", "Orbit Tester")
	gitCmd(t, repoDir, "commit", "-q", "--allow-empty", "-m", "init")

	r, err := Open(repoDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	wtDir := filepath.Join(tmpDir, "wt1")
	branch := "task-reuse"

	// Create first worktree
	if err := r.AddWorktree(wtDir, branch); err != nil {
		t.Fatalf("AddWorktree failed: %v", err)
	}

	// Remove worktree directory manually (simulating developer manual cleanup)
	if err := os.RemoveAll(wtDir); err != nil {
		t.Fatal(err)
	}

	// Add worktree again on same branch (exercises r.hasBranch(branch) reuse path)
	if err := r.AddWorktree(wtDir, branch); err != nil {
		t.Fatalf("AddWorktree with existing branch failed: %v", err)
	}

	// Remove via RemoveWorktree
	if err := r.RemoveWorktree(wtDir); err != nil {
		t.Fatalf("RemoveWorktree failed: %v", err)
	}
}

func TestRemoveWorktreeInvalidDir(t *testing.T) {
	r := Repo{Path: "/tmp", Base: "main"}
	err := r.RemoveWorktree("/nonexistent/worktree/dir")
	if err == nil {
		t.Fatal("expected error removing non-existent worktree")
	}
}

func TestSplitLinesEdgeCases(t *testing.T) {
	lines := splitLines("  \n\n  origin  \n  upstream  \n\n")
	if len(lines) != 2 || lines[0] != "origin" || lines[1] != "upstream" {
		t.Errorf("unexpected splitLines output: %v", lines)
	}

	empty := splitLines("   \n \t \n  ")
	if len(empty) != 0 {
		t.Errorf("expected empty slice for whitespace string, got %v", empty)
	}
}

func TestDiscoverRootError(t *testing.T) {
	tmpDir := t.TempDir()
	unreadableRoot := filepath.Join(tmpDir, "unreadable_root")
	if err := os.MkdirAll(unreadableRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unreadableRoot, 0o000); err == nil {
		defer func() { _ = os.Chmod(unreadableRoot, 0o700) }() //nolint:errcheck
		_, _ = Discover(unreadableRoot)                        //nolint:errcheck
	}
}
