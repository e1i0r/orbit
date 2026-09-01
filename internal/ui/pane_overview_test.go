package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// overviewText is the pane as a reader sees it, with the paint taken off.
func overviewText(m Model) string {
	return ansi.Strip(strings.Join(m.overviewLines(), "\n"))
}

// TestTheOverviewOpensOnTheTaskAsWritten. The brief is on disk in task.md
// and was drawn nowhere: the pane opened on figures about a task whose own
// words the reader could not see without leaving the window.
func TestTheOverviewOpensOnTheTaskAsWritten(t *testing.T) {
	m := openOn(t, "ACME-2705")
	m.entries = []view.Entry{{
		Kind: "task.created",
		Text: "Reconciliation endpoint\n\nSettlements arrive twice. Match them on the bank reference.",
	}}

	if got := overviewText(m); !strings.Contains(got, "Match them on the bank reference") {
		t.Errorf("the overview does not carry the brief:\n%s", got)
	}
}

// TestTheTitleIsNotSetTwiceOnOneScreen. The header above the tab strip names
// the task; setting it again at the top of the pane spent the first two
// lines of the screen saying what the line above them said.
func TestTheTitleIsNotSetTwiceOnOneScreen(t *testing.T) {
	m := openOn(t, "ACME-2705")
	m.entries = []view.Entry{{
		Kind: "task.created",
		Text: "Reconciliation endpoint\n\nSettlements arrive twice.",
	}}

	head := ansi.Strip(strings.Join(m.detailHeadLines(m.frame.Body.W), "\n"))
	if !strings.Contains(head, "Reconciliation endpoint") {
		t.Fatalf("the header stopped naming the task: %q", head)
	}

	if got := overviewText(m); strings.Contains(got, "Reconciliation endpoint") {
		t.Errorf("the pane repeats the title the header carries:\n%s", got)
	}
}

// TestATaskWithoutABriefIsNamedOnce. A one-line task.md has a title and
// nothing under it. The header names it, as it does for every other task,
// and the pane does not answer a missing brief by setting that title again.
func TestATaskWithoutABriefIsNamedOnce(t *testing.T) {
	m := openOn(t, "ACME-2705")
	m.entries = []view.Entry{{Kind: "task.created", Text: "Reconciliation endpoint"}}

	head := ansi.Strip(strings.Join(m.detailHeadLines(m.frame.Body.W), "\n"))
	if !strings.Contains(head, "Reconciliation endpoint") {
		t.Fatalf("the header does not name a task with no brief: %q", head)
	}

	if got := overviewText(m); strings.Contains(got, "Reconciliation endpoint") {
		t.Errorf("the pane sets the title the header already carries:\n%s", got)
	}
}

// TestALongBriefIsFoldedUntilItIsAskedFor. The brief is the reader's own
// text and a long one would push the figures, the phases and the changes off
// the first screen of every task.
func TestALongBriefIsFoldedUntilItIsAskedFor(t *testing.T) {
	m := openOn(t, "ACME-2705")
	m.entries = []view.Entry{{
		Kind: "task.created",
		Text: "Reconciliation endpoint\n" + strings.Repeat("a line of the brief\n", overviewBriefRows+2) +
			"the last thing asked for",
	}}

	if got := overviewText(m); strings.Contains(got, "the last thing asked for") {
		t.Errorf("a closed pane set the whole brief:\n%s", got)
	}

	m.expandedDetail = true

	if got := overviewText(m); !strings.Contains(got, "the last thing asked for") {
		t.Errorf("an opened pane still withholds the end of the brief:\n%s", got)
	}
}

// TestTheHeaderKeepsTheRepositoryWhateverTheTitleIs. spread gives up its
// right-hand side entirely when both halves will not fit, so a first line of
// task.md that runs to a paragraph took the repository and the state with
// it — and printed the marks it was written with besides.
func TestTheHeaderKeepsTheRepositoryWhateverTheTitleIs(t *testing.T) {
	tasks := fixtureTasks()
	tasks[0].Title = "Make `retry.go` back off, and **then** say so in the log, at " +
		"such length that the line runs past the edge of any terminal it is drawn in"

	m := modelWith(t, words.For("en"), fixtureBoard(tasks, 4), 100, 30, &recorder{})
	m.screen, m.detail = screenDetail, tasks[0].ID

	head := ansi.Strip(strings.Join(m.detailHeadLines(100), "\n"))

	for _, mark := range []string{"`", "**"} {
		if strings.Contains(head, mark) {
			t.Errorf("the header prints the %s the title was written with: %q", mark, head)
		}
	}

	if !strings.Contains(head, tasks[0].Repo) {
		t.Errorf("a long title pushed the repository off the header: %q", head)
	}
}

// The two action rows are a table: whatever the labels of the first row are
// as long as, the keys of the second row start under the keys of the first.
// Joined with middots instead, each column began wherever the row above it
// happened to end, and finding a key meant reading both lines through.
func TestTheDeliverActionsStandInColumns(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	rows := m.overviewActions(120)
	if len(rows) < 3 {
		t.Fatalf("overviewActions drew %d lines, want a head and two rows", len(rows))
	}

	first, second := ansi.Strip(rows[1]), ansi.Strip(rows[2])
	for _, pair := range [][2]string{{"u ", "T "}, {"M ", "a "}, {"X ", "0 "}} {
		top, bottom := strings.Index(first, pair[0]), strings.Index(second, pair[1])
		if top < 0 || bottom < 0 {
			t.Fatalf("rows %q / %q do not both carry %v", first, second, pair)
		}

		if top != bottom {
			t.Errorf("%q starts at cell %d and %q at cell %d, want one column", pair[0], top, pair[1], bottom)
		}
	}

	if strings.TrimRight(second, " ") != second {
		t.Errorf("row %q was padded past its last column", second)
	}
}
