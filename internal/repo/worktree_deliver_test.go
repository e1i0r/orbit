package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommitAndPushWorktree(t *testing.T) {
	// Initialize bare remote repo and a clone
	remoteDir := t.TempDir()
	if _, err := git(remoteDir, "init", "--bare"); err != nil {
		t.Fatalf("git init --bare: %v", err)
	}

	cloneDir := t.TempDir()
	if _, err := git(cloneDir, "clone", remoteDir, "."); err != nil {
		t.Fatalf("git clone: %v", err)
	}
	if _, err := git(cloneDir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("git config user.name: %v", err)
	}
	if _, err := git(cloneDir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("git config user.email: %v", err)
	}

	// Initial commit on main
	f := filepath.Join(cloneDir, "README.md")
	if err := os.WriteFile(f, []byte("# Test\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := git(cloneDir, "add", "README.md"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := git(cloneDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if _, err := git(cloneDir, "push", "origin", "HEAD:main"); err != nil {
		t.Fatalf("git push: %v", err)
	}

	r := Repo{Path: cloneDir, Base: "main"}
	wtDir := filepath.Join(t.TempDir(), "wt-test")
	branch := "orbit/TEST-1"
	if err := r.AddWorktree(wtDir, branch); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}

	// Create a new file in worktree
	newFile := filepath.Join(wtDir, "test.txt")
	if err := os.WriteFile(newFile, []byte("orbit test\n"), 0o600); err != nil {
		t.Fatalf("write file in worktree: %v", err)
	}

	// Test CommitWorktree
	if err := r.CommitWorktree(wtDir, "feat(TEST-1): add test.txt"); err != nil {
		t.Errorf("CommitWorktree failed: %v", err)
	}

	// Test PushBranch
	if err := r.PushBranch(wtDir, branch); err != nil {
		t.Errorf("PushBranch failed: %v", err)
	}
}
