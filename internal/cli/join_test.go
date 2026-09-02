package cli

// join_test.go is the command an engine runs when the work turns out to
// need a second repository.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/task"
)

// joinable is a workspace of two repositories with a task written against
// the first, which is the situation the command exists for.
func joinable(t *testing.T) (pay string) {
	t.Helper()

	root, _ := workspace(t)
	initRepo(t, filepath.Join(root, "ledger"))

	pay = filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", pay, "-id", "ACME-1", "retry the webhook on 5xx"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	return pay
}

// TestJoinAnswersWithTheCheckoutToWorkIn. The command is run by an engine
// halfway through a phase, and what it needs back is one line it can cd
// into — not a report about what happened.
func TestJoinAnswersWithTheCheckoutToWorkIn(t *testing.T) {
	pay := joinable(t)
	t.Setenv(task.IDEnv, "ACME-1")

	code, out, errOut := run(t, "join", "-repo", pay, "ledger")
	if code != 0 {
		t.Fatalf("join exited %d: %s", code, errOut)
	}

	wt := strings.TrimSpace(out)
	if _, err := os.Stat(filepath.Join(wt, ".git")); err != nil {
		t.Errorf("join printed %q, which is not a checkout: %v", wt, err)
	}

	// And the task now reads as a task in two repositories. The event is the
	// only account of that, so a row of it that does not name the repository
	// is a task whose scope cannot be read back.
	code, shown, errOut := run(t, "show", "-repo", pay, "ACME-1")
	if code != 0 {
		t.Fatalf("show exited %d: %s", code, errOut)
	}

	if !strings.Contains(shown, "ledger") {
		t.Errorf("the record does not say ledger joined:\n%s", shown)
	}
}

// TestJoinSaysWhatItIsMissing rather than guessing it. A name it was not
// given is not a repository it can pick, and a task it was not told is not
// one it may write to.
func TestJoinSaysWhatItIsMissing(t *testing.T) {
	pay := joinable(t)

	t.Setenv(task.IDEnv, "")

	if code, _, errOut := run(t, "join", "-repo", pay); code == 0 {
		t.Error("join with no repository named exited 0")
	} else if !strings.Contains(errOut, "orbit repos") {
		t.Errorf("the refusal does not say where the names are: %s", errOut)
	}

	if code, _, errOut := run(t, "join", "-repo", pay, "ledger"); code == 0 {
		t.Error("join with no task exited 0")
	} else if !strings.Contains(errOut, task.IDEnv) {
		t.Errorf("the refusal does not name the variable a run sets: %s", errOut)
	}

	if code, _, errOut := run(t, "join", "-repo", pay, "-task", "ACME-1", "warehouse"); code == 0 {
		t.Error("join of a repository nothing answers to exited 0")
	} else if !strings.Contains(errOut, "ledger") {
		t.Errorf("the refusal does not list the names that would have worked: %s", errOut)
	}
}
