package panes

// The artifacts pane: what Orbit wrote about the run, what the run wrote
// about the repository, and what one of those files turns out to hold.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// listed is the pane with a directory behind it and the worktree read.
func listed(t *testing.T) Env {
	t.Helper()

	e := world(t, nil)
	e.FilesKnown, e.DiffKnown = true, true
	e.Files = []view.File{
		{Name: "task.md", Size: 512},
		{Name: "events.jsonl", Size: 4096},
		{Name: "control", Size: 6},
	}
	e.Diff = aDiff

	return e
}

// TestEveryFileIsNamedMeasuredAndExplained. The name and the size are read
// off the disk and are true; a sentence invented beside them would be the
// one part of the row that is not, so a name this build does not know is
// described as nothing at all.
func TestEveryFileIsNamedMeasuredAndExplained(t *testing.T) {
	got := text(first(Artifacts(listed(t))))

	for _, want := range []string{
		"task.md", "512 B", "the task as it was written",
		"events.jsonl", "4 k", "one line per event",
		"control", "6 B", "the word the run was last told",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the listing does not carry %q:\n%s", want, got)
		}
	}
}

// TestTheWorktreeHalfIsCountedFromTheDiff. The worktree is a checkout Orbit
// does not keep, so the file that is not in the diff is the file the run
// left alone.
func TestTheWorktreeHalfIsCountedFromTheDiff(t *testing.T) {
	got := text(first(Artifacts(listed(t))))

	for _, want := range []string{"what the run changed", "2 files", "store/items.go", "see the diff tab"} {
		if !strings.Contains(got, want) {
			t.Errorf("the worktree half does not carry %q:\n%s", want, got)
		}
	}
}

// TestAnUnreadAnswerAndAnEmptyOneAreTwoSentences. A directory nothing has
// answered about yet and one that is empty are different facts, and the
// second is the one that says the task has never run.
func TestAnUnreadAnswerAndAnEmptyOneAreTwoSentences(t *testing.T) {
	for _, c := range []struct {
		name string
		of   func(Env) Env
		want string
	}{
		{"a listing still out", func(e Env) Env { e.FilesKnown = false; return e }, "reading the task's directory"},
		{"an empty directory", func(e Env) Env { e.Files = nil; return e }, "nothing written yet"},
		{"a listing that failed", func(e Env) Env { e.FilesFailed = "no such directory"; return e }, "no such directory"},
		{"a worktree still out", func(e Env) Env { e.DiffKnown = false; return e }, "reading the worktree"},
		{"a worktree that failed", func(e Env) Env { e.DiffFailed = "not a git checkout"; return e }, "not a git checkout"},
		{"a worktree nothing touched", func(e Env) Env { e.Diff = ""; return e }, "no changes in this task's worktree"},
	} {
		if got := text(first(Artifacts(c.of(listed(t))))); !strings.Contains(got, c.want) {
			t.Errorf("%s drew a pane that does not say %q:\n%s", c.name, c.want, got)
		}
	}
}

// TestAnOpenedFileShowsWhatItHoldsOrWhyItCannot. A file is read when it is
// opened and not before, so the first frame after a click says the read is
// on its way rather than showing an empty file.
func TestAnOpenedFileShowsWhatItHoldsOrWhyItCannot(t *testing.T) {
	e := listed(t)
	e.RowOpen = func(i int) bool { return i == 0 }

	if got := text(first(Artifacts(e))); !strings.Contains(got, "opening the file") {
		t.Errorf("a file whose read is still out drew:\n%s", got)
	}

	for _, c := range []struct {
		name string
		held File
		want string
	}{
		{"a file that was read", File{Text: "Reconcile settlements\n", Whole: true}, "Reconcile settlements"},
		{"a file that would not open", File{Failed: "permission denied"}, "permission denied"},
		{"a file with nothing in it", File{Text: "   \n", Whole: true}, "this file is empty"},
		{"a file too big to read whole", File{Text: "the first page\n"}, "the rest of this file was not read"},
	} {
		e.Read = func(string) (File, bool) { return c.held, true }

		if got := text(first(Artifacts(e))); !strings.Contains(got, c.want) {
			t.Errorf("%s drew a pane that does not say %q:\n%s", c.name, c.want, got)
		}
	}
}

// TestEachFileIsTheHeadOfItsOwnRow, or a click opens the file above the one
// it landed on.
func TestEachFileIsTheHeadOfItsOwnRow(t *testing.T) {
	e := listed(t)

	rows, heads := Artifacts(e)
	if len(heads) != len(e.Files) {
		t.Fatalf("the pane offers %d rows to open for %d files", len(heads), len(e.Files))
	}

	for at, i := range heads {
		if !strings.Contains(text(rows[at:at+1]), e.Files[i].Name) {
			t.Errorf("row %d is counted as %q and drew %q", at, e.Files[i].Name, text(rows[at:at+1]))
		}
	}
}

// TestATaskThatLeftTheBoardListsNothing.
func TestATaskThatLeftTheBoardListsNothing(t *testing.T) {
	e := listed(t)
	e.Gone = true

	if got := text(first(Artifacts(e))); !strings.Contains(got, "no longer on the board") {
		t.Errorf("the artifacts of a task that left drew %q", got)
	}
}
