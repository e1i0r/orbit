package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
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
	root, orbitHome := workspace(t)
	dir := writeTask(t, root)

	code, out, errOut := run(t, "read", "-repo", dir, "ACME-1")
	if code != 0 {
		t.Fatalf("read exited %d: %s", code, errOut)
	}
	if !strings.Contains(out, "ACME-1") {
		t.Errorf("read said %q, which does not name the task", out)
	}
	body, err := os.ReadFile(findFile(t, orbitHome, "events.jsonl"))
	if err != nil {
		t.Fatalf("read the log: %v", err)
	}
	if !strings.Contains(string(body), `"task.read"`) {
		t.Errorf("the log does not say the task was read:\n%s", body)
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

// TestUnreadCountsOnlyFinishedWorkNobodyHasLookedAt walks the four bands and
// the two exemptions. The rows are built by hand, which is exactly what
// view.BandOf's exported field is for.
func TestUnreadCountsOnlyFinishedWorkNobodyHasLookedAt(t *testing.T) {
	for _, tc := range []struct {
		name  string
		tasks []view.Task
		want  int
	}{
		{"an empty board", nil, 0},
		{"one finished task nobody read", []view.Task{{Band: view.Done}}, 1},
		{"one finished task somebody read", []view.Task{{Band: view.Done, Read: true}}, 0},
		{
			"a cancelled run is not homework",
			[]view.Task{{Band: view.Done, Reason: view.Reason{Key: view.ReasonCancelled}}},
			0,
		},
		{
			"a failure is already in front of the reader",
			[]view.Task{{Band: view.NeedsYou, Reason: view.Reason{Key: view.ReasonFailed}}},
			0,
		},
		{"nothing has run yet", []view.Task{{Band: view.ToDo}, {Band: view.Running}}, 0},
		{
			"three finished and one of them read",
			[]view.Task{
				{Band: view.Done},
				{Band: view.Done, Read: true},
				{Band: view.Done},
				{Band: view.Done, Reason: view.Reason{Key: view.ReasonCancelled}},
			},
			2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := board.Unread(board.Board{Tasks: tc.tasks}); got != tc.want {
				t.Errorf("Unread is %d, want %d", got, tc.want)
			}
		})
	}
}
