package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveNeedsAnID(t *testing.T) {
	root, _ := workspace(t)

	code, _, errOut := run(t, "resolve", "-repo", filepath.Join(root, "payments"))
	if code == 0 {
		t.Error("resolve with no id exited 0")
	}

	if errOut == "" {
		t.Error("resolve failed silently")
	}
}

// TestResolveSaysSoWhenNobodyAskedForAnything. A pull request that cannot be
// read is reported and is not fatal — but a task nobody reviewed still has
// to read as a task with nothing to answer, not as an error.
func TestResolveSaysSoWhenNobodyAskedForAnything(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	code, out, _ := run(t, "resolve", "-repo", repoDir, "ACME-1")
	if code != 0 {
		t.Fatalf("resolve exited %d", code)
	}

	if !strings.Contains(out, "ACME-1") {
		t.Errorf("resolve said %q, want it naming the task it found nothing for", out)
	}
}

func TestResolveRefusesATaskNobodyWrote(t *testing.T) {
	root, _ := workspace(t)

	code, _, errOut := run(t, "resolve", "-repo", filepath.Join(root, "payments"), "ACME-404")
	if code == 0 {
		t.Error("resolve reported success about a task nobody wrote")
	}

	if !strings.Contains(errOut, "ACME-404") {
		t.Errorf("the error does not name the task:\n%s", errOut)
	}
}
