package ui

// diffstate_test.go is the diff pane's three states, and the one word in
// the tab strip that says which comparison a reader is looking at.
//
// It is a file of its own for the reason gitdiff.go is: these are questions
// about what the pane says once it has an answer, or before it does, or
// when the answer is a failure — not about which file a line of the diff
// belongs to, which diffwalk_test.go owns, and not about bounding the
// subprocess that produces the answer, which gitrepo_test.go does with a
// real git on disk.
//
// The clock the diff rides is here too, for the same reason: how often the
// pane asks, and what it does with a tick that arrives while the last answer
// is still out, are questions about the pane's state and not about git.

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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

// TestARescanDoesNotPileOnADiffThatIsStillOut is the other half of putting
// the diff on a clock: the clock is faster than the work it starts.
//
// One diffOf can cost the base lookup's two seconds and the diff's five, and
// the rescan comes round every two. A tick that asked again regardless would
// have six of them out at once against exactly the repository that is slow
// enough to need the bounds — each with a git under it, and the first one
// with a goroutine that cannot be cancelled. So a tick that finds one still
// out does not ask, and the flag that says so is cleared by the answer
// rather than by another tick, or the pane would go quiet for good.
func TestARescanDoesNotPileOnADiffThatIsStillOut(t *testing.T) {
	m := openOn(t, "ACME-2662")
	asked, cmd := m.Update(rescanMsg(fixtureNow))

	m = asModel(t, asked)
	if !m.diffAsking || len(commandsIn(t, cmd)) != 3 {
		t.Fatalf("the first rescan asked for %d commands with diffAsking=%v, want the diff among three",
			len(commandsIn(t, cmd)), m.diffAsking)
	}

	again, cmd := m.Update(rescanMsg(fixtureNow))

	m = asModel(t, again)
	if n := len(commandsIn(t, cmd)); n != 2 {
		t.Errorf("a second rescan asked for %d commands while the first diff was still out, want the rescan and the tick alone", n)
	}

	if !m.diffAsking {
		t.Error("the second rescan cleared the outstanding diff without an answer having landed")
	}
	// The answer releases it. A flag that latched would leave the pane on
	// the diff it opened with for as long as the reader stayed on it.
	m = next(t, m, diffMsg{ID: "ACME-2662", Text: fixtureDiff})
	if m.diffAsking {
		t.Fatal("a diff that landed left the view still believing one was out")
	}

	after, cmd := m.Update(rescanMsg(fixtureNow))
	if n := len(commandsIn(t, cmd)); n != 3 || !asModel(t, after).diffAsking {
		t.Errorf("the rescan after an answer asked for %d commands, want the diff asked for again", n)
	}
}

// TestTheBaseComesBackAndStays is the once-per-open half. What the base
// lookup costs is three git subprocesses through a helper that takes no
// context and cannot be killed; what it answers does not change while a
// reader looks at one task. So it rides back on the diffMsg and is held.
func TestTheBaseComesBackAndStays(t *testing.T) {
	m := openOn(t, "ACME-2662")

	m = next(t, m, diffMsg{ID: "ACME-2662", Text: fixtureDiff, Base: baseRef{name: "main", known: true}})
	if m.diffBase.name != "main" || !m.diffBase.known {
		t.Fatalf("the view kept base %+v, want the one the diff came back with", m.diffBase)
	}
	// And an open forgets it: the next task is in some other repository,
	// and a base carried across would be a branch from the wrong one.
	reopened, _ := m.openDetail(m.subject())
	if reopened.diffBase.known {
		t.Errorf("opening a task view kept base %+v from the one before it", reopened.diffBase)
	}
}

// asModel is the type assertion every transition makes, written once.
func asModel(t *testing.T, m tea.Model) Model {
	t.Helper()

	got, ok := m.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", m)
	}

	return got
}

// commandsIn unpacks a batch without running any of it. One of the commands
// under a rescan is the next rescan tick, and running that one would sit for
// board.RescanEvery.
func commandsIn(t *testing.T, cmd tea.Cmd) tea.BatchMsg {
	t.Helper()

	if cmd == nil {
		return nil
	}

	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("the transition returned %T, want a batch of commands", cmd())
	}

	return batch
}
