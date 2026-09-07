package ui

// What a key does, in the words ? answers with, and the two gestures that
// move around a diff.

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/words"
)

// TestEveryVerbHasASentenceAndNoTwoAreTheSame. The sheet and the ? key both
// read these, and two verbs that answer alike are two keys a reader cannot
// tell apart.
func TestEveryVerbHasASentenceAndNoTwoAreTheSame(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	said := map[string]string{}

	for _, b := range m.keys.TaskVerbs() {
		k := firstKey(b)

		got := m.meaning(k)
		if strings.TrimSpace(got) == "" {
			t.Errorf("%q is a verb with nothing said about it", k)
			continue
		}

		if was, seen := said[got]; seen {
			t.Errorf("%q and %q are answered with the same sentence", was, k)
		}

		said[got] = string(k)
	}
}

// TestAKeyNothingIsBoundToStillGetsAnAnswer, because that it does nothing is
// what the reader asked about.
func TestAKeyNothingIsBoundToStillGetsAnAnswer(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	if got := m.meaning(keystroke("§")); strings.TrimSpace(got) == "" {
		t.Error("a key nothing is bound to is answered with nothing at all")
	}
}

// TestTheKeysThatOpenAScreenSayWhichOne.
func TestTheKeysThatOpenAScreenSayWhichOne(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	for _, c := range []struct{ b, want string }{
		{m.keys.Supervisor.Help().Key, "supervisor"},
		{m.keys.Quota.Help().Key, "engine"},
		{m.keys.Knowledge.Help().Key, "orbit"},
	} {
		if got := strings.ToLower(m.meaning(keystroke(c.b))); !strings.Contains(got, c.want) {
			t.Errorf("%q is answered with %q, want it to mention %q", c.b, got, c.want)
		}
	}
}

// TestTheDiffJumpsGoHunkToHunk. A diff of forty files is not read a row at a
// time, and the hunk header is the only landmark it has.
func TestTheDiffJumpsGoHunkToHunk(t *testing.T) {
	m, _ := openIn(t, words.For("en"), "ACME-2662", fixtureEntries(), longDiff())
	m = m.showTab(tabDiff)

	rows := strings.Split(m.diff, "\n")

	next := m.jumpNextDiffHunk()

	landed := next.panes[tabDiff].YOffset()
	if landed == 0 {
		t.Fatal("the jump to the next hunk moved nothing")
	}

	// The line it landed on is a hunk header and not a row of context.
	if got := rows[landed]; !strings.HasPrefix(got, "@@") {
		t.Errorf("the jump landed on %q, want a hunk header", got)
	}

	// The second one is further down, and the jump back returns to the
	// first.
	second := next.jumpNextDiffHunk()

	at := second.panes[tabDiff].YOffset()
	if at <= landed {
		t.Fatalf("the second hunk is at row %d, from %d", at, landed)
	}

	if got := rows[at]; !strings.HasPrefix(got, "@@") {
		t.Errorf("the second jump landed on %q, want a hunk header", got)
	}

	if back := second.jumpPrevDiffHunk(); back.panes[tabDiff].YOffset() != landed {
		t.Errorf("the jump back left the pane at row %d, want the first hunk at %d",
			back.panes[tabDiff].YOffset(), landed)
	}

	// At either end there is nowhere to go, and the pane stays where it is
	// rather than scrolling off it.
	if again := second.jumpNextDiffHunk(); again.panes[tabDiff].YOffset() != at {
		t.Errorf("a jump past the last hunk moved the pane from %d to %d",
			at, again.panes[tabDiff].YOffset())
	}

	if top := next.jumpPrevDiffHunk(); top.panes[tabDiff].YOffset() != landed {
		t.Errorf("a jump back from the first hunk moved the pane to %d", top.panes[tabDiff].YOffset())
	}
}

// longDiff is two hunks with enough between them that a pane can scroll from
// one to the other.
func longDiff() string {
	var b strings.Builder

	b.WriteString("diff --git a/retry.go b/retry.go\n--- a/retry.go\n+++ b/retry.go\n")

	for i := range 2 {
		b.WriteString("@@ -" + strconv.Itoa(i*100) + ",4 +" + strconv.Itoa(i*100) + ",6 @@ func send() error {\n")

		for range 30 {
			b.WriteString(" \tcontext\n")
		}
	}

	return b.String()
}

// TestDeletingATaskIsAskedAboutFirst. Nothing brings the task or its record
// back, so the one question the window ever asks is asked here.
func TestDeletingATaskIsAskedAboutFirst(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = onto(t, m, "ACME-2662")

	asked, _ := m.askDeleteTask()

	after := asModel(t, asked)
	if after.confirm != confirmDeleteTask || after.confirmID != "ACME-2662" {
		t.Errorf("the window asks %v about %q", after.confirm, after.confirmID)
	}

	if !strings.Contains(after.message, "ACME-2662") {
		t.Errorf("the question is %q, want it to name the task", after.message)
	}

	// And n leaves the task where it is: the safe answer to a question
	// asked while somebody looked away is no.
	kept, _ := after.confirmKey(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if got := asModel(t, kept); got.confirm != confirmNone {
		t.Errorf("answering no left the question up: %v", got.confirm)
	}

	// On a band's heading there is no task to delete and nothing is asked.
	head, _ := testModel(t, 100, 30)
	head.cursor = 0

	if _, ok := head.selected(); !ok {
		if got, _ := head.askDeleteTask(); asModel(t, got).confirm != confirmNone {
			t.Error("a row with no task asked to delete one")
		}
	}
}
