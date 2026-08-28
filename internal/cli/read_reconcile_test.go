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

func TestReadTaskFailsWhenTheLogCannotBeAppendedTo(t *testing.T) {
	root, orbitHome := workspace(t)
	dir := writeTask(t, root)

	events := findFile(t, orbitHome, "events.jsonl")
	if err := os.Chmod(events, 0o400); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer func() { _ = os.Chmod(events, 0o600) }() //nolint:errcheck

	code, _, errOut := run(t, "read", "-repo", dir, "ACME-1")
	if code == 0 {
		t.Error("read over a read-only log exited 0")
	}

	if errOut == "" {
		t.Error("read failed silently over a read-only log")
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
	root, orbitHome := workspace(t)
	dir := writeTask(t, root)

	// A run marker nothing in this codebase wrote: task.Reconcile's call to
	// Alive fails parsing the pid line, rather than reporting the task as
	// abandoned or as still running.
	events := findFile(t, orbitHome, "events.jsonl")

	marker := filepath.Join(filepath.Dir(events), "run")
	if err := os.WriteFile(marker, []byte("pid: not-a-number\nstarted: 2026-08-24T12:00:00Z\n"), 0o600); err != nil {
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
