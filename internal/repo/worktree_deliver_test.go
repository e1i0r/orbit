package repo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cloneWithRemote is a repository with one commit on main and a bare remote
// under the name given.
//
// The name is a parameter because it is the whole point of the second test:
// git calls a clone's remote "origin" and almost every repository leaves it
// there, which is exactly why hardcoding the word survives so long.
func cloneWithRemote(t *testing.T, remote string) string {
	t.Helper()
	remoteDir := t.TempDir()
	if _, err := git(remoteDir, "init", "--bare"); err != nil {
		t.Fatalf("git init --bare: %v", err)
	}

	cloneDir := t.TempDir()
	if _, err := git(cloneDir, "clone", remoteDir, "."); err != nil {
		t.Fatalf("git clone: %v", err)
	}
	if remote != "origin" {
		if _, err := git(cloneDir, "remote", "rename", "origin", remote); err != nil {
			t.Fatalf("git remote rename: %v", err)
		}
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
	if _, err := git(cloneDir, "push", remote, "HEAD:main"); err != nil {
		t.Fatalf("git push: %v", err)
	}
	return cloneDir
}

func TestCommitAndPushWorktree(t *testing.T) {
	cloneDir := cloneWithRemote(t, "origin")
	r := Repo{Path: cloneDir, Remote: "origin", Base: "main"}
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

// TestDeliveryUsesTheRepositorysOwnRemote is the regression. Repo carries a
// Remote because it is not always "origin" — the repository this tool was
// built against calls its remote "acme" — and the delivery path said the
// word anyway. Everything up to it worked, so the failure landed at the last
// step, after the work had been committed.
//
// The old test could not have caught it: it built a Repo with no Remote at
// all and pushed successfully, which is only possible while the field is
// being ignored.
func TestDeliveryUsesTheRepositorysOwnRemote(t *testing.T) {
	cloneDir := cloneWithRemote(t, "acme")
	r := Repo{Path: cloneDir, Name: "acme-app", Remote: "acme", Base: "main"}
	wtDir := filepath.Join(t.TempDir(), "wt-acme")
	branch := "orbit/TEST-2"
	if err := r.AddWorktree(wtDir, branch); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}
	if err := r.SyncBaseBranch(wtDir, "main"); err != nil {
		t.Errorf("SyncBaseBranch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wtDir, "test.txt"), []byte("orbit test\n"), 0o600); err != nil {
		t.Fatalf("write file in worktree: %v", err)
	}
	if err := r.CommitWorktree(wtDir, "feat(TEST-2): add test.txt"); err != nil {
		t.Fatalf("CommitWorktree: %v", err)
	}
	if err := r.PushBranch(wtDir, branch); err != nil {
		t.Fatalf("PushBranch: %v", err)
	}
	if out, err := git(cloneDir, "ls-remote", "--heads", "acme", branch); err != nil || out == "" {
		t.Errorf("the branch is not on the remote: %q, %v", out, err)
	}
}

// TestPushingWithNoRemoteSaysSo. A local repository is a repository — Open
// accepts one and pickRemote answers "" for it — so the empty name reaches
// here, and `git push -u "" HEAD:branch` would fail with a message about a
// repository named "" and nothing about what Orbit was trying to do.
func TestPushingWithNoRemoteSaysSo(t *testing.T) {
	dir := t.TempDir()
	if _, err := git(dir, "init"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	r := Repo{Path: dir, Name: "local", Base: "main"}
	err := r.PushBranch(dir, "orbit/TEST-3")
	if err == nil {
		t.Fatal("pushing a repository with no remote succeeded")
	}
	if got := err.Error(); !strings.Contains(got, "no remote") || !strings.Contains(got, "local") {
		t.Errorf("the refusal reads %q, want it to name the repository and the missing remote", got)
	}

	// And syncing is not an error, because there is nothing to sync from:
	// the delivery stops at the push, in terms of the push.
	if err := r.SyncBaseBranch(dir, "main"); err != nil {
		t.Errorf("SyncBaseBranch with no remote: %v", err)
	}
}
