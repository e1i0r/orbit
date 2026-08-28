package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootPathFallbackAndReposErrorBranches(t *testing.T) {
	// 1. RootPath with unset ORBIT_HOME
	t.Setenv("ORBIT_HOME", "")

	rp, err := RootPath()
	if err != nil {
		t.Fatalf("RootPath fallback failed: %v", err)
	}

	if !strings.Contains(rp, ".orbit") {
		t.Errorf("expected RootPath to contain '.orbit', got %q", rp)
	}

	// 2. Open() with unset ORBIT_HOME
	st, err := Open()
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	if st == nil {
		t.Fatal("expected non-nil store from Open()")
	}

	// 3. Repos() edge cases: regular file in repos dir, corrupt marker, missing marker
	tmpRoot := t.TempDir()

	storeObj, err := New(tmpRoot)
	if err != nil {
		t.Fatal(err)
	}

	reposDir := filepath.Join(tmpRoot, "repos")
	if err := os.MkdirAll(reposDir, 0o700); err != nil {
		t.Fatal(err)
	}

	// Non-directory file in reposDir
	if err := os.WriteFile(filepath.Join(reposDir, "regular_file.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Subdir without 'repo' file
	if err := os.MkdirAll(filepath.Join(reposDir, "hash1"), 0o700); err != nil {
		t.Fatal(err)
	}

	// Subdir with corrupt 'repo' file (missing "path: " prefix)
	hash2 := filepath.Join(reposDir, "hash2")
	if err := os.MkdirAll(hash2, 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(hash2, "repo"), []byte("invalid-format\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Subdir with valid 'repo' file
	hash3 := filepath.Join(reposDir, "hash3")
	if err := os.MkdirAll(hash3, 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(hash3, "repo"), []byte("path: /valid/path\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	repos, rErr := storeObj.Repos()
	if rErr == nil {
		t.Error("expected ReposError when encountering missing and corrupt repo markers")
	}

	if len(repos) != 1 || repos[0].Path != "/valid/path" {
		t.Errorf("expected 1 valid repo, got %+v", repos)
	}
}

func TestNewAndCreateTaskDirBlockerErrors(t *testing.T) {
	// 1. New error when root cannot be created
	blockerFile := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blockerFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := New(filepath.Join(blockerFile, "sub_store")); err == nil {
		t.Error("expected error creating store under regular file")
	}

	// 2. CreateTaskDir error when repos directory is blocked by a file
	validRoot := t.TempDir()

	s, err := New(validRoot)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(validRoot, "repos"), []byte("blocker"), 0o600); err != nil {
		t.Fatal(err)
	}

	repoPath := filepath.Join(t.TempDir(), "project")
	if _, err := s.CreateTaskDir(repoPath, "TASK-1"); err == nil {
		t.Error("expected error creating task dir when repos is a file")
	}

	// 3. CreateWorktreeParent error when worktrees is a file
	if err := os.WriteFile(filepath.Join(validRoot, "worktrees"), []byte("blocker"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := s.CreateWorktreeParent(repoPath, "TASK-1"); err == nil {
		t.Error("expected error creating worktree parent when worktrees is a file")
	}

	// 4. CreateTaskDir error when tasks directory is blocked by a file
	validRoot2 := t.TempDir()

	s2, err := New(validRoot2)
	if err != nil {
		t.Fatal(err)
	}

	rDir, err := s2.RepoDir(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(rDir, 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(rDir, "tasks"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := s2.CreateTaskDir(repoPath, "TASK-1"); err == nil {
		t.Error("expected error when tasks is a regular file")
	}
}
