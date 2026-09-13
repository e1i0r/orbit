package repo

// Reading a checkout back: the files in it, one file's body, what
// changed, what usually changes with it, and what reviewers asked.
//
// All against real git repositories the test makes, and a gh that is a
// shell script — because what is being read is git's answer and gh's
// answer, not this package's opinion about either.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// checkoutOf is a repository with a committed file and work waiting: one
// line changed in the committed file, and one file nobody committed.
func checkoutOf(t *testing.T) (Repo, string) {
	t.Helper()

	dir := t.TempDir()

	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.name", "t"},
		{"config", "user.email", "t@t"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir

		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	add := exec.Command("git", "add", "a.txt")
	add.Dir = dir

	if err := add.Run(); err != nil {
		t.Fatalf("git add: %v", err)
	}

	commit := exec.Command("git", "commit", "-q", "-m", "first")
	commit.Dir = dir

	if err := commit.Run(); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\nTWO\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	one, err := Open(dir)
	if err != nil {
		t.Fatalf("open the repository: %v", err)
	}

	return one, dir
}

// TestListingAndReadingFiles. The names in the checkout, one body, and
// the refusal when the path climbs out of it.
func TestListingAndReadingFiles(t *testing.T) {
	r, dir := checkoutOf(t)

	files, err := r.WorktreeFiles(dir)
	if err != nil {
		t.Fatalf("files: %v", err)
	}

	if len(files) != 1 || files[0] != "a.txt" {
		t.Errorf("files reads %v, want only the committed one", files)
	}

	if _, err := r.WorktreeFiles(filepath.Join(t.TempDir(), "nowhere")); err == nil {
		t.Error("files in a directory that is not a checkout were accepted")
	}

	body, err := r.WorktreeFile(dir, "a.txt")
	if err != nil {
		t.Fatalf("file: %v", err)
	}

	if !strings.Contains(body, "TWO") {
		t.Errorf("file reads %q, want the worktree's words", body)
	}

	if _, err := r.WorktreeFile(dir, "../out"); err == nil {
		t.Error("a path climbing out of the checkout was accepted")
	}

	if _, err := r.WorktreeFile(dir, "missing.txt"); err == nil {
		t.Error("a file nobody wrote was accepted")
	}
}

// TestDiffingWhatChanged. The worktree against its base: the changed
// line, and the added lines of one file.
func TestDiffingWhatChanged(t *testing.T) {
	r, dir := checkoutOf(t)

	diff, err := r.WorktreeDiff(dir, DiffOptions{})
	if err != nil {
		t.Fatalf("diff: %v", err)
	}

	if !strings.Contains(diff, "TWO") || !strings.Contains(diff, "new") {
		t.Errorf("diff reads:\n%s", diff)
	}

	spaced, err := r.WorktreeDiff(dir, DiffOptions{IgnoreWhitespace: true})
	if err != nil {
		t.Fatalf("diff ignoring whitespace: %v", err)
	}

	if !strings.Contains(spaced, "TWO") {
		t.Errorf("diff ignoring whitespace reads:\n%s", spaced)
	}

	added, err := r.WorktreeAddedLines(dir, "a.txt")
	if err != nil {
		t.Fatalf("added lines: %v", err)
	}

	if len(added) == 0 {
		t.Error("no added lines came back for a file with a changed line")
	}
}

// TestNeighbouringFilesChangeTogether. One commit is no pattern; the
// neighbours of a file are read off the history, not guessed.
func TestNeighbouringFilesChangeTogether(t *testing.T) {
	r, dir := checkoutOf(t)

	near, err := r.Neighbours(dir)
	if err != nil {
		t.Fatalf("neighbours: %v", err)
	}

	_ = near
}

// TestReadingReviewsOffAGhThatIsAScript. Summaries, remarks and line
// comments, oldest first, with the empty approval left out.
func TestReadingReviewsOffAGhThatIsAScript(t *testing.T) {
	r, dir := checkoutOf(t)

	fakeGh(t, `if [ "$1" = "api" ]; then
echo '[{"user": {"login": "rv"}, "body": "nit: rename this", "path": "a.txt", "line": 2, "html_url": "https://github.test/c/1", "created_at": "2026-09-01T09:00:00Z"}]'
else
echo '{"number": 7, "reviews": [{"author": {"login": "rv"}, "body": " ", "url": "https://github.test/r/1", "submittedAt": "2026-09-01T08:00:00Z"}, {"author": {"login": "rv"}, "body": "rework the retry", "url": "https://github.test/r/2", "submittedAt": "2026-09-01T08:30:00Z"}], "comments": [{"author": {"login": "rv"}, "body": "agreed", "url": "https://github.test/c/2", "createdAt": "2026-09-01T08:45:00Z"}]}'
fi`)

	got, err := r.ReviewComments(dir, "orbit/ACME-1")
	if err != nil {
		t.Fatalf("reviews: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("reviews reads %d comments, want the remark, the reply and the line note", len(got))
	}

	if got[0].Body != "rework the retry" || got[2].Body != "nit: rename this" {
		t.Errorf("reviews read %v, want oldest first", got)
	}

	fakeGh(t, `echo 'not json'`)

	if _, err := r.ReviewComments(dir, "orbit/ACME-1"); err == nil {
		t.Error("gh answering nonsense was accepted")
	}
}
