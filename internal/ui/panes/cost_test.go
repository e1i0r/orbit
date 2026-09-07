package panes

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/view"
)

// TestTheCostTabCountsAnAttemptItsGateRefused. The header totals every
// phase-ending event the record holds, phase.retried among them. A tab whose
// rows leave that attempt out is a table that does not add up to the number
// printed above it, and the money it is missing is the money a reader most
// wants to see: what the run spent getting it wrong.
func TestTheCostTabCountsAnAttemptItsGateRefused(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "phase.started", Phase: "implement", At: ago(3 * time.Minute), Engine: "claude", Model: "opus"},
		{Kind: "phase.retried", Phase: "implement", At: ago(2 * time.Minute), Gate: "build", Cost: 0.25},
		{Kind: "phase.started", Phase: "implement", At: ago(2 * time.Minute), Engine: "claude", Model: "opus"},
		{Kind: "phase.finished", Phase: "implement", At: ago(time.Minute), Cost: 0.25},
	})

	var rows int

	for _, l := range Cost(e) {
		if strings.Contains(l, "$0.25") {
			rows++
		}
	}

	if rows != 2 {
		t.Errorf("the cost tab draws %d rows at $0.25, want two: the attempt that was refused and the one that stood\n%s",
			rows, strings.Join(Cost(e), "\n"))
	}
}

// TestATaskThatLeftTheBoardSaysSoRatherThanDrawingAnEmptyTable.
func TestATaskThatLeftTheBoardSaysSoRatherThanDrawingAnEmptyTable(t *testing.T) {
	e := world(t, nil)
	e.Gone = true

	if got := strings.Join(Cost(e), "\n"); !strings.Contains(got, "no longer on the board") {
		t.Errorf("the cost pane of a task that left drew %q, want it to say the task is gone", got)
	}
}

// TestUnderASubscriptionTheColumnSaysSoRatherThanPrintingZero. $0.0000
// against a phase that ran for twenty minutes reads as "this one was free",
// which is the one thing it certainly was not.
func TestUnderASubscriptionTheColumnSaysSoRatherThanPrintingZero(t *testing.T) {
	e := world(t, []view.Entry{
		{Kind: "phase.started", Phase: "implement", At: ago(3 * time.Minute)},
		{Kind: "phase.finished", Phase: "implement", At: ago(time.Minute)},
	})
	e.Priced = false

	got := strings.Join(Cost(e), "\n")
	if strings.Contains(got, "$0.0000") {
		t.Errorf("the cost pane printed a zero for a run nobody was charged for:\n%s", got)
	}

	if !strings.Contains(got, "subscription") {
		t.Errorf("the cost pane does not say which kind of number is missing:\n%s", got)
	}
}
