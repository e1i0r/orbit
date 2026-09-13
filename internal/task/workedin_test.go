package task

// The folder a task's work is in, folded out of what it changed.

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/e1i0r/orbit/internal/repo"
)

// changed is a list of files, as the diff hands them over.
func changed(paths ...string) []repo.Change {
	out := make([]repo.Change, 0, len(paths))
	for _, p := range paths {
		out = append(out, repo.Change{Path: p, Added: 1})
	}

	return out
}

// TestTheFolderIsTheDeepestOneHoldingEverythingChanged.
//
// Three files under internal/db is work in internal/db. One file under it
// and one under internal/ui is work in internal, which is true and is the
// only thing that is: a folder that held one of them would leave a rule
// reaching half of what somebody was looking at.
func TestTheFolderIsTheDeepestOneHoldingEverythingChanged(t *testing.T) {
	for _, one := range []struct {
		name  string
		files []string
		want  string
	}{
		{"one folder", []string{"internal/db/schema.go", "internal/db/queries.go"}, "internal/db"},
		{"two folders", []string{"internal/db/schema.go", "internal/ui/draw.go"}, "internal"},
		{"nothing changed", nil, ""},
	} {
		t.Run(one.name, func(t *testing.T) {
			if got := commonFolder(changed(one.files...)); got != one.want {
				t.Errorf("the work reads as %q, and it is in %q", got, one.want)
			}
		})
	}
}

// TestOneFileIsStillItsFolder.
//
// A rule about exactly one file is a precision a person means and types.
// What can be read off a diff is the folder, and answering with the file
// would file half the rules there are somewhere they reach nothing else —
// including the next file somebody adds beside it.
func TestOneFileIsStillItsFolder(t *testing.T) {
	if got := commonFolder(changed("internal/db/schema.go")); got != "internal/db" {
		t.Errorf("one file under a folder reads as %q", got)
	}
}

// TestWorkAtTheRootIsTheWholeCheckout, and the root is answered as nothing
// rather than as ".": a rule filed on a dot would be a second spelling of
// the checkout, and two spellings are two rules nobody can tell apart.
func TestWorkAtTheRootIsTheWholeCheckout(t *testing.T) {
	for _, files := range [][]string{
		{"README.md"},
		{"README.md", "internal/db/schema.go"},
	} {
		if got := commonFolder(changed(files...)); got != "" {
			t.Errorf("%v reads as %q", files, got)
		}
	}
}

// TestAFolderOnlyInTheWorktreeIsNotOffered.
//
// The work happens on a branch and the rule is filed against the checkout
// that branch was cut from, so a folder the task has just created is in one
// and not the other. Offered, it would put a refusal in front of somebody
// who typed nothing at all.
func TestAFolderOnlyInTheWorktreeIsNotOffered(t *testing.T) {
	here := t.TempDir()

	if err := os.MkdirAll(filepath.Join(here, "internal", "db"), 0o750); err != nil {
		t.Fatalf("make the folder: %v", err)
	}

	if got := thereToo(here, "internal/db"); got != "internal/db" {
		t.Errorf("a folder that is in the checkout reads as %q", got)
	}

	if got := thereToo(here, "internal/brandnew"); got != "" {
		t.Errorf("a folder only the worktree has reads as %q", got)
	}

	// A file is not a folder, and the fold answers folders: anything else
	// arriving here is a reading that went wrong somewhere earlier.
	at := filepath.Join(here, "internal", "db", "schema.go")
	if err := os.WriteFile(at, []byte("package db\n"), 0o600); err != nil {
		t.Fatalf("write the file: %v", err)
	}

	if got := thereToo(here, "internal/db/schema.go"); got != "" {
		t.Errorf("a file was offered as the folder the work was in: %q", got)
	}
}

// TestWhatTheRunChangedIsReadOffTheWorktree, which is the only part of this
// nothing can fake: the fold and the check above take lists, and a list is
// not a diff. Here git is asked what the task actually wrote.
func TestWhatTheRunChangedIsReadOffTheWorktree(t *testing.T) {
	s, r := fixture(t)
	tk := Task{ID: "ACME-40", Repo: r}

	// The folder has to be in the checkout the rule will be filed against,
	// so it is committed on the branch the worktree is cut from.
	commitFolder(t, r.Path, "internal/db/schema.go")

	wt, err := s.WorktreeDir(r.Path, tk.ID)
	if err != nil {
		t.Fatalf("where the worktree goes: %v", err)
	}

	if err := r.AddWorktree(wt, "orbit/"+tk.ID); err != nil {
		t.Fatalf("add the worktree: %v", err)
	}

	write(t, filepath.Join(wt, "internal/db/schema.go"), "package db\n\n// changed\n")
	write(t, filepath.Join(wt, "internal/db/queries.go"), "package db\n")

	if got := workedIn(s, tk); got != "internal/db" {
		t.Errorf("the run worked in %q", got)
	}
}

// commitFolder puts one file on the branch, so that the folder holding it is
// somewhere a rule can point at.
func commitFolder(t *testing.T, dir, at string) {
	t.Helper()

	write(t, filepath.Join(dir, at), "package db\n")

	for _, args := range [][]string{{"add", "."}, {"commit", "-q", "-m", "the folder"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
			"HOME="+t.TempDir(),
		)

		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// write puts a file down, making the folders above it.
func write(t *testing.T, at, body string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
		t.Fatalf("make the folder for %s: %v", at, err)
	}

	if err := os.WriteFile(at, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", at, err)
	}
}
