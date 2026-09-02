package ui

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
	m, _ := onTab(t, tabCost, []view.Entry{
		{Kind: "phase.started", Phase: "implement", At: ago(3 * time.Minute), Engine: "claude", Model: "opus"},
		{Kind: "phase.retried", Phase: "implement", At: ago(2 * time.Minute), Gate: "build", Cost: 0.25},
		{Kind: "phase.started", Phase: "implement", At: ago(2 * time.Minute), Engine: "claude", Model: "opus"},
		{Kind: "phase.finished", Phase: "implement", At: ago(time.Minute), Cost: 0.25},
	})

	var rows int

	for _, l := range m.costLines() {
		if strings.Contains(l, "$0.25") {
			rows++
		}
	}

	if rows != 2 {
		t.Errorf("the cost tab draws %d rows at $0.25, want two: the attempt that was refused and the one that stood\n%s", rows, strings.Join(m.costLines(), "\n"))
	}
}
