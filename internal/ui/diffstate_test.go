package ui

// diffstate_test.go is the diff pane's three states, and the one word in
// the tab strip that says which comparison a reader is looking at.
//
// It is a file of its own for the reason gitdiff.go is: these are questions
// about what the pane says once it has an answer, or before it does, or
// when the answer is a failure — not about which file a line of the diff
// belongs to, which detail_render_test.go already owns, and not about
// bounding the subprocess that produces the answer, which gitrepo_test.go
// does with a real git on disk.

import (
	"errors"
	"strings"
	"testing"
)

// TestTheDiffPaneSaysNotYetBeforeItKnows is I2's first requirement: before
// the first diffMsg lands, the pane says something true of not-yet-knowing,
// never the sentence it would say once git had answered and found nothing.
func TestTheDiffPaneSaysNotYetBeforeItKnows(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = step(t, onto(t, m, "ACME-2662"), "enter")
	// Opening the task view only returns the commands that will fetch a
	// diff; none of them has run. diffKnown is still false here, the same
	// way it would be for the seconds a reader's own git actually takes.
	text := paneText(t, showing(t, m, tabDiff))
	wantIn(t, text, "reading this task's worktree")
	if strings.Contains(text, "no changes in this task's worktree") {
		t.Errorf("the diff pane asserts no changes before any diffMsg has landed:\n%s", text)
	}
}

// TestAFailedDiffIsSaidInThePaneNotSwallowed is the diff's third state: an
// answer that came back as a failure is said, in the pane a reader is
// already looking at, rather than folded into an empty diff that reads the
// same as "nothing changed".
func TestAFailedDiffIsSaidInThePaneNotSwallowed(t *testing.T) {
	m := openOn(t, "ACME-2662")
	m = next(t, m, diffMsg{ID: "ACME-2662", Err: errors.New("git did not answer in time")})
	text := paneText(t, showing(t, m, tabDiff))
	wantIn(t, text, "git did not answer in time")
}

// TestANoBaseDiffSaysSoInTheStrip is M6: a diff that fell back to the plain
// working tree because there was no base branch to compare against says so
// where the reader is already looking, rather than leaving the shape of the
// comparison to be assumed.
func TestANoBaseDiffSaysSoInTheStrip(t *testing.T) {
	m := openOn(t, "ACME-2662")
	m = next(t, m, diffMsg{ID: "ACME-2662", Text: fixtureDiff, NoBase: true})
	text := paneText(t, showing(t, m, tabDiff))
	wantIn(t, text, "no base branch")
}

// TestAnOrdinaryDiffSaysNothingAboutTheBase is the other half: NoBase's
// zero value means there was a base, and a diffMsg built the ordinary way —
// every one before this task, and most of the ones after it — must not
// start claiming a fallback that never happened.
func TestAnOrdinaryDiffSaysNothingAboutTheBase(t *testing.T) {
	m := openOn(t, "ACME-2662")
	m = next(t, m, diffMsg{ID: "ACME-2662", Text: fixtureDiff})
	text := paneText(t, showing(t, m, tabDiff))
	if strings.Contains(text, "no base branch") {
		t.Errorf("the strip says %q for an ordinary diff, want no mention of the base", text)
	}
}
