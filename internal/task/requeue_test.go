package task

// Taking a task back: the run stops, and the record says the task is going
// to be started again rather than that it is over.

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestRequeueStopsWhatIsRunningBeforeItSaysTheTaskIsBack is the ordering the
// whole verb turns on. The event has to be the last thing in the record, and
// it can only be that if the run that would write task.cancelled on its way
// out is gone before it is written.
func TestRequeueStopsWhatIsRunningBeforeItSaysTheTaskIsBack(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "REQ-1", "wrong brief", "quick")
	if err != nil {
		t.Fatal(err)
	}

	pid, gone := child(t)
	if _, err := mark(s, tk, pid); err != nil {
		t.Fatalf("mark: %v", err)
	}

	if err := Requeue(context.Background(), s, tk, "operator", "wrong engine"); err != nil {
		t.Fatalf("Requeue on a task a process holds: %v", err)
	}

	select {
	case <-gone:
	case <-time.After(5 * time.Second):
		t.Fatal("Requeue came back with the run it was supposed to stop still running")
	}

	last := lastEvent(t, s, tk)
	if last.Kind != "task.requeued" {
		t.Errorf("the record ends with %q, want task.requeued — anything after it is the state the board reads", last.Kind)
	}

	if last.Data["by"] != "operator" || !strings.Contains(last.Text, "wrong engine") {
		t.Errorf("the record says %q by %q, want the reason and who gave it", last.Text, last.Data["by"])
	}
}

// A task nothing is running is requeued by the same call, with no signal
// sent to anybody. Returning a failed task to the queue is the same gesture
// as returning a live one, and a caller that had to know which case it was
// in would be a caller keeping its own copy of Alive.
func TestRequeueTakesBackATaskNoProcessHolds(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "REQ-2", "nothing is running", "quick")
	if err != nil {
		t.Fatal(err)
	}

	if err := Requeue(context.Background(), s, tk, "", ""); err != nil {
		t.Fatalf("Requeue on a task nothing holds: %v", err)
	}

	last := lastEvent(t, s, tk)
	if last.Kind != "task.requeued" {
		t.Fatalf("the record ends with %q, want task.requeued", last.Kind)
	}

	// Nobody said who or why, and neither is invented: an empty "by" in the
	// record would be a field the reader has to know is meaningless.
	if _, said := last.Data["by"]; said || last.Text != "" {
		t.Errorf("the record carries %q by %q, want neither said", last.Text, last.Data["by"])
	}
}
