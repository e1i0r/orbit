package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestTheDigestReadsAnEmptyWorkspaceWithoutLying. A heading over no rows
// reads as a fact about the work — that nobody is ever stopped — when it is
// a fact about a record with nothing in it.
func TestTheDigestReadsAnEmptyWorkspaceWithoutLying(t *testing.T) {
	workspace(t)

	code, out, errOut := run(t, "digest")
	if code != 0 {
		t.Fatalf("digest exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "merged") {
		t.Errorf("the digest says nothing about what landed:\n%s", out)
	}

	if strings.Contains(out, "where people are stopped") {
		t.Errorf("the digest drew a heading over nothing:\n%s", out)
	}
}

// TestTheDigestCountsTheTasksThatAreThere.
func TestTheDigestCountsTheTasksThatAreThere(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	for _, id := range []string{"ACME-1", "ACME-2"} {
		if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", id, "x"); code != 0 {
			t.Fatalf("new %s exited %d: %s", id, code, errOut)
		}
	}

	code, out, errOut := run(t, "digest")
	if code != 0 {
		t.Fatalf("digest exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "of 2") {
		t.Errorf("the digest does not count the two tasks that exist:\n%s", out)
	}
}
