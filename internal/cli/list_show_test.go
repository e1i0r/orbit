package cli

// list() and show() branches nothing else in this package reaches: list on
// a repository with no tasks at all, list when the tasks directory cannot be
// read, show when openBoth fails, show on an id that names no task at all
// (task.Events answers no error and no events, which show tells apart from
// a damaged log by its own "nothing recorded" refusal).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListSaysSoWhenThereAreNoTasks(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	code, out, errOut := run(t, "list", "-repo", repoDir)
	if code != 0 {
		t.Fatalf("list on an empty repository exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "no tasks against") {
		t.Errorf("list did not say there were no tasks:\n%s", out)
	}
}

func TestListFailsWhenTheTasksDirCannotBeRead(t *testing.T) {
	root, orbitHome := workspace(t)
	dir := writeTask(t, root)

	tasksDir := filepath.Dir(filepath.Dir(findFile(t, orbitHome, "task.md")))
	if err := os.Chmod(tasksDir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer func() { _ = os.Chmod(tasksDir, 0o700) }() //nolint:errcheck

	code, _, errOut := run(t, "list", "-repo", dir)
	if code == 0 {
		t.Error("list over an unreadable tasks directory exited 0")
	}

	if errOut == "" {
		t.Error("list failed silently over an unreadable tasks directory")
	}
}

func TestShowFailsOutsideARepository(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	code, _, errOut := run(t, "show", "-repo", t.TempDir(), "ACME-1")
	if code == 0 {
		t.Error("show outside a repository exited 0")
	}

	if errOut == "" {
		t.Error("show failed silently outside a repository")
	}
}

func TestShowRefusesATaskWithNothingRecorded(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	code, _, errOut := run(t, "show", "-repo", repoDir, "ACME-GHOST")
	if code == 0 {
		t.Error("show on a task with nothing recorded exited 0")
	}

	if !strings.Contains(errOut, "nothing recorded") {
		t.Errorf("show did not say nothing was recorded:\n%s", errOut)
	}
}

// TestShowFailsWhenTaskEventsErrors covers the branch above the "nothing
// recorded" refusal: task.Events itself returning an error rather than an
// empty log. An id containing a path separator is refused by store's own
// validateTaskID before any file is touched, which is the cheapest way to
// make task.Events answer (nil, error) rather than (nil, nil).
func TestShowFailsWhenTaskEventsErrors(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	code, _, errOut := run(t, "show", "-repo", repoDir, "bad/id")
	if code == 0 {
		t.Error("show with a path-separator id exited 0")
	}

	if strings.Contains(errOut, "nothing recorded") {
		t.Errorf("show took the empty-log branch instead of the task.Events error:\n%s", errOut)
	}
}
