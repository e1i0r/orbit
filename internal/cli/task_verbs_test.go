package cli

// The early exits cancelTask (cancel.go) and controlTask/noteTask
// (pause.go, note.go) take before they ever reach a write, plus the two
// write failures and the two cancel success paths none of the hand-written
// tests reach: those are built around a task that can always be loaded and
// a control/events file that can always be written.
//
// cli_workflows_test.go's TestCancelTaskExecution (a hand-written test this
// file leaves untouched) plants its run marker
// at s.RunPath(...) using the -repo string exactly as typed, while
// cancelTask resolves the repository through repo.Open first — which, on a
// machine where the temp directory is itself a symlink (macOS: /var ->
// /private/var), returns a different, canonicalised path. The marker and
// the lookup disagree, task.Alive never finds it, and both of that test's
// calls end in "not running". markerPath (cancel_test.go) asks the store,
// which resolves the same way every command does.
//
// The write failures are forced on the record rather than on one task: it is
// one file for every task now, so the fault is breakRecord — the file taken
// away and a directory left where it goes — and what fails is a command that
// cannot write down what it did.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCancelTaskEarlyExits(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	// 1. A flag parse failure.
	if code, _, errOut := run(t, "cancel", "-repo", repoDir, "-nosuchflag"); code == 0 {
		t.Error("cancel with an unknown flag exited 0")
	} else if errOut == "" {
		t.Error("cancel failed silently on a bad flag")
	}

	// 2. openBoth fails: not a repository at all.
	if code, _, errOut := run(t, "cancel", "-repo", t.TempDir(), "ACME-1"); code == 0 {
		t.Error("cancel outside a repository exited 0")
	} else if errOut == "" {
		t.Error("cancel failed silently outside a repository")
	}

	// 3. task.Load fails: a real repository, a task never written.
	if code, _, errOut := run(t, "cancel", "-repo", repoDir, "ACME-404"); code == 0 {
		t.Error("cancel on a task that was never created exited 0")
	} else if errOut == "" {
		t.Error("cancel failed silently on an unknown task")
	}
}

// plantLiveMarker writes a run marker naming a real, currently-running
// process for one task, at the path the store gives for it.
func plantLiveMarker(t *testing.T, id string) *exec.Cmd {
	t.Helper()

	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start a background process: %v", err)
	}

	t.Cleanup(func() { _ = cmd.Process.Kill() }) //nolint:errcheck

	body := fmt.Sprintf("pid: %d\nstarted: %s\n", cmd.Process.Pid, time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(markerPath(t, id), []byte(body), 0o600); err != nil {
		t.Fatalf("write the run marker: %v", err)
	}

	return cmd
}

func TestCancelNowKillsALiveProcess(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-KILL", "kill me"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	plantLiveMarker(t, "ACME-KILL")

	code, out, errOut := run(t, "cancel", "-now", "-repo", repoDir, "ACME-KILL")
	if code != 0 {
		t.Fatalf("cancel -now on a live process exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "killed") {
		t.Errorf("cancel -now did not say it killed the task:\n%s", out)
	}
}

func TestCancelGracefullyAsksALiveProcess(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-ASK", "ask me"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	plantLiveMarker(t, "ACME-ASK")

	code, out, errOut := run(t, "cancel", "-repo", repoDir, "ACME-ASK")
	if code != 0 {
		t.Fatalf("cancel on a live process exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "asked to stop") {
		t.Errorf("cancel did not say it asked the task to stop:\n%s", out)
	}
}

func TestControlTaskEarlyExits(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	// 1. openBoth fails outside a repository.
	if code, _, errOut := run(t, "pause", "-repo", t.TempDir(), "ACME-1"); code == 0 {
		t.Error("pause outside a repository exited 0")
	} else if errOut == "" {
		t.Error("pause failed silently outside a repository")
	}

	// 2. task.Load fails: a real repository, a task never written.
	if code, _, errOut := run(t, "resume", "-repo", repoDir, "ACME-404"); code == 0 {
		t.Error("resume on a task that was never created exited 0")
	} else if errOut == "" {
		t.Error("resume failed silently on an unknown task")
	}
}

func TestControlTaskFailsWhenTheTaskDirCannotBeWrittenTo(t *testing.T) {
	root, orbitHome := workspace(t)
	dir := writeTask(t, root)

	taskDir := filepath.Dir(findFile(t, orbitHome, "task.md"))
	if err := os.Chmod(taskDir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer func() { _ = os.Chmod(taskDir, 0o700) }() //nolint:errcheck

	code, _, errOut := run(t, "pause", "-repo", dir, "ACME-1")
	if code == 0 {
		t.Error("pause over a read-only task directory exited 0")
	}

	if errOut == "" {
		t.Error("pause failed silently over a read-only task directory")
	}
}

func TestNoteTaskEarlyExits(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	// 1. A flag parse failure.
	if code, _, errOut := run(t, "note", "-repo", repoDir, "-nosuchflag"); code == 0 {
		t.Error("note with an unknown flag exited 0")
	} else if errOut == "" {
		t.Error("note failed silently on a bad flag")
	}

	// 2. openBoth fails outside a repository.
	if code, _, errOut := run(t, "note", "-repo", t.TempDir(), "ACME-1", "text"); code == 0 {
		t.Error("note outside a repository exited 0")
	} else if errOut == "" {
		t.Error("note failed silently outside a repository")
	}

	// 3. task.Load fails: a real repository, a task never written.
	if code, _, errOut := run(t, "note", "-repo", repoDir, "ACME-404", "text"); code == 0 {
		t.Error("note on a task that was never created exited 0")
	} else if errOut == "" {
		t.Error("note failed silently on an unknown task")
	}
}

func TestNoteTaskFailsOverARecordItCannotReach(t *testing.T) {
	root, _ := workspace(t)
	dir := writeTask(t, root)

	breakRecord(t)

	code, _, errOut := run(t, "note", "-repo", dir, "ACME-1", "a note nobody can write")
	if code == 0 {
		t.Error("note over a record nothing can reach exited 0")
	}

	if errOut == "" {
		t.Error("note failed silently over a record nothing can reach")
	}
}
