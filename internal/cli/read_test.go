package cli

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// writeTask writes one task down and gives back the repository it is
// against, so a test of a verb does not repeat the commands that come before
// it. It is not called `task`: internal/task is imported by nearly every file
// in this package, and a helper by that name shadows it package-wide.
func writeTask(t *testing.T, root string) string {
	t.Helper()

	dir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", dir, "-id", "ACME-1", "make the numbers add up"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	return dir
}

func TestReadWritesDownThatSomebodyLooked(t *testing.T) {
	root, _ := workspace(t)
	dir := writeTask(t, root)

	code, out, errOut := run(t, "read", "-repo", dir, "ACME-1")
	if code != 0 {
		t.Fatalf("read exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "ACME-1") {
		t.Errorf("read said %q, which does not name the task", out)
	}

	kinds := recorded(t, dir, "ACME-1")
	if !slices.Contains(kinds, "task.read") {
		t.Errorf("the record does not say the task was read: %v", kinds)
	}
}

func TestReadNeedsAnID(t *testing.T) {
	root, _ := workspace(t)
	dir := filepath.Join(root, "payments")

	code, _, errOut := run(t, "read", "-repo", dir)
	if code == 0 {
		t.Fatal("read with no id exited 0")
	}

	if !strings.Contains(errOut, "id of a task") {
		t.Errorf("the refusal is %q, and does not say what is missing", errOut)
	}
}
