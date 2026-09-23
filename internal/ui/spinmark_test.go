package ui

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/panes"
)

// TestThePhaseInFlightTurnsOnTheTree. Elio asked for the spinner beside
// what is in progress. The tree is built when the record changes, so the
// frame is swapped in as it is drawn, and it moves with the clock.
func TestThePhaseInFlightTurnsOnTheTree(t *testing.T) {
	m, _ := testModel(t, 120, 40)

	id := runningTask(t, m)
	m = onto(t, m, id)

	task, ok := m.task(id)
	if !ok {
		t.Fatalf("%s is not on the board", id)
	}

	m, _ = m.openDetail(task)
	m = m.showTab(tabFlow).syncPanes()

	if !m.moving() {
		t.Skip("the fixture's running task is not inside a phase, so nothing turns")
	}

	drawn := strings.Join(m.paneRows(30, 118), "\n")
	if strings.Contains(drawn, panes.SpinMark) {
		t.Errorf("the tree drew the still mark rather than a frame of the spinner:\n%s", drawn)
	}

	if !strings.Contains(drawn, m.spin()) {
		t.Errorf("the tree carries no frame of the spinner:\n%s", drawn)
	}
}
