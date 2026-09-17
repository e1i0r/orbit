package repo

// What a task hands back, and what it does not: Orbit's own copies of its
// decisions live in the worktree and must not reach the user's commit.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// worktreeOf is a repository with one worktree cut from it, which is the
// shape every test here needs.
func worktreeOf(t *testing.T) (Repo, string) {
	t.Helper()

	dir := t.TempDir()
	makeRepo(t, dir, "main", "")
	signs(t, dir)

	r := Repo{Path: dir, Base: "main"}

	wt := filepath.Join(t.TempDir(), "wt")
	if err := r.AddWorktree(wt, "orbit/T-1"); err != nil {
		t.Fatalf("add a worktree: %v", err)
	}

	return r, wt
}

// signs gives the repository an author, which the machine running the test
// may not have. makeRepo commits through an environment of its own; these
// tests commit through Orbit's own git runner, which carries the machine's
// config and nothing else, and CI's git has no identity in it at all.
func signs(t *testing.T, dir string) {
	t.Helper()

	for _, kv := range [][2]string{{"user.email", "t@t"}, {"user.name", "t"}} {
		if _, err := git(dir, "config", kv[0], kv[1]); err != nil {
			t.Fatalf("set %s: %v", kv[0], err)
		}
	}
}

// wrote puts a file in a worktree, making the directories it needs.
func wrote(t *testing.T, wtDir, path, text string) {
	t.Helper()

	full := filepath.Join(wtDir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("make the directory for %q: %v", path, err)
	}

	if err := os.WriteFile(full, []byte(text), 0o600); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

// committed is what the last commit of a worktree holds, one path per line.
func committed(t *testing.T, wtDir string) string {
	t.Helper()

	out, err := git(wtDir, "show", "--name-only", "--pretty=format:", "HEAD")
	if err != nil {
		t.Fatalf("read the last commit: %v", err)
	}

	return out
}

// TestACommitLeavesOrbitsOwnFilesBehind is the whole of the bug reported on
// 2026-09-09: a task that did clean work handed back a pull request with
// Orbit's decision copies in it.
func TestACommitLeavesOrbitsOwnFilesBehind(t *testing.T) {
	r, wt := worktreeOf(t)

	wrote(t, wt, "ledger.go", "package ledger\n")
	wrote(t, wt, ".orbit/decisions/sort-on-amount.md", "# why\n")

	if err := r.CommitWorktree(wt, "the work"); err != nil {
		t.Fatalf("commit: %v", err)
	}

	held := committed(t, wt)
	if !strings.Contains(held, "ledger.go") {
		t.Errorf("the work is not in the commit:\n%s", held)
	}

	if strings.Contains(held, orbitDir) {
		t.Errorf("Orbit's own files are in the commit:\n%s", held)
	}
}

// TestAWorktreeMadeBeforeTheExcludeIsCleanedAnyway covers the boards that
// already have the copies in their index: the exclude does nothing for a
// path git has been told about, so the staging is undone as well.
func TestAWorktreeMadeBeforeTheExcludeIsCleanedAnyway(t *testing.T) {
	r, wt := worktreeOf(t)

	wrote(t, wt, ".orbit/decisions/kept.md", "# why\n")

	if _, err := git(wt, "add", "-f", filepath.Join(orbitDir, "decisions", "kept.md")); err != nil {
		t.Fatalf("stage it the way the old worktrees did: %v", err)
	}

	wrote(t, wt, "ledger.go", "package ledger\n")

	if err := r.CommitWorktree(wt, "the work"); err != nil {
		t.Fatalf("commit: %v", err)
	}

	if held := committed(t, wt); strings.Contains(held, orbitDir) {
		t.Errorf("a staged decision copy still reached the commit:\n%s", held)
	}
}

// TestARepositoryOfItsOwnOrbitDirKeepsIt. An exclude says nothing about a
// path git already tracks, and the unstaging is written to leave those
// alone: a project that keeps its own .orbit/ commits changes to it like any
// other file.
func TestARepositoryOfItsOwnOrbitDirKeepsIt(t *testing.T) {
	dir := t.TempDir()
	makeRepo(t, dir, "main", "")
	signs(t, dir)

	// Tracked before any task ever ran, which is what makes it the
	// project's file rather than one of Orbit's copies.
	wrote(t, dir, ".orbit/theirs.md", "the project's own\n")

	if _, err := git(dir, "add", "-f", filepath.Join(orbitDir, "theirs.md")); err != nil {
		t.Fatalf("stage the project's own file: %v", err)
	}

	if _, err := git(dir, "commit", "-q", "-m", "theirs"); err != nil {
		t.Fatalf("commit the project's own file: %v", err)
	}

	r := Repo{Path: dir, Base: "main"}

	wt := filepath.Join(t.TempDir(), "wt")
	if err := r.AddWorktree(wt, "orbit/T-2"); err != nil {
		t.Fatalf("add a worktree: %v", err)
	}

	wrote(t, wt, ".orbit/theirs.md", "changed by the task\n")

	if err := r.CommitWorktree(wt, "the work"); err != nil {
		t.Fatalf("commit the change: %v", err)
	}

	if held := committed(t, wt); !strings.Contains(held, "theirs.md") {
		t.Errorf("the project's own file stopped being committed:\n%s", held)
	}
}

// TestTheExcludeIsWrittenOnce however many times it is asked for, and beside
// the reason it is there.
func TestTheExcludeIsWrittenOnce(t *testing.T) {
	r, wt := worktreeOf(t)

	for range 3 {
		if err := excludeOrbit(wt); err != nil {
			t.Fatalf("exclude: %v", err)
		}
	}

	said, err := os.ReadFile(filepath.Join(r.Path, ".git", "info", "exclude"))
	if err != nil {
		t.Fatalf("read the exclude: %v", err)
	}

	if n := strings.Count(string(said), excludeLine+"\n"); n != 1 {
		t.Errorf("the line is in the file %d times, want once:\n%s", n, said)
	}

	if !strings.Contains(string(said), "# Orbit's") {
		t.Errorf("no line saying who wrote it:\n%s", said)
	}
}

// TestTheExcludeIsTheOneEveryWorktreeReads. git keeps info/ in the common
// directory, so a linked worktree does not read a copy under its own git
// directory — which is the reason excludeOrbit asks git where to write
// rather than joining .git to the path it was given.
func TestTheExcludeIsTheOneEveryWorktreeReads(t *testing.T) {
	r, wt := worktreeOf(t)

	if _, err := os.Stat(filepath.Join(r.Path, ".git", "info", "exclude")); err != nil {
		t.Fatalf("the repository's own exclude was not written: %v", err)
	}

	wrote(t, wt, ".orbit/decisions/d.md", "# why\n")

	out, err := git(wt, "status", "--porcelain")
	if err != nil {
		t.Fatalf("status: %v", err)
	}

	if strings.Contains(out, orbitDir) {
		t.Errorf("the worktree still sees Orbit's directory:\n%s", out)
	}
}
