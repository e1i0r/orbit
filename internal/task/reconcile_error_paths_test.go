package task

// The three error returns of Reconcile that reconcile_test.go's happy paths
// never provoke: Alive itself failing, the log being unreadable, and the
// task.abandoned write failing after everything upstream succeeded.

import (
	"os"
	"strings"
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

// TestReconcileEventsUnreadablePropagates covers the second error return: a
// marker names a dead process, so Reconcile goes to read the log to decide
// whether it was in flight, and the log itself cannot be read back.
func TestReconcileEventsUnreadablePropagates(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "REC-EVT-ERR-1", "reconcile events error test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := mark(s, tk, deadPid(t)); err != nil {
		t.Fatalf("mark: %v", err)
	}

	path, err := s.EventsPath(r.Path, tk.ID)
	if err != nil {
		t.Fatalf("EventsPath: %v", err)
	}

	oversized := strings.Repeat("x", 5<<20) // over record.MaxLine (4 MiB)
	if err := os.WriteFile(path, []byte(oversized+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := Reconcile(s, tk); err == nil {
		t.Error("Reconcile over an unreadable log should have failed")
	}
}

// TestReconcileAbandonedEmitFailurePropagates covers the third error return:
// everything upstream succeeds — a dead marker over a run still open — and
// the write that closes the record fails.
func TestReconcileAbandonedEmitFailurePropagates(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "REC-EMIT-ERR-1", "reconcile emit error test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	openRun(t, s, tk)

	if _, err := mark(s, tk, deadPid(t)); err != nil {
		t.Fatalf("mark: %v", err)
	}

	// Read-only: Events (a read) still succeeds, but the task.abandoned
	// append does not.
	path, err := s.EventsPath(r.Path, tk.ID)
	if err != nil {
		t.Fatalf("EventsPath: %v", err)
	}

	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	t.Cleanup(func() { _ = os.Chmod(path, 0o600) }) //nolint:errcheck

	if _, err := Reconcile(s, tk); err == nil {
		t.Error("Reconcile that cannot write task.abandoned should have failed")
	}
}
