package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// abandon leaves behind what a run killed with SIGKILL leaves behind: a log
// that says a phase started and stops there, and a marker naming a process
// that is gone.
//
// The two lines are written as text rather than through internal/record,
// which this package may not import — and that is the point being made as
// well as a constraint obeyed: the record is JSON lines that anything can
// append to, and a stranger with an editor writes exactly this.
func abandon(t *testing.T, orbitHome string) {
	t.Helper()
	events := findFile(t, orbitHome, "events.jsonl")

	f, err := os.OpenFile(events, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open the log: %v", err)
	}
	defer f.Close()

	lines := `{"at":"2026-08-23T09:00:00Z","kind":"task.started","data":{"flow":"task"}}
{"at":"2026-08-23T09:00:01Z","kind":"phase.started","phase":"implement","data":{"engine":"claude","n":"1"}}
`
	if _, err := f.WriteString(lines); err != nil {
		t.Fatalf("append to the log: %v", err)
	}

	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	if err := cmd.Run(); err != nil {
		t.Fatalf("run a process that exits immediately: %v", err)
	}

	marker := filepath.Join(filepath.Dir(events), "run")

	body := "pid: " + strconv.Itoa(cmd.ProcessState.Pid()) + "\nstarted: 2026-08-23T09:00:00Z\n"
	if err := os.WriteFile(marker, []byte(body), 0o600); err != nil {
		t.Fatalf("write the marker: %v", err)
	}
}

// findFile returns the one file of a name under a root, so a test does not
// have to know how the store hashes a repository's path into a directory.
func findFile(t *testing.T, root, name string) string {
	t.Helper()

	var found []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && d.Name() == name {
			found = append(found, path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	if len(found) != 1 {
		t.Fatalf("found %d files named %s under %s, want 1", len(found), name, root)
	}

	return found[0]
}

func TestCancelNeedsAnID(t *testing.T) {
	root, _ := workspace(t)

	code, _, errOut := run(t, "cancel", "-repo", filepath.Join(root, "payments"))
	if code == 0 {
		t.Error("cancel with no id exited 0")
	}

	if errOut == "" {
		t.Error("cancel failed silently")
	}
}

func TestCancelSaysSoWhenNothingIsRunning(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	code, _, errOut := run(t, "cancel", "-repo", repoDir, "ACME-1")
	if code == 0 {
		t.Error("cancel reported success against a task no process holds")
	}

	if !strings.Contains(errOut, "ACME-1") {
		t.Errorf("the error does not name the task:\n%s", errOut)
	}
}

func TestReconcileClosesARecordAndSaysWhatItDid(t *testing.T) {
	root, orbitHome := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	abandon(t, orbitHome)

	code, out, errOut := run(t, "reconcile", "-repo", repoDir)
	if code != 0 {
		t.Fatalf("reconcile exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "ACME-1") {
		t.Errorf("reconcile does not say which task it closed:\n%s", out)
	}

	code, out, errOut = run(t, "show", "-repo", repoDir, "ACME-1")
	if code != 0 {
		t.Fatalf("show exited %d: %s", code, errOut)
	}

	if !strings.Contains(out, "task.abandoned") {
		t.Errorf("the record still says the run is going:\n%s", out)
	}
}

func TestReconcileSaysWhenThereIsNothingToDo(t *testing.T) {
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	code, out, errOut := run(t, "reconcile", "-repo", repoDir)
	if code != 0 {
		t.Fatalf("reconcile exited %d: %s", code, errOut)
	}

	if out == "" {
		t.Error("reconcile said nothing at all, so a reader cannot tell it ran")
	}

	code, out, _ = run(t, "show", "-repo", repoDir, "ACME-1")
	if code != 0 {
		t.Fatalf("show exited %d", code)
	}

	if strings.Contains(out, "task.abandoned") {
		t.Errorf("reconcile wrote to the record of a task that never ran:\n%s", out)
	}
}

// The one command that spends money must be stoppable, and the flag that
// stops it on a clock has to be discoverable without reading the source.
func TestRunOffersATimeout(t *testing.T) {
	code, out, _ := run(t, "run", "-h")
	if code != 0 {
		t.Errorf("run -h exited %d", code)
	}

	if !strings.Contains(out, "-timeout") {
		t.Errorf("run does not offer a timeout:\n%s", out)
	}
}
