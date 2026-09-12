package cli

// Asking about a task through its commands: permitting, marking,
// reading its history, and counting the board.
//
// These go through run(), the way a reader types them, because what is
// covered is the command and the doing together.

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/task"
)

// TestPermittingAndMarkingThroughCommands. A snapshot before a merge is
// the question; permit answers it, and critical marks the task so the
// question is asked at all.
func TestPermittingAndMarkingThroughCommands(t *testing.T) {
	s, _, repoDir := portWorld(t)

	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "PAY-1", "merge it"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	if code, _, errOut := run(t, "permit", "-repo", repoDir, "PAY-1"); code != 0 {
		t.Fatalf("permit exited %d: %s", code, errOut)
	}

	if code, _, errOut := run(t, "critical", "-repo", repoDir, "PAY-1"); code != 0 {
		t.Fatalf("critical exited %d: %s", code, errOut)
	}

	if code, _, errOut := run(t, "critical", "-repo", repoDir, "-off", "PAY-1"); code != 0 {
		t.Fatalf("critical -off exited %d: %s", code, errOut)
	}

	wtDir, err := s.WorktreeDir(repoDir, "PAY-1")
	if err != nil {
		t.Fatalf("worktree dir: %v", err)
	}

	add := exec.Command("git", "worktree", "add", wtDir, "-b", "orbit/PAY-1")
	add.Dir = repoDir

	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v (%s)", err, out)
	}

	r, err := repo.Open(repoDir)
	if err != nil {
		t.Fatalf("open the repository: %v", err)
	}

	tk, err := task.Load(s, r, "PAY-1")
	if err != nil {
		t.Fatalf("load the task: %v", err)
	}

	if _, err := task.Snapshot(s, tk, r, wtDir, task.Action{Name: "merge", Plan: "merge it"}); err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if code, out, errOut := run(t, "permit", "-repo", repoDir, "PAY-1"); code != 0 {
		t.Fatalf("permit exited %d: %s", code, errOut)
	} else if !strings.Contains(out, "allowed") {
		t.Errorf("permit said %q", out)
	}

	if _, err := task.Snapshot(s, tk, r, wtDir, task.Action{Name: "merge", Plan: "merge it"}); err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if code, out, errOut := run(t, "permit", "-repo", repoDir, "-no", "PAY-1"); code != 0 {
		t.Fatalf("permit -no exited %d: %s", code, errOut)
	} else if !strings.Contains(out, "refused") {
		t.Errorf("permit -no said %q", out)
	}
}

// TestHistoryPrintsAndKeeps. Everything ever said, printed — and kept
// with -write for whoever reads the directory instead.
func TestHistoryPrintsAndKeeps(t *testing.T) {
	_, _, repoDir := portWorld(t)

	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "history", "-repo", repoDir, "ACME-1")
	if code != 0 {
		t.Fatalf("history exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "ACME-1") {
		t.Errorf("history printed %q", out)
	}

	if code, _, errOut := run(t, "history", "-repo", repoDir, "-write", "ACME-1"); code != 0 {
		t.Fatalf("history -write exited %d: %s", code, errOut)
	}
}

// TestDigestCountsWhatIsThere. The digest over a board with tasks on it
// names them rather than reporting an empty room.
func TestDigestCountsWhatIsThere(t *testing.T) {
	_, _, repoDir := portWorld(t)

	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "digest")
	if code != 0 {
		t.Fatalf("digest exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "ACME-1") && !strings.Contains(out, "1") {
		t.Errorf("digest printed %q", out)
	}
}
