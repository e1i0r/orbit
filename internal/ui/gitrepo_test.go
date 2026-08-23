package ui

// gitrepo_test.go is the one fixture in this package that is a real
// repository on disk, and it exists for one assertion: that the diff tab
// shows the task's worktree and not the reader's own checkout.
//
// It is written here rather than borrowed because internal/repo's makeRepo
// is unexported and lives in another package's test binary. Ten lines
// duplicated across a package boundary is the ordinary Go answer; exporting
// a test helper into production code to save them is the worse trade.
//
// Nothing here reaches the network, the real HOME, or $ORBIT_HOME. The
// remote is never added, both git config files are pointed at /dev/null, and
// TestMain has already moved HOME to a temporary directory.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// gitRepo builds a repository with one commit on main and returns its path.
func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	write(t, filepath.Join(dir, "retry.go"), "package retry\n\nfunc send() {}\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "first")
	return dir
}

// worktreeOf cuts a branch of its own for one task, the way the store does.
func worktreeOf(t *testing.T, repoPath, id string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), id)
	git(t, repoPath, "worktree", "add", "-q", "-b", id, dir, "main")
	return dir
}

// git runs one git command in one directory, with the contributor's own
// configuration kept out of it. A global commit.gpgsign or core.hooksPath
// would otherwise decide whether this suite passes, and would run that
// contributor's hooks inside it.
func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestTheDiffIsTheWorktreesAndNotTheRepositorys is the defect this task was
// dispatched with, written down as a test.
//
// Before the diff tab existed, this command ran git in view.Task.RepoPath —
// the repository the reader has open in their own editor. Both directories
// have uncommitted changes here, they say different things, and the pane may
// only ever show one of them: an agent's work under the reader's own heading
// is a screen that lies quietly, which is the expensive kind.
func TestTheDiffIsTheWorktreesAndNotTheRepositorys(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	repoPath := gitRepo(t)
	tree := worktreeOf(t, repoPath, "ACME-2662")
	write(t, filepath.Join(repoPath, "retry.go"), "package retry\n\nfunc send() { theReadersOwnEdit() }\n")
	write(t, filepath.Join(tree, "retry.go"), "package retry\n\nfunc send() { backoff() }\n")

	r := &fakeReader{worktree: tree}
	msg, ok := diffOf(r, view.Task{ID: "ACME-2662", RepoPath: repoPath})().(diffMsg)
	if !ok {
		t.Fatal("diffOf did not answer with a diff")
	}
	if msg.Err != nil {
		t.Fatalf("diff the worktree: %v", msg.Err)
	}
	if !strings.Contains(msg.Text, "backoff()") {
		t.Errorf("the diff says:\n%s\nwant the change made in the worktree", msg.Text)
	}
	if strings.Contains(msg.Text, "theReadersOwnEdit") {
		t.Errorf("the diff says:\n%s\nwant the reader's own checkout left out of it", msg.Text)
	}
	if msg.Tree != tree {
		t.Errorf("the diff came back from %q, want %q", msg.Tree, tree)
	}
}

// TestADiffWithoutAWorktreeSaysSo covers the other end: a task whose
// worktree the port cannot find is a sentence in the pane, never a blank one
// that reads as "no changes".
func TestADiffWithoutAWorktreeSaysSo(t *testing.T) {
	r := &fakeReader{treeErr: os.ErrNotExist}
	msg, ok := diffOf(r, view.Task{ID: "ACME-2662", RepoPath: "/nowhere"})().(diffMsg)
	if !ok {
		t.Fatal("diffOf did not answer with a diff")
	}
	if msg.Err == nil {
		t.Fatal("a worktree that could not be found came back as a diff")
	}
}
