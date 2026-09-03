package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lastLine is the final non-empty line of a command's output, which is where
// `orbit join` prints the checkout it opened.
func lastLine(out string) string {
	lines := strings.Fields(strings.TrimSpace(out))
	if len(lines) == 0 {
		return ""
	}

	return lines[len(lines)-1]
}

func TestCriticalAndPermitNeedAnID(t *testing.T) {
	root, _ := workspace(t)

	for _, name := range []string{"critical", "permit"} {
		code, _, errOut := run(t, name, "-repo", filepath.Join(root, "payments"))
		if code == 0 {
			t.Errorf("%s with no id exited 0", name)
		}

		if errOut == "" {
			t.Errorf("%s failed silently", name)
		}
	}
}

// TestMarkingATaskCriticalSaysWhatItMeans. A toggle that answers "ok" leaves
// the reader to find out what they turned on by watching it happen.
func TestMarkingATaskCriticalSaysWhatItMeans(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "critical", "-repo", repoDir, "ACME-1")
	if code != 0 {
		t.Fatalf("critical exited %d: %s", code, errOut)
	}

	for _, want := range []string{"ACME-1", "backup", "asks you"} {
		if !strings.Contains(out, want) {
			t.Errorf("marking a task said %q, want it saying %q", out, want)
		}
	}

	code, out, _ = run(t, "critical", "-off", "-repo", repoDir, "ACME-1")
	if code != 0 || !strings.Contains(out, "ordinary") {
		t.Errorf("taking the mark off said %q", out)
	}
}

// TestPermitSaysSoWhenNothingIsWaiting.
func TestPermitSaysSoWhenNothingIsWaiting(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-2", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	code, out, _ := run(t, "permit", "-repo", repoDir, "ACME-2")
	if code != 0 {
		t.Fatalf("permit exited %d", code)
	}

	if !strings.Contains(out, "ACME-2") || !strings.Contains(out, "not waiting") {
		t.Errorf("permit said %q, want it saying the task is waiting on nobody", out)
	}
}

// TestACriticalTaskWillNotPushWithoutAWord. The boundary: past it the work
// is on a remote other people pull from, and that is the only place a
// critical task stops.
func TestACriticalTaskWillNotPushWithoutAWord(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-3", "touch the ledger"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	if code, _, errOut := run(t, "critical", "-repo", repoDir, "ACME-3"); code != 0 {
		t.Fatalf("critical exited %d: %s", code, errOut)
	}

	// A worktree with something in it. A repository with nothing to deliver
	// is not pushed at all, so it never reaches the boundary — which is
	// correct, and would make this test pass for the wrong reason.
	code, out, errOut := run(t, "join", "-repo", repoDir, "-task", "ACME-3", "payments")
	if code != 0 {
		t.Fatalf("join exited %d: %s", code, errOut)
	}

	wt := strings.TrimSpace(lastLine(out))
	if err := os.WriteFile(filepath.Join(wt, "ledger.txt"), []byte("a change\n"), 0o600); err != nil {
		t.Fatalf("write in the worktree %q: %v", wt, err)
	}

	code, _, errOut = run(t, "pr", "-repo", repoDir, "ACME-3")
	if code == 0 {
		t.Error("a critical task pushed without anybody being asked")
	}

	if !strings.Contains(errOut, "ACME-3") {
		t.Errorf("the refusal does not name the task:\n%s", errOut)
	}
}
