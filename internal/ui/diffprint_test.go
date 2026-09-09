package ui

// The fingerprint, against a real worktree: the whole point of it is that it
// is cheaper than the diff and never says "the same" about a worktree that
// moved, and neither can be asserted against a fake git.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// TestTheFingerprintIsSteadyWhileNothingHappens. An engine that is thinking
// moves no files, and thinking is most of a phase: a fingerprint that
// changed on its own would put the diff back on the clock it was taken off.
func TestTheFingerprintIsSteadyWhileNothingHappens(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	tree := worktreeOf(t, gitRepo(t), "ACME-1")
	write(t, filepath.Join(tree, "retry.go"), "package retry\n")

	first := worktreePrint(tree)
	if first == "" {
		t.Fatal("no fingerprint at all, and the worktree is there")
	}

	if second := worktreePrint(tree); second != first {
		t.Errorf("the fingerprint moved on its own: %q then %q", first, second)
	}
}

// TestAFileWrittenInPlaceMovesTheFingerprint is why the names alone are not
// enough. An agent that rewrites one line leaves the same file in the same
// status list, and a fingerprint made of that list would report a worktree
// that changed as unchanged — the diff on screen would then be wrong for as
// long as the reader looked at it.
func TestAFileWrittenInPlaceMovesTheFingerprint(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	tree := worktreeOf(t, gitRepo(t), "ACME-1")
	path := filepath.Join(tree, "retry.go")
	write(t, path, "package retry\n\nfunc send() { a() }\n")

	before := worktreePrint(tree)

	// The same length, so only the content and the clock differ.
	write(t, path, "package retry\n\nfunc send() { b() }\n")

	if after := worktreePrint(tree); after == before {
		t.Error("a line rewritten in place left the fingerprint where it was")
	}
}

// TestANewFileMovesTheFingerprint, tracked or not: a file the agent wrote
// and never staged is the most interesting file there is.
func TestANewFileMovesTheFingerprint(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	tree := worktreeOf(t, gitRepo(t), "ACME-1")
	before := worktreePrint(tree)

	write(t, filepath.Join(tree, "brand_new.go"), "package retry\n")

	if after := worktreePrint(tree); after == before {
		t.Error("an untracked file left the fingerprint where it was")
	}
}

// TestAWorktreeThatIsGoneHasNoFingerprint, and no fingerprint means "ask git
// properly" rather than "nothing changed": a failure must never be read as
// sameness, or a broken worktree would freeze the last diff on screen for
// ever.
func TestAWorktreeThatIsGoneHasNoFingerprint(t *testing.T) {
	if print := worktreePrint(filepath.Join(t.TempDir(), "nowhere")); print != "" {
		t.Errorf("worktreePrint of a missing directory = %q, want empty", print)
	}
}

// TestTheDiffIsNotReadAgainWhenNothingMoved is the change itself: handed the
// fingerprint it last drew, the read answers that it is the same and writes
// no diff at all.
func TestTheDiffIsNotReadAgainWhenNothingMoved(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	repoPath := gitRepo(t)
	tree := worktreeOf(t, repoPath, "ACME-1")
	write(t, filepath.Join(tree, "retry.go"), "package retry\n\nfunc send() { backoff() }\n")

	r := &fakeReader{worktree: tree}
	task := view.Task{ID: "ACME-1", RepoPath: repoPath}

	first, ok := diffOf(r, task, baseRef{known: true}, "")().(diffMsg)
	if !ok {
		t.Fatal("diffOf did not answer with a diff")
	}

	if first.Same || first.Print == "" {
		t.Fatalf("the first read answered same=%v print=%q", first.Same, first.Print)
	}

	if !strings.Contains(first.Text, "backoff()") {
		t.Fatalf("the first read says:\n%s", first.Text)
	}

	again, ok := diffOf(r, task, baseRef{known: true}, first.Print)().(diffMsg)
	if !ok {
		t.Fatal("diffOf did not answer with a diff")
	}

	if !again.Same {
		t.Error("a worktree nobody touched was read again")
	}

	if again.Text != "" {
		t.Errorf("the second read carries a text: %q", again.Text)
	}
}

// TestTheDiffIsReadAgainWhenTheWorktreeMoved, which is the other half: the
// pane is on a clock so that a run writing in the worktree is watched, and a
// fingerprint that suppressed a real change would be worse than no clock.
func TestTheDiffIsReadAgainWhenTheWorktreeMoved(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	repoPath := gitRepo(t)
	tree := worktreeOf(t, repoPath, "ACME-1")
	path := filepath.Join(tree, "retry.go")
	write(t, path, "package retry\n\nfunc send() { backoff() }\n")

	r := &fakeReader{worktree: tree}
	task := view.Task{ID: "ACME-1", RepoPath: repoPath}

	first, ok := diffOf(r, task, baseRef{known: true}, "")().(diffMsg)
	if !ok {
		t.Fatal("diffOf did not answer with a diff")
	}

	// mtime has a resolution, and a test that writes twice in the same
	// nanosecond would be testing the clock rather than the fingerprint.
	if err := os.Chtimes(path, time.Now().Add(time.Second), time.Now().Add(time.Second)); err != nil {
		t.Fatalf("age the file: %v", err)
	}

	write(t, path, "package retry\n\nfunc send() { giveUp() }\n")

	again, ok := diffOf(r, task, baseRef{known: true}, first.Print)().(diffMsg)
	if !ok {
		t.Fatal("diffOf did not answer with a diff")
	}

	if again.Same {
		t.Fatal("the worktree changed and the read said it had not")
	}

	if !strings.Contains(again.Text, "giveUp()") {
		t.Errorf("the second read says:\n%s\nwant the change that was just made", again.Text)
	}
}
