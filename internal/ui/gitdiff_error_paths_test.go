package ui

// gitdiff_more_coverage_test.go rounds out gitDiff, runGitDiff and diffOf's
// error branches that gitrepo_test.go's own suite does not reach: a task
// with no repository path at all, a worktree that is not a git repository,
// and a merge-base compare against a base that never answers.

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

func TestDiffOfWithNoRepositoryPath(t *testing.T) {
	r := &fakeReader{worktree: "/somewhere"}
	msg, ok := diffOf(r, view.Task{ID: "ACME-1"}, baseRef{})().(diffMsg)
	if !ok {
		t.Fatal("diffOf did not answer with a diffMsg")
	}
	if msg.Err == nil || !strings.Contains(msg.Err.Error(), "does not say where its repository is") {
		t.Errorf("diffOf with no RepoPath = %v, want the missing-repository sentence", msg.Err)
	}
}

func TestGitDiffAgainstANonRepository(t *testing.T) {
	dir := t.TempDir() // no .git here at all

	out, noBase, err := gitDiff(dir, "")
	if err == nil {
		t.Fatal("gitDiff against a plain directory returned no error")
	}
	if out != "" || noBase {
		t.Errorf("gitDiff on failure returned out=%q noBase=%v, want both zero", out, noBase)
	}

	// The same failure, propagated all the way through diffOf.
	r := &fakeReader{worktree: dir}
	msg, ok := diffOf(r, view.Task{ID: "ACME-1", RepoPath: dir}, baseRef{known: true})().(diffMsg)
	if !ok {
		t.Fatal("diffOf did not answer with a diffMsg")
	}
	if msg.Err == nil {
		t.Error("diffOf against a non-repository worktree returned no error")
	}
}

func TestGitDiffTimesOutOnAHungBaseCompare(t *testing.T) {
	old := gitDiffTimeout
	gitDiffTimeout = 50 * time.Millisecond
	t.Cleanup(func() { gitDiffTimeout = old })
	fakeGit(t, 5)

	_, noBase, err := gitDiff(t.TempDir(), "main")
	if !errors.Is(err, errGitTimedOut) {
		t.Errorf("gitDiff against a hung base compare = %v, want errGitTimedOut", err)
	}
	if noBase {
		t.Error("a timed-out compare was reported as noBase, want the claim withheld")
	}
}
