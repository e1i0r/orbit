package task

// The two ways a task is not running, which are different situations for
// whoever asked.

import (
	"strings"
	"testing"
)

// TestNothingEverClaimedItReadsDifferentlyFromSomethingThatIsGone.
//
// One of them has a next step — reconcile closes the record the dead
// process left open — and the other has none. A task nothing ever claimed
// named as having been held by process 0 sends the reader looking through a
// process table for a number that was never there.
func TestNothingEverClaimedItReadsDifferentlyFromSomethingThatIsGone(t *testing.T) {
	one := Task{ID: "ACME-1"}

	never := notRunning(one, 0).Error()
	if strings.Contains(never, "which held it") || strings.Contains(never, "reconcile") {
		t.Errorf("a task nothing ever claimed is refused with %q", never)
	}

	if !strings.Contains(never, "ACME-1") {
		t.Errorf("the refusal names no task: %q", never)
	}

	gone := notRunning(one, 4242).Error()
	for _, want := range []string{"4242", "which held it", "orbit reconcile"} {
		if !strings.Contains(gone, want) {
			t.Errorf("a task whose process is gone is refused with %q, want it to carry %q", gone, want)
		}
	}
}
