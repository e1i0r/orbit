package panes

// The overview: the task as its author wrote it, the figures, and the keys
// that move it along.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// text is the pane as a reader sees it, with the paint taken off.
func text(lines []string) string { return ansi.Strip(strings.Join(lines, "\n")) }

// TestTheOverviewOpensOnTheTaskAsWritten. The brief is on disk in task.md
// and was drawn nowhere: the pane opened on figures about a task whose own
// words the reader could not see without leaving the window.
func TestTheOverviewOpensOnTheTaskAsWritten(t *testing.T) {
	e := world(t, []view.Entry{{
		Kind: "task.created",
		Text: "Reconciliation endpoint\n\nSettlements arrive twice. Match them on the bank reference.",
	}})

	got := text(Overview(e))
	if !strings.Contains(got, "Match them on the bank reference") {
		t.Errorf("the overview does not carry the brief:\n%s", got)
	}

	// And not the title, which the header above the tab strip already
	// carries: setting it again spent the first two lines of the screen
	// saying what the line above them said.
	if strings.Contains(got, "Reconciliation endpoint") {
		t.Errorf("the pane repeats the title the header carries:\n%s", got)
	}
}

// TestALongBriefIsFoldedUntilItIsAskedFor. The brief is the reader's own
// text and a long one would push the figures, the phases and the changes off
// the first screen of every task.
func TestALongBriefIsFoldedUntilItIsAskedFor(t *testing.T) {
	e := world(t, []view.Entry{{
		Kind: "task.created",
		Text: "Reconciliation endpoint\n" + strings.Repeat("a line of the brief\n", overviewBriefRows+2) +
			"the last thing asked for",
	}})

	if got := text(Overview(e)); strings.Contains(got, "the last thing asked for") {
		t.Errorf("a closed pane set the whole brief:\n%s", got)
	}

	e.Expanded = true

	if got := text(Overview(e)); !strings.Contains(got, "the last thing asked for") {
		t.Errorf("an opened pane still withholds the end of the brief:\n%s", got)
	}
}

// TestTheDeliverActionsStandInColumns. They are laid on the grid the dials
// above them use: the verb captions on one line, the key that sends each one
// under its own caption. Joined with middots instead, each column began
// wherever the row above it happened to end, and finding a key meant reading
// both lines through.
func TestTheDeliverActionsStandInColumns(t *testing.T) {
	e := world(t, nil)

	rows := e.actions(120)
	if len(rows) < 6 {
		t.Fatalf("the deliver block drew %d lines, want a head and two rows of keys under captions", len(rows))
	}

	for _, c := range []struct {
		captions, keys int
		caption, key   string
	}{
		{1, 2, "UPDATE PR", "u"},
		{1, 2, "MERGE PR", "M"},
		{1, 2, "CLOSE PR", "X"},
		{4, 5, "MORE TESTS", "T"},
		{4, 5, "RESOLVE COMMENTS", "R"},
		{4, 5, "DEEP REVIEW", "D"},
		// The ninth and tenth verbs start a third row of their own, which is
		// what the grid does with anything past two full rows.
		{7, 8, "FEEDBACK", "a"},
		{7, 8, "DIFF", "0"},
	} {
		above, below := ansi.Strip(rows[c.captions]), ansi.Strip(rows[c.keys])

		top, bottom := strings.Index(above, c.caption), strings.Index(below, c.key)
		if top < 0 || bottom < 0 {
			t.Fatalf("rows %q / %q do not carry %q over %q", above, below, c.caption, c.key)
		}

		if top != bottom {
			t.Errorf("%q starts at cell %d and its key %q at cell %d, want one column", c.caption, top, c.key, bottom)
		}

		if strings.TrimRight(below, " ") != below {
			t.Errorf("row %q was padded past its last column", below)
		}
	}
}

// TestTheBannerNamesAKeyThatDoesSomething. A task that was abandoned is
// waiting for the reader, and the banner told them to press resume — which
// answers that resuming needs a paused task, there being no process left to
// let go of. The window had sent them to a key it refuses.
func TestTheBannerNamesAKeyThatDoesSomething(t *testing.T) {
	e := world(t, nil)

	abandoned := view.Task{
		ID: "ORB-102", Repo: "orbit", Band: view.NeedsYou,
		Reason: view.Reason{Key: view.ReasonAbandoned},
	}

	hint := e.waitingHint(abandoned)
	if strings.Contains(hint, "'"+e.Keys.Resume.Help().Key+"'") {
		t.Errorf("the banner on an abandoned task names the resume key: %q", hint)
	}

	if !strings.Contains(hint, "'"+e.Keys.Start.Help().Key+"'") {
		t.Errorf("the banner does not say how to set the task going again: %q", hint)
	}

	// And where resume is the verb — a run stopped at a phase boundary — it
	// is still the one named.
	held := view.Task{
		ID: "ORB-103", Repo: "orbit", Band: view.NeedsYou, Live: view.LiveHeld,
		Reason: view.Reason{Key: view.ReasonHeld},
	}

	if got := e.waitingHint(held); !strings.Contains(got, "'"+e.Keys.Resume.Help().Key+"'") {
		t.Errorf("the banner on a held run does not name the resume key: %q", got)
	}
}

// TestEverySectionHeadIsWhereTheHitTestLooksForIt. The heads are counted
// rather than searched for in the drawn rows, and the story was left out of
// that count: a task that wrote one put every head that many rows below
// where a click looked.
func TestEverySectionHeadIsWhereTheHitTestLooksForIt(t *testing.T) {
	e := world(t, []view.Entry{{
		Kind:  "phase.finished",
		Phase: "review",
		Story: &view.Story{
			Entry:   "POST /settlements",
			Purpose: "match what the bank sent",
			Symptom: "duplicates",
			Cause:   "no reference check",
			Fix:     "match on the bank reference",
		},
	}})

	rows := text(Overview(e))
	drawn := strings.Split(rows, "\n")

	for at, key := range FoldRows(e) {
		if at >= len(drawn) {
			t.Fatalf("the head of %q is counted at row %d and the pane drew %d rows", key, at, len(drawn))
		}

		// A head is the row with the fold arrow on it. What it says is
		// translated; that it is a head is not.
		if strings.TrimSpace(drawn[at]) == "" {
			t.Errorf("the head of %q is counted at row %d, which is blank:\n%s", key, at, rows)
		}
	}
}
