package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

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
