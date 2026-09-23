package ui

import (
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// TestAVerbStillOutIsAskedAboutOnTheClock. A CREATE PR whose window had
// died read "in progress" for twenty-seven minutes. While a verb is out,
// every tick asks whether anything still carries it; while none is, the
// tick asks nothing.
func TestAVerbStillOutIsAskedAboutOnTheClock(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.detail = "ACME-1"

	var asked string

	m.opts.SettleDeliveries = func(task view.Task) error { asked = task.ID; return nil }

	if cmd := m.settle(); cmd != nil {
		t.Error("the clock asks about deliveries on a task with none out")
	}

	m.entries = []view.Entry{{Kind: "deliver.asked", Verb: "CREATE PR", By: "supervisor"}}

	cmd := m.settle()
	if cmd == nil {
		t.Fatal("a verb still out is never asked about")
	}

	cmd()

	if asked != "ACME-1" {
		t.Errorf("the port was asked about %q, want ACME-1", asked)
	}
}
