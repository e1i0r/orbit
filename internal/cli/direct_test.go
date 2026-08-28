package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/store"
)

func TestDirectTaskEarlyExits(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	// 1. Bad flag
	if code, _, errOut := run(t, "direct", "-repo", repoDir, "-nosuchflag"); code == 0 {
		t.Error("direct with unknown flag exited 0")
	} else if errOut == "" {
		t.Error("direct failed silently on bad flag")
	}

	// 2. Missing task id
	if code, _, errOut := run(t, "direct", "-repo", repoDir); code == 0 {
		t.Error("direct with no task id exited 0")
	} else if !strings.Contains(errOut, "needs the id") {
		t.Errorf("unexpected error: %s", errOut)
	}

	// 3. Missing message
	if code, _, errOut := run(t, "direct", "-repo", repoDir, "ACME-1"); code == 0 {
		t.Error("direct with no message exited 0")
	} else if !strings.Contains(errOut, "needs a message") {
		t.Errorf("unexpected error: %s", errOut)
	}

	// 4. Unknown repo
	if code, _, errOut := run(t, "direct", "-repo", t.TempDir(), "ACME-1", "msg"); code == 0 {
		t.Error("direct outside repo exited 0")
	} else if errOut == "" {
		t.Error("direct failed silently outside repo")
	}

	// 5. Unknown task
	if code, _, errOut := run(t, "direct", "-repo", repoDir, "ACME-404", "msg"); code == 0 {
		t.Error("direct on unknown task exited 0")
	} else if errOut == "" {
		t.Error("direct failed silently on unknown task")
	}
}

func TestDirectTaskRecordsDirective(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-DIR", "direct me"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "direct", "-repo", repoDir, "-by", "supervisor", "ACME-DIR", "switch to redis backend")
	if code != 0 {
		t.Fatalf("direct exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "ACME-DIR redirected") {
		t.Errorf("direct did not say it redirected the task: %s", out)
	}
}

func TestDirectTaskWithRestartFlag(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-REOPEN", "reopen me"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "direct", "-repo", repoDir, "-restart", "ACME-REOPEN", "retry with new plan")
	if code != 0 {
		t.Fatalf("direct -restart exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "restarted") {
		t.Errorf("direct -restart did not report restart: %s", out)
	}
}

// TestUnreadCannotBeGuessedAtZero. What is unread is the brake on starting a
// run, and zero is the one count atCap never refuses: a board that could not
// be read used to be reported as zero unread, so `direct -restart` started a
// run the cap existed to stop and nothing said why.
//
// It is asked of the function rather than of the command because the only
// board Refresh refuses is one whose root cannot be walked, and by the time
// the command reaches this line it has already opened that root as a
// repository — there is no way to take it away in between from inside one
// process.
func TestUnreadCannotBeGuessedAtZero(t *testing.T) {
	root, _ := workspace(t)

	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := unreadCount(s, filepath.Join(root, "no-such-directory")); err == nil {
		t.Error("a board that could not be read was counted as nothing unread")
	}

	n, err := unreadCount(s, filepath.Join(root, "payments"))
	if err != nil {
		t.Fatalf("count what is unread in a repository that is there: %v", err)
	}

	if n != 0 {
		t.Errorf("a repository with no finished tasks has %d unread, want 0", n)
	}
}
