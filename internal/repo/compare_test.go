package repo

// Running the same checks on both sides of the change.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// changed is a repository on a branch, with a worktree holding a change.
func changed(t *testing.T) (Repo, string) {
	t.Helper()

	dir := t.TempDir()
	r := Repo{Path: dir, Name: "sums", Base: "main"}

	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", "a@b.c"},
		{"config", "user.name", "a"},
	} {
		if _, err := git(dir, args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}

	write(t, filepath.Join(dir, "check.sh"), "#!/bin/sh\nexit 0\n")

	if _, err := git(dir, "add", "-A"); err != nil {
		t.Fatalf("add: %v", err)
	}

	if _, err := git(dir, "commit", "-m", "the base"); err != nil {
		t.Fatalf("commit: %v", err)
	}

	wt := filepath.Join(t.TempDir(), "work")
	if err := r.AddWorktree(wt, "orbit/task"); err != nil {
		t.Fatalf("worktree: %v", err)
	}

	return r, wt
}

func write(t *testing.T, path, body string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestACheckThatPassedBeforeAndFailsNowIsTheOneToLookAt, and the reverse is
// the work: both are facts with an exit code behind them.
func TestACheckThatPassedBeforeAndFailsNowIsTheOneToLookAt(t *testing.T) {
	r, wt := changed(t)

	// The work breaks the check that passed on the base.
	write(t, filepath.Join(wt, "check.sh"), "#!/bin/sh\necho 'two assertions failed'\nexit 1\n")

	got, err := r.Compare(wt, []Check{{Name: "checks", Command: "sh check.sh"}})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("ran %d checks", len(got))
	}

	d := got[0]
	if d.Same() || !d.Broke() || d.Fixed() {
		t.Errorf("the verdict is same=%v broke=%v fixed=%v", d.Same(), d.Broke(), d.Fixed())
	}

	if !strings.Contains(d.Now.Out, "two assertions failed") {
		t.Errorf("what it printed is %q", d.Now.Out)
	}

	if !d.Base.Passed() {
		t.Errorf("the base answered exit %d: %q", d.Base.Exit, d.Base.Out)
	}
}

// TestACheckThatFailedBeforeAndPassesNowIsTheWork.
func TestACheckThatFailedBeforeAndPassesNowIsTheWork(t *testing.T) {
	r, wt := changed(t)

	// The base is broken; the work fixes it.
	write(t, filepath.Join(r.Path, "check.sh"), "#!/bin/sh\nexit 1\n")

	if _, err := git(r.Path, "commit", "-am", "break it"); err != nil {
		t.Fatalf("commit: %v", err)
	}

	got, err := r.Compare(wt, []Check{{Name: "checks", Command: "sh check.sh"}})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}

	if !got[0].Fixed() || got[0].Same() {
		t.Errorf("the verdict is %+v", got[0])
	}
}

// TestBothSidesAgreeingIsNotNews, whatever either of them printed.
func TestBothSidesAgreeingIsNotNews(t *testing.T) {
	r, wt := changed(t)

	got, err := r.Compare(wt, []Check{{Name: "checks", Command: "sh check.sh"}})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}

	if !got[0].Same() || got[0].Broke() || got[0].Fixed() {
		t.Errorf("two passes read as a divergence: %+v", got[0])
	}
}

// TestTheBaseCheckoutLeavesNothingBehind: no branch, no worktree entry, no
// directory. It is the one thing Orbit writes into a repository it does not
// own.
func TestTheBaseCheckoutLeavesNothingBehind(t *testing.T) {
	r, wt := changed(t)

	if _, err := r.Compare(wt, []Check{{Name: "checks", Command: "sh check.sh"}}); err != nil {
		t.Fatalf("Compare: %v", err)
	}

	out, err := git(r.Path, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list: %v", err)
	}

	if strings.Contains(out, "orbit-base-") {
		t.Errorf("the base checkout is still registered:\n%s", out)
	}

	branches, err := git(r.Path, "branch", "--list")
	if err != nil {
		t.Fatalf("branch: %v", err)
	}

	if strings.Contains(branches, "orbit-base") {
		t.Errorf("a branch was left behind:\n%s", branches)
	}
}

// TestNothingToCompareAgainstIsSaidRatherThanGuessed.
func TestNothingToCompareAgainstIsSaidRatherThanGuessed(t *testing.T) {
	detachedRepo := Repo{Path: t.TempDir(), Name: "none"}

	if _, err := detachedRepo.Compare("anywhere", []Check{{Name: "x", Command: "true"}}); err == nil {
		t.Error("a repository on no branch compared against something")
	}

	got, err := detachedRepo.Compare("anywhere", nil)
	if err != nil || got != nil {
		t.Errorf("no checks answered %+v, %v", got, err)
	}
}
