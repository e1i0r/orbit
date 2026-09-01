package cli

// readTask (read.go) and reconcile (reconcile.go) branches nothing else in
// this package reaches: a bad flag, no id, openBoth failing outside a
// repository, task.Load failing on an unknown task, task.MarkRead failing
// because the log cannot be appended to, task.List failing because the
// tasks directory cannot be read, and task.Reconcile failing on one id
// because its run marker is damaged (a pid line nothing wrote).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTaskEarlyExits(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	// 1. A flag parse failure.
	if code, _, errOut := run(t, "read", "-repo", repoDir, "-nosuchflag"); code == 0 {
		t.Error("read with an unknown flag exited 0")
	} else if errOut == "" {
		t.Error("read failed silently on a bad flag")
	}

	// 2. No id at all.
	if code, _, errOut := run(t, "read", "-repo", repoDir); code == 0 {
		t.Error("read with no id exited 0")
	} else if errOut == "" {
		t.Error("read failed silently with no id")
	}

	// 3. openBoth fails outside a repository.
	if code, _, errOut := run(t, "read", "-repo", t.TempDir(), "ACME-1"); code == 0 {
		t.Error("read outside a repository exited 0")
	} else if errOut == "" {
		t.Error("read failed silently outside a repository")
	}

	// 4. task.Load fails: a real repository, a task never written.
	if code, _, errOut := run(t, "read", "-repo", repoDir, "ACME-404"); code == 0 {
		t.Error("read on a task that was never created exited 0")
	} else if errOut == "" {
		t.Error("read failed silently on an unknown task")
	}
}

// TestReadTaskFailsOverARecordItCannotReach is what is left of a test that
// made one task's log read-only. The record is one file for every task now,
// and a command that cannot write to it cannot read it either, so the fault
// is the whole record being gone rather than one log refusing a line. What
// this still pins is that looking is written down: a read nobody could
// record is a failure and not a shrug.
func TestReadTaskFailsOverARecordItCannotReach(t *testing.T) {
	root, _ := workspace(t)
	dir := writeTask(t, root)

	breakRecord(t)

	code, _, errOut := run(t, "read", "-repo", dir, "ACME-1")
	if code == 0 {
		t.Error("read over a record nothing can reach exited 0")
	}

	if errOut == "" {
		t.Error("read failed silently over a record nothing can reach")
	}
}

func TestReconcileEarlyExitOnBadFlag(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "reconcile", "-repo", repoDir, "-nosuchflag"); code == 0 {
		t.Error("reconcile with an unknown flag exited 0")
	} else if errOut == "" {
		t.Error("reconcile failed silently on a bad flag")
	}
}

func TestReconcileFailsWhenTheTasksDirCannotBeListed(t *testing.T) {
	root, orbitHome := workspace(t)
	dir := writeTask(t, root)

	tasksDir := filepath.Dir(filepath.Dir(findFile(t, orbitHome, "task.md")))
	if err := os.Chmod(tasksDir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer func() { _ = os.Chmod(tasksDir, 0o700) }() //nolint:errcheck

	code, _, errOut := run(t, "reconcile", "-repo", dir)
	if code == 0 {
		t.Error("reconcile over an unlistable tasks directory exited 0")
	}

	if errOut == "" {
		t.Error("reconcile failed silently over an unlistable tasks directory")
	}
}

func TestReconcileReportsAPerTaskFailureAndKeepsGoing(t *testing.T) {
	root, _ := workspace(t)
	dir := writeTask(t, root)

	// A run marker nothing in this codebase wrote: task.Reconcile's call to
	// Alive fails parsing the pid line, rather than reporting the task as
	// abandoned or as still running.
	if err := os.WriteFile(markerPath(t, "ACME-1"), []byte("pid: not-a-number\nstarted: 2026-08-24T12:00:00Z\n"), 0o600); err != nil {
		t.Fatalf("write the run marker: %v", err)
	}

	code, _, errOut := run(t, "reconcile", "-repo", dir, "ACME-1")
	if code == 0 {
		t.Error("reconcile over a damaged run marker exited 0")
	}

	if !strings.Contains(errOut, "ACME-1") {
		t.Errorf("the refusal does not name the task:\n%s", errOut)
	}
}
