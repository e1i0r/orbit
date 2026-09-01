package task

// The error returns of Reconcile that reconcile_test.go's happy paths never
// provoke: Alive itself failing, the record being out of reach, and the
// record answering the read and refusing the write.
//
// The third is the one that says the most. Reconcile decides from the events
// that a run was abandoned and then writes that down, and a fault that stops
// both halves never reaches the second one — so the record is left readable
// and made read-only, which is a thing one file can be and a log per task
// never was.

import (
	"testing"
)

// TestReconcileAliveErrorPropagates covers the first error return: Alive
// itself cannot answer.
func TestReconcileAliveErrorPropagates(t *testing.T) {
	s, r := fixture(t)

	bad := Task{ID: "has/slash", Repo: r}
	if _, err := Reconcile(s, bad); err == nil {
		t.Error("Reconcile with a slash in the id should have failed")
	}
}

// TestReconcileOverARecordItCannotReachFails covers the second error return:
// a marker names a dead process, so Reconcile goes to the record to decide
// whether the run was in flight, and the record is not there to be asked.
func TestReconcileOverARecordItCannotReachFails(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "REC-EVT-ERR-1", "reconcile events error test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	openRun(t, s, tk)

	if _, err := mark(s, tk, deadPid(t)); err != nil {
		t.Fatalf("mark: %v", err)
	}

	breakRecord(t, s)

	if _, err := Reconcile(s, tk); err == nil {
		t.Error("Reconcile over a record it could not reach should have failed")
	}
}

// TestReconcileFailsWhenTheAbandonmentCannotBeWrittenDown covers the third
// error return: everything Reconcile needs to decide is readable, it decides
// the run was abandoned, and the record will not take the event saying so.
//
// The marker is what makes this matter rather than a branch to colour in. It
// is removed only after the event lands, so a failure here leaves the task
// still claiming to be running — which is the truth until somebody can write
// down that it is not.
func TestReconcileFailsWhenTheAbandonmentCannotBeWrittenDown(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "REC-WRITE-1", "reconcile write error test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	openRun(t, s, tk)

	if _, err := mark(s, tk, deadPid(t)); err != nil {
		t.Fatalf("mark: %v", err)
	}

	readOnlyRecord(t, s)

	// The read still works, which is what makes this the write's failure and
	// not the same fault twice.
	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("read the events of a record that is only read-only: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("the record answered no events, so Reconcile would have nothing to abandon")
	}

	abandoned, err := Reconcile(s, tk)
	if err == nil {
		t.Error("Reconcile reported success over a record that would not take the event")
	}

	if abandoned {
		t.Error("Reconcile said it abandoned the run, with nothing in the record saying so")
	}

	pid, _, err := Alive(s, tk)
	if err != nil {
		t.Fatalf("read the marker back: %v", err)
	}

	if pid == 0 {
		t.Error("the run marker was taken away by a Reconcile that recorded nothing")
	}
}
