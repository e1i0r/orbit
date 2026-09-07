package panes

// The diff pane: four states before there is a diff to draw, and the cards
// once there is one.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// worktree is the pane with a diff read into it.
func worktree(t *testing.T) Env {
	t.Helper()

	e := world(t, nil)
	e.Task.Band = view.Running
	e.DiffKnown, e.Diff, e.Width = true, aDiff, 100

	return e
}

// TestTheDiffSaysWhichKindOfNothingItHas. Before the first answer lands
// there is nothing read yet, and that is not the same fact as an answer that
// came back empty — collapsing the two is how a git that hangs ends up
// asserting "no changes" to a question it was never answered.
func TestTheDiffSaysWhichKindOfNothingItHas(t *testing.T) {
	for _, c := range []struct {
		name string
		of   func(Env) Env
		want string
	}{
		{"a task that has not run", func(e Env) Env { e.Task.Band = view.ToDo; return e }, "todo queue"},
		{"an answer still out", func(e Env) Env { e.DiffKnown = false; return e }, "reading this task's worktree"},
		{"a worktree that is gone", func(e Env) Env { e.DiffMissing = true; return e }, "no working tree modifications"},
		{"a git that broke", func(e Env) Env { e.DiffFailed = "git did not answer in time"; return e }, "did not answer in time"},
		{"a worktree nothing touched", func(e Env) Env { e.Diff = ""; return e }, "no changes in this task's worktree"},
	} {
		if got := text(first(Diff(c.of(worktree(t))))); !strings.Contains(got, c.want) {
			t.Errorf("%s drew a pane that does not say %q:\n%s", c.name, c.want, got)
		}
	}
}

// TestEachFileIsACardAndTheCardsAreWhereTheyAreCounted, or the pointer
// collapses the file above the one it is on.
func TestEachFileIsACardAndTheCardsAreWhereTheyAreCounted(t *testing.T) {
	e := worktree(t)

	rows, heads := Diff(e)
	if len(heads) != 2 {
		t.Fatalf("the pane counted %d cards for a diff of two files", len(heads))
	}

	got := text(rows)
	for _, want := range []string{"store/items.go", "store/items_test.go", "upsert(ctx, item)"} {
		if !strings.Contains(got, want) {
			t.Errorf("the diff does not carry %q:\n%s", want, got)
		}
	}

	for at, i := range heads {
		if at < 0 || at >= len(rows) {
			t.Errorf("card %d is counted at row %d and the pane drew %d rows", i, at, len(rows))
		}
	}
}

// TestACollapsedFileIsAHeadWithNothingUnderIt. A diff of thirty files is
// read one file at a time, and the ones already read are in the way.
func TestACollapsedFileIsAHeadWithNothingUnderIt(t *testing.T) {
	e := worktree(t)
	e.Collapsed = map[string]bool{"store/items.go": true}

	got := text(first(Diff(e)))
	if strings.Contains(got, "upsert(ctx, item)") {
		t.Errorf("a collapsed file still shows its lines:\n%s", got)
	}

	// Its head stays, or the file would read as one the run did not touch.
	if !strings.Contains(got, "store/items.go") {
		t.Errorf("a collapsed file lost its own name:\n%s", got)
	}

	// And the file beside it is untouched by that.
	if !strings.Contains(got, "TestUpsertIsIdempotent") {
		t.Errorf("collapsing one file hid another:\n%s", got)
	}
}

// TestEachCardCarriesWhyTheFileWasTouched, read out of what the run wrote
// about it, and the reader can turn it off: the sentence is the record's
// word and not a proof, and a diff read line by line does not want a
// paragraph over every card.
func TestEachCardCarriesWhyTheFileWasTouched(t *testing.T) {
	e := worktree(t)
	e.Rationale = true
	e.Entries = []view.Entry{{
		Kind: "phase.finished", Phase: "implement",
		Text: "- store/items.go: matched on the bank reference instead of the id",
	}}

	if got := text(first(Diff(e))); !strings.Contains(got, "matched on the bank reference") {
		t.Errorf("the card does not say why the file was touched:\n%s", got)
	}

	e.Rationale = false

	if got := text(first(Diff(e))); strings.Contains(got, "matched on the bank reference") {
		t.Errorf("the sentence is still drawn with the rationale turned off:\n%s", got)
	}
}

// TestALongLineIsCutUnlessTheReaderAsksForItWhole. A diff of a generated
// file arrives as one line of several thousand cells, and wrapping it would
// push the rest of the hunk off the bottom of the screen.
func TestALongLineIsCutUnlessTheReaderAsksForItWhole(t *testing.T) {
	long := "+	" + strings.Repeat("aVeryLongIdentifier", 30)
	e := worktree(t)
	e.Diff = "diff --git a/bundle.js b/bundle.js\n--- a/bundle.js\n+++ b/bundle.js\n@@\n" + long + "\n"

	rows := len(first(Diff(e)))

	e.Expanded = true

	if wrapped := len(first(Diff(e))); wrapped <= rows {
		t.Errorf("asking for the whole line drew %d rows against %d cut, want it wrapped", wrapped, rows)
	}
}
