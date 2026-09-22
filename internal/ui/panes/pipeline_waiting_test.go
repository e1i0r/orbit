package panes

// A phase that stopped at a gate and then went on is not still at the gate.
//
// The flag was written once and never taken back. FRA-128 ran all three of
// its phases and its tree read implement completed, review waiting at
// gate, fix completed: a run visibly past the thing it was said to be
// stuck on, with the task itself marked finished above it.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// TestAPhaseThatWasResumedIsNotStillWaiting walks the record FRA-128 wrote,
// in order, and asks what the tree would draw at each point.
func TestAPhaseThatWasResumedIsNotStillWaiting(t *testing.T) {
	// The gate fires before the phase runs, so every one of these carries
	// the name of the phase the run is about to enter.
	record := []view.Entry{
		{Kind: "phase.waiting", Phase: "review", At: ago(30 * time.Minute)},
		{Kind: "phase.resumed", Phase: "review", At: ago(10 * time.Minute)},
		{Kind: "phase.started", Phase: "review", At: ago(9 * time.Minute)},
		{Kind: "phase.finished", Phase: "review", At: ago(6 * time.Minute)},
	}

	for _, c := range []struct {
		upTo int
		want bool
		why  string
	}{
		{1, true, "parked at the gate and nothing since"},
		{2, false, "a person let it go"},
		{3, false, "it is running"},
		{4, false, "it ran through"},
	} {
		e := world(t, record[:c.upTo])

		if got := e.execOf("review").waiting; got != c.want {
			t.Errorf("after %d entries (%s) waiting = %v, want %v",
				c.upTo, c.why, got, c.want)
		}
	}
}

// TestAResumedPhaseDrawsAsCompleted is the same fact where a reader meets
// it: the line in the tree.
func TestAResumedPhaseDrawsAsCompleted(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "phase.waiting", Phase: "review", At: ago(30 * time.Minute)},
		{Kind: "phase.resumed", Phase: "review", At: ago(10 * time.Minute)},
		{Kind: "phase.started", Phase: "review", At: ago(9 * time.Minute)},
		{Kind: "phase.finished", Phase: "review", At: ago(6 * time.Minute)},
	})

	st := e.phaseStanding(e.execOf("review"), where{past: true})

	if !strings.Contains(st.text, "completed") {
		t.Errorf("a phase that waited, was let go and ran through draws %q, want completed",
			st.text)
	}
}

// TestAPhaseStillAtItsGateSaysSo is the other direction, so the fix above
// cannot be "never say waiting".
func TestAPhaseStillAtItsGateSaysSo(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "phase.waiting", Phase: "review", At: ago(2 * time.Minute)},
	})

	st := e.phaseStanding(e.execOf("review"), where{})

	if !strings.Contains(st.text, "waiting") {
		t.Errorf("a phase parked at its gate draws %q, want it to say it is waiting", st.text)
	}
}
