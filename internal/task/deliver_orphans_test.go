package task

import (
	"os"
	"strconv"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestADeliveryWhoseWindowClosedIsClosedToo. ORB-121's CREATE PR was handed
// to the supervisor, the window carrying it was closed thirteen seconds
// later, and the tree said "in progress" for as long as anybody looked.
// A window opening closes what a dead window left open, and leaves alone
// what a live one is still carrying.
func TestADeliveryWhoseWindowClosedIsClosedToo(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-81", "open the pull request", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	for _, e := range []record.Event{
		// Written by a build that did not say which window carried it.
		{Kind: record.DeliverAsked, Data: map[string]string{"verb": "CREATE PR", "by": "supervisor"}},
		// Carried by this process, which is alive.
		{Kind: record.DeliverAsked, Data: map[string]string{
			"verb": "FIX CHECKS", "by": "supervisor", "pid": strconv.Itoa(os.Getpid()),
		}},
	} {
		if err := emit(s, tk, e); err != nil {
			t.Fatalf("emit: %v", err)
		}
	}

	closed, err := ReconcileDeliveries(s, tk)
	if err != nil {
		t.Fatalf("ReconcileDeliveries: %v", err)
	}

	if closed != 1 {
		t.Errorf("closed %d deliveries, want the one nothing carries any more", closed)
	}

	events := mustEvents(t, s, tk)

	last := events[len(events)-1]
	if last.Kind != record.DeliverAnswered || last.Data["verb"] != "CREATE PR" || last.Data["error"] == "" {
		t.Errorf("the record ends %+v, want CREATE PR answered as broken", last)
	}

	again, err := ReconcileDeliveries(s, tk)
	if err != nil {
		t.Fatalf("ReconcileDeliveries, again: %v", err)
	}

	if again != 0 {
		t.Errorf("a second window closed %d more, want none", again)
	}
}

// TestAStepOfADeliveryIsWrittenOntoItsTask. What the supervisor does while
// a verb is out is what the reader watches, so each step is a line of the
// task's record under the verb it is for.
func TestAStepOfADeliveryIsWrittenOntoItsTask(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-82", "open the pull request", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := DeliveryStep(s, tk, "CREATE PR", "Bash", `{"command":"gh pr create"}`); err != nil {
		t.Fatalf("DeliveryStep: %v", err)
	}

	events := mustEvents(t, s, tk)

	got := events[len(events)-1]
	if got.Kind != record.DeliverStep || got.Data["verb"] != "CREATE PR" || got.Data["tool"] != "Bash" {
		t.Errorf("the record ends %+v, want the step under CREATE PR", got)
	}
}
