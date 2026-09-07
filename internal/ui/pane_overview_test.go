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

// TestTheNeedsYouLineNamesTheKeysThisScreenHonours.
//
// The banner read "press 't' to open interactive session", and t on the
// detail screen is the thinking dial: a reader who did as it said turned
// thinking off and got no session. The letters come from the bindings now,
// so the sentence cannot drift away from the keys again.
func TestTheNeedsYouLineNamesTheKeysThisScreenHonours(t *testing.T) {
	m := openOn(t, "ACME-2662")

	got := overviewText(m)
	if !strings.Contains(got, "NEEDS YOU") {
		t.Fatalf("the banner a task in needs you draws is not there:\n%s", got)
	}

	if strings.Contains(got, "'t'") {
		t.Errorf("the banner sends the reader to the thinking dial:\n%s", got)
	}

	// The third key is whichever sets this task going again, and this one
	// failed at a gate with no process left, so it is the start key rather
	// than resume. TestTheBannerNamesAKeyThatDoesSomething is about that
	// choice; this is about the letters coming from the bindings.
	for _, k := range []string{m.keys.CLI.Help().Key, m.keys.Ask.Help().Key, m.keys.Start.Help().Key} {
		if !strings.Contains(got, "'"+k+"'") {
			t.Errorf("the banner does not name %q, which is a key this screen honours:\n%s", k, got)
		}
	}

	// And the key it names for feedback is the key that takes it.
	if next := step(t, m, m.keys.Ask.Help().Key); !next.note.open {
		t.Errorf("%q did not open the note the banner offers", m.keys.Ask.Help().Key)
	}
}
