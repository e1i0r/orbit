package queue

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestATaskThatWouldNotStartLeavesTheQueue. A run that fails before it
// begins writes task.failed; the queue must not start it again on every
// look, holding a slot each time.
func TestATaskThatWouldNotStartLeavesTheQueue(t *testing.T) {
	q := aQueue(t, 4, 10)

	for _, tk := range q.tasks {
		if _, err := Start(q.s, tk, "", 0); err != nil {
			t.Fatalf("start %s: %v", tk.ID, err)
		}
	}

	if err := write(q.s, q.tasks[3], record.Event{Kind: record.TaskFailed}); err != nil {
		t.Fatalf("emit: %v", err)
	}

	if q.waitingIn(t, q.tasks[3]) {
		t.Error("a task whose run failed to start is still waiting in the queue")
	}
}

// TestTheWindowRestartsAStrandedLine. Tasks waiting with no service to move
// them start one; an empty line starts nothing.
func TestTheWindowRestartsAStrandedLine(t *testing.T) {
	q := aQueue(t, 4, 10)

	if err := Ensure(q.s); err != nil {
		t.Fatalf("Ensure on an empty line: %v", err)
	}

	if q.services != 0 {
		t.Errorf("an empty line started the service %d times", q.services)
	}

	for _, tk := range q.tasks {
		if _, err := Start(q.s, tk, "", 0); err != nil {
			t.Fatalf("start %s: %v", tk.ID, err)
		}
	}

	q.services = 0

	if err := Ensure(q.s); err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	if q.services != 1 {
		t.Errorf("a waiting line started the service %d times, want 1", q.services)
	}
}
