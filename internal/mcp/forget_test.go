package mcp

// Forgetting a repository, against a task worked in one and a task worked in
// two.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestForgettingARepositoryReallyDeletesTheTasksItSaysItDeletes. The answer
// carried tasks_deleted from the shape where a task lived under the
// repository it was written in, so removing the directory removed it. The
// record moved to the root of the state tree and the count went on being
// reported for a task that was still on the board — a caller reads a
// cleanup that did not happen and does not look again.
func TestForgettingARepositoryReallyDeletesTheTasksItSaysItDeletes(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})

	got := call(t, sn, "orbit_forget_repo", map[string]any{"repo": "payments", "delete_tasks": true})
	if got["tasks_deleted"] != float64(1) {
		t.Fatalf("orbit_forget_repo answered %v, want 1 task deleted", got["tasks_deleted"])
	}

	if said := refused(t, sn, "orbit_inspect_task", map[string]any{"task_id": "PAY-1"}); !strings.Contains(said, "PAY-1") {
		t.Errorf("PAY-1 was reported deleted and is still there: %s", said)
	}
}

// TestATaskCarriedIntoAnotherCheckoutSurvivesForgettingTheFirst. The refusal
// is about what would be lost, and a task still being worked in a second
// repository loses nothing. Deleting it because one of its repositories was
// forgotten would take work nobody asked about off the board.
func TestATaskCarriedIntoAnotherCheckoutSurvivesForgettingTheFirst(t *testing.T) {
	s, work := newRoot(t)
	payments := gitRepo(t, work, "payments")
	ledger := gitRepo(t, work, "ledger")
	sn := Session{Root: work, Version: "test"}

	addTask(t, s, payments, "PAY-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written in payments"})
	addTask(t, s, ledger, "PAY-1")

	// No delete_tasks: nothing here would be lost, so nothing is refused.
	got := call(t, sn, "orbit_forget_repo", map[string]any{"repo": "payments"})
	if got["tasks_deleted"] != float64(0) {
		t.Errorf("forgetting payments deleted %v tasks, want none", got["tasks_deleted"])
	}

	kept, ok := got["tasks_kept"].([]any)
	if !ok || len(kept) != 1 || kept[0] != "PAY-1" {
		t.Errorf("the answer says %v was kept, want PAY-1", got["tasks_kept"])
	}

	if found := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "PAY-1"}); found["id"] != "PAY-1" {
		t.Errorf("PAY-1 went with the repository it was only half worked in: %v", found)
	}
}
