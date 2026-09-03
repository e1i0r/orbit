package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestApproveNeedsAnID(t *testing.T) {
	root, _ := workspace(t)

	code, _, errOut := run(t, "approve", "-repo", filepath.Join(root, "payments"))
	if code == 0 {
		t.Error("approve with no id exited 0")
	}

	if errOut == "" {
		t.Error("approve failed silently")
	}
}

// TestApproveSaysSoWhenNothingWasAdded. A command that answered "approved"
// about a task with nothing pending would have told the reader they made a
// decision they never made.
func TestApproveSaysSoWhenNothingWasAdded(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "approve", "-repo", repoDir, "ACME-1")
	if code != 0 {
		t.Fatalf("approve exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "ACME-1") || !strings.Contains(out, "no dependency") {
		t.Errorf("approve said %q, want it saying nothing is waiting on the reader", out)
	}
}

// TestApproveRefusesATaskNobodyWrote. The reader has to be answering a
// question that exists, and a typo in an id is not a task with nothing
// pending — it is a task there is nothing to say about at all.
func TestApproveRefusesATaskNobodyWrote(t *testing.T) {
	root, _ := workspace(t)

	code, _, errOut := run(t, "approve", "-repo", filepath.Join(root, "payments"), "ACME-404")
	if code == 0 {
		t.Error("approve reported success about a task nobody wrote")
	}

	if !strings.Contains(errOut, "ACME-404") {
		t.Errorf("the error does not name the task:\n%s", errOut)
	}
}

// TestApproveRefusesADirectoryThatIsNotARepository.
func TestApproveRefusesADirectoryThatIsNotARepository(t *testing.T) {
	code, _, errOut := run(t, "approve", "-repo", t.TempDir(), "ACME-1")
	if code == 0 {
		t.Error("approve reported success against a directory that is not a repository")
	}

	if errOut == "" {
		t.Error("approve failed silently")
	}
}
