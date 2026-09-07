package panes

// The overview's blocks: the flow card each band draws, what the run did to
// the working tree, and where it ran.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
)

// aDiff is a small unified diff: two files, three added lines and one taken
// away.
const aDiff = `diff --git a/store/items.go b/store/items.go
--- a/store/items.go
+++ b/store/items.go
@@
-	insert(ctx, item)
+	upsert(ctx, item)
+	log(ctx, item)
diff --git a/store/items_test.go b/store/items_test.go
--- a/store/items_test.go
+++ b/store/items_test.go
@@
+	TestUpsertIsIdempotent(t)
`

// TestTheFlowCardSaysWhatTheBandMeans. The overview draws one card and the
// tree next door draws all of it, so this card is where a reader learns
// whether the run is going, waiting, done or has never started.
func TestTheFlowCardSaysWhatTheBandMeans(t *testing.T) {
	for _, c := range []struct {
		band view.Band
		want string
	}{
		{view.Running, "running model"},
		{view.NeedsYou, "NEEDS YOU"},
		{view.Done, "flow completed successfully"},
		{view.ToDo, "ready to start"},
	} {
		e := world(t, []view.Entry{
			{Kind: "phase.started", Phase: "implement", At: ago(30 * time.Minute)},
			{Kind: "phase.finished", Phase: "implement", At: ago(20 * time.Minute)},
		})
		e.Task.Band = c.band
		e.Word, e.Role = "waiting: review", theme.Warn
		e.Live = "⚡ "

		if got := text(Overview(e)); !strings.Contains(got, c.want) {
			t.Errorf("a task in %v drew a card that does not say %q:\n%s", c.band, c.want, got)
		}
	}
}

// TestARunThatBrokeSaysSoOnTheCard rather than reporting the flow completed.
func TestARunThatBrokeSaysSoOnTheCard(t *testing.T) {
	e := world(t, nil)
	e.Task.Band = view.Done
	e.Task.Reason = view.Reason{Key: view.ReasonFailed}

	if got := text(Overview(e)); !strings.Contains(got, "stopped on failure") {
		t.Errorf("a run that failed reads as one that finished:\n%s", got)
	}
}

// TestTheLiveCardCarriesWhatTheRunIsDoingNow: the phase, the action, the
// thought in flight and how many tools it has called.
func TestTheLiveCardCarriesWhatTheRunIsDoingNow(t *testing.T) {
	e := world(t, nil)
	e.Live = "⠋ "
	e.Task.Band = view.Running
	e.Task.Phase = "implement"
	e.Task.CurrentAction = "editing store/items.go"
	e.Task.CurrentThought = "the collision is on the primary key"
	e.Task.ToolCallCount = 7

	got := text(Overview(e))
	for _, want := range []string{"implement", "editing store/items.go", "primary key", "7 tool calls"} {
		if !strings.Contains(got, want) {
			t.Errorf("the live card does not carry %q:\n%s", want, got)
		}
	}
}

// TestTheChangesBlockCountsBothSidesOfTheDiff, and names the files it
// touched.
func TestTheChangesBlockCountsBothSidesOfTheDiff(t *testing.T) {
	e := world(t, nil)
	e.Diff = aDiff

	got := text(Overview(e))
	for _, want := range []string{"+3", "−1", "2 files", "store/items.go", "store/items_test.go"} {
		if !strings.Contains(got, want) {
			t.Errorf("the changes block does not carry %q:\n%s", want, got)
		}
	}

	// Changed is the same reading, asked from the artifacts pane: the file
	// that is not in the diff is the file the run left alone.
	if files := Changed(aDiff); len(files) != 2 {
		t.Errorf("Changed read %v out of the diff, want the two files it touched", files)
	}

	if files := Changed(""); files != nil {
		t.Errorf("Changed read %v out of no diff at all", files)
	}
}

// TestARunThatChangedNothingSaysSo rather than drawing an empty block with a
// count of zero beside it.
func TestARunThatChangedNothingSaysSo(t *testing.T) {
	if got := text(Overview(world(t, nil))); !strings.Contains(got, "no working tree modifications") {
		t.Errorf("a run that touched nothing drew:\n%s", got)
	}
}

// TestAFoldedSectionShowsItsHeadAndWhatItIsHolding. Folding is what a reader
// does to a screen they come back to: the phases matter while a run is going
// and the deliver keys matter when it is over.
func TestAFoldedSectionShowsItsHeadAndWhatItIsHolding(t *testing.T) {
	e := world(t, nil)
	e.Diff = aDiff
	e.Folded = func(key string) bool { return key == FoldChanges }

	got := text(Overview(e))
	if strings.Contains(got, "store/items_test.go") {
		t.Errorf("a folded changes block still lists its files:\n%s", got)
	}

	// The count rides on the head, or a closed block says nothing about what
	// is inside it.
	if !strings.Contains(got, "+3") {
		t.Errorf("the folded head does not say what it is holding:\n%s", got)
	}
}

// TestTheCheckoutIsWrittenFromItsEnd. Which checkout this is is the useful
// half of the path; whose machine it sits on the reader knows.
func TestTheCheckoutIsWrittenFromItsEnd(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("this machine has no home directory: %v", err)
	}

	e := world(t, nil)
	e.Task.RepoPath = filepath.Join(home, "checkouts", "acme")

	if got := text(Overview(e)); !strings.Contains(got, "~") {
		t.Errorf("a checkout under the reader's home is drawn in full:\n%s", got)
	}

	// And one too long for the pane keeps its last segments.
	e.Task.RepoPath = "/" + strings.Repeat("a-long-directory-name/", 12) + "acme"

	if got := text(Overview(e)); !strings.Contains(got, "…") {
		t.Errorf("a path too long for the pane was not cut:\n%s", got)
	}
}

// TestTheDialsSayWhatTheRunWouldBeMadeUnder, and a window whose engines port
// answers nothing names no engine rather than one it does not have.
func TestTheDialsSayWhatTheRunWouldBeMadeUnder(t *testing.T) {
	e := world(t, nil)
	e.Dials = Dials{Engine: "claude", Model: "opus", Effort: "high", Thinking: "adaptive"}

	got := text(Overview(e))
	for _, want := range []string{"ENGINE [k]", "claude opus", "EFFORT [E]", "THINKING [t]", "adaptive"} {
		if !strings.Contains(got, want) {
			t.Errorf("the dials do not carry %q:\n%s", want, got)
		}
	}

	// With nothing set, the defaults are named rather than left blank.
	if got := text(Overview(world(t, nil))); !strings.Contains(got, "high") {
		t.Errorf("a task with no effort set draws no effort at all:\n%s", got)
	}
}
