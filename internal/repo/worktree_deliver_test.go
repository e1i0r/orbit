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

// TestAWorktreeNobodyChangedHasNothingToDeliver. A task may open a checkout
// of a repository, read it and decide it needs nothing. The branch is then
// the base branch under another name, and the delivery has to be able to
// tell that apart from work — a pull request for it asks a reviewer to look
// at no change at all.
func TestAWorktreeNobodyChangedHasNothingToDeliver(t *testing.T) {
	cloneDir := cloneWithRemote(t, "origin")
	r := Repo{Path: cloneDir, Name: "ledger", Remote: "origin", Base: "main"}
	wtDir := filepath.Join(t.TempDir(), "wt-untouched")

	if err := r.AddWorktree(wtDir, "orbit/TEST-4"); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}

	ahead, err := r.WorktreeAhead(wtDir, "main")
	if err != nil {
		t.Fatalf("WorktreeAhead: %v", err)
	}

	if ahead {
		t.Error("a checkout with no commit of its own reads as having work to deliver")
	}

	if err := os.WriteFile(filepath.Join(wtDir, "test.txt"), []byte("orbit test\n"), 0o600); err != nil {
		t.Fatalf("write file in worktree: %v", err)
	}

	// Uncommitted work is not counted, and does not have to be: the delivery
	// commits before it asks.
	if err := r.CommitWorktree(wtDir, "feat(TEST-4): add test.txt"); err != nil {
		t.Fatalf("CommitWorktree: %v", err)
	}

	if ahead, err = r.WorktreeAhead(wtDir, "main"); err != nil || !ahead {
		t.Errorf("a checkout with a commit on it reads as %v, %v", ahead, err)
	}

	// A base branch nothing answers to is not a count of nothing. It is a
	// question that could not be asked, and reading it as "nothing to
	// deliver" leaves a repository's work on the floor without a word.
	if _, err := r.WorktreeAhead(wtDir, "no-such-branch"); err == nil {
		t.Error("counting against a branch that is not there came back without an error")
	}
}

// TestABaseNobodyFetchedIsNotWhatTheWorkIsCountedAgainst. The local branch of
// a checkout that has not been fetched in a week is behind its remote, and
// counting a task's work against it reports a repository as changed because
// somebody else changed it. Here somebody else pushes from their own clone,
// so this one's main stays where it was and only origin/main moves.
func TestABaseNobodyFetchedIsNotWhatTheWorkIsCountedAgainst(t *testing.T) {
	cloneDir := cloneWithRemote(t, "origin")
	r := Repo{Path: cloneDir, Name: "ledger", Remote: "origin", Base: "main"}
	wtDir := filepath.Join(t.TempDir(), "wt-behind")

	// The local base branch has to exist before the worktree is cut from it,
	// or git makes one out of the remote-tracking ref at the moment it is
	// asked — which is a local branch that cannot be stale, and the staleness
	// is the whole of what this test is about.
	if _, err := git(cloneDir, "rev-parse", "--verify", "--quiet", "main"); err != nil {
		if _, err := git(cloneDir, "branch", "main"); err != nil {
			t.Fatalf("git branch main: %v", err)
		}
	}

	if err := r.AddWorktree(wtDir, "orbit/TEST-5"); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}

	theirs(t, cloneDir)

	if err := r.SyncBaseBranch(wtDir, "main"); err != nil {
		t.Fatalf("SyncBaseBranch: %v", err)
	}

	// The worktree now holds one commit its own repository's main has never
	// heard of, and not one of them is this task's.
	ahead, err := r.WorktreeAhead(wtDir, "main")
	if err != nil {
		t.Fatalf("WorktreeAhead: %v", err)
	}

	if ahead {
		t.Error("a checkout that only caught up with its remote reads as having work of its own")
	}
}

// theirs is a commit pushed to the same remote by somebody else — a second
// clone, which is what makes the first one's main go stale.
func theirs(t *testing.T, cloneDir string) {
	t.Helper()

	remote, err := git(cloneDir, "remote", "get-url", "origin")
	if err != nil {
		t.Fatalf("git remote get-url: %v", err)
	}

	other := t.TempDir()
	if _, err := git(other, "clone", "-b", "main", strings.TrimSpace(remote), "."); err != nil {
		t.Fatalf("git clone: %v", err)
	}

	if err := os.WriteFile(filepath.Join(other, "theirs.txt"), []byte("theirs\n"), 0o600); err != nil {
		t.Fatalf("write their file: %v", err)
	}

	for _, args := range [][]string{
		{"config", "user.name", "Somebody Else"},
		{"config", "user.email", "else@example.com"},
		{"add", "-A"},
		{"commit", "-m", "somebody else"},
		{"push", "origin", "HEAD:main"},
	} {
		if _, err := git(other, args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
}
