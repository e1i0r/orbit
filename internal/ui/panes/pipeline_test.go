package panes

// The flow tree: where each phase got to, and the verbs asked for by hand
// under it.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/view"
)

// TestEachAskIsClosedByItsOwnAnswer. Two verbs can be out at once — the
// window waits on one at a time, but a run of the same task, or a second
// cockpit, is not asked — so an answer pairs with the last ask of that verb
// still open, and a verb asked for twice is two rows.
func TestEachAskIsClosedByItsOwnAnswer(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "FIX CHECKS", At: ago(3 * time.Minute)},
		{Kind: "deliver.answered", Verb: "FIX CHECKS", Text: "green", At: ago(2 * time.Minute)},
		{Kind: "deliver.asked", Verb: "FIX CHECKS", At: ago(time.Minute)},
	})

	steps := e.byHand()
	if len(steps) != 2 {
		t.Fatalf("read %d steps, want one per ask", len(steps))
	}

	if !steps[0].done || steps[1].done {
		t.Errorf("steps = %+v, want the first answered and the second still out", steps)
	}
}

// TestTheVerbStillOutIsTheOneTheBandSays. The band and the deliver block ask
// the same question, from the same call: a verb out is the one thing about a
// task that is happening somewhere the reader cannot see.
func TestTheVerbStillOutIsTheOneTheBandSays(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "PR", By: "supervisor", At: ago(2 * time.Minute)},
		{Kind: "deliver.answered", Verb: "PR", Text: "opened", At: ago(time.Minute)},
		{Kind: "deliver.asked", Verb: "RESOLVE COMMENTS", By: "supervisor", At: ago(time.Minute)},
	})

	st, out := Waiting(e)
	if !out || st.Verb != "RESOLVE COMMENTS" {
		t.Fatalf("the verb still out is %+v (%v), want RESOLVE COMMENTS", st, out)
	}

	row := strings.Join(HandOut(e), "\n")
	if !strings.Contains(row, "RESOLVE COMMENTS") || !strings.Contains(row, "supervisor") {
		t.Errorf("the deliver block says %q, want the verb and what has it", row)
	}

	// And with everything answered there is no row at all.
	done := world(t, []view.Entry{
		{Kind: "deliver.asked", Verb: "PR", At: ago(2 * time.Minute)},
		{Kind: "deliver.answered", Verb: "PR", Text: "opened", At: ago(time.Minute)},
	})

	if _, still := Waiting(done); still {
		t.Error("a task whose verbs all came back still reads as waiting on one")
	}
}

// TestALoopThatRanIsNotPending. A loop runs no engine of its own — its
// phases do — so it writes no phase.started and no phase.finished, and it
// read as pending both while it was going round and after it had closed.
func TestALoopThatRanIsNotPending(t *testing.T) {
	f, err := flow.Resolve(nil, "coverage")
	if err != nil {
		t.Skipf("this build has no coverage flow: %v", err)
	}

	entries := []view.Entry{
		{Kind: "phase.started", Phase: "1-implement"},
		{Kind: "phase.finished", Phase: "1-implement"},
		{Kind: "loop.checked", Phase: "2-until-it-passes"},
		{Kind: "phase.waiting", Phase: "3-review"},
	}

	// While it is going round: something of its own to say.
	looping := world(t, entries[:3])
	if ex := looping.execOf("2-until-it-passes"); !ex.checked {
		t.Error("a loop that wrote a turn down is not marked as having gone round")
	}

	// And once the run has moved past it: done.
	e := world(t, entries)
	e.Flow = f

	if !e.pastPhase(f, 1) {
		t.Error("a loop the run has gone past reads as unfinished")
	}

	if e.pastPhase(f, 2) {
		t.Error("the phase the run is waiting in reads as past")
	}
}

// TestAFlowThatCouldNotBeResolvedSaysSo, rather than drawing a tree with no
// phases in it.
func TestAFlowThatCouldNotBeResolvedSaysSo(t *testing.T) {
	e := world(t, nil)
	e.FlowFailed = `flow "nope": no such flow`

	got := strings.Join(first(Pipeline(e)), "\n")
	if !strings.Contains(got, "no such flow") {
		t.Errorf("the pane drew %q, want the reason the flow could not be read", got)
	}
}
