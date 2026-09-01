package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
)

// abandon leaves behind what a run killed with SIGKILL leaves behind: a
// record that says a phase started and stops there, and a marker naming a
// process that is gone.
func abandon(t *testing.T, repoDir, id string) {
	t.Helper()

	plant(t, repoDir, id,
		`{"at":"2026-08-23T09:00:00Z","kind":"task.started","data":{"flow":"task"}}`,
		`{"at":"2026-08-23T09:00:01Z","kind":"phase.started","phase":"implement","data":{"engine":"claude","n":"1"}}`,
	)

	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	if err := cmd.Run(); err != nil {
		t.Fatalf("run a process that exits immediately: %v", err)
	}

	body := "pid: " + strconv.Itoa(cmd.ProcessState.Pid()) + "\nstarted: 2026-08-23T09:00:00Z\n"
	if err := os.WriteFile(markerPath(t, id), []byte(body), 0o600); err != nil {
		t.Fatalf("write the marker: %v", err)
	}
}

// plant puts events in the record that no command writes, by writing them
// into the log an older Orbit kept and letting the next command carry them
// across.
//
// The lines are text, and that is the point being made as well as a
// constraint obeyed. This package may reach neither internal/record nor
// internal/db — the import table in internal/arch says so, and it counts test
// files — so its one door onto the contents of the record is internal/migrate,
// which every command runs before it does anything. What a stranger with an
// editor can still write is exactly this file, and the migration is its last
// reader.
//
// The lines given are the ones the record has not got yet. Where the
// migration resumes is a count — the first n lines of the log are the n
// events already recorded — so the file is padded to that count first, and
// nothing reads what the padding says.
func plant(t *testing.T, repoDir, id string, lines ...string) {
	t.Helper()

	s, err := store.Open()
	if err != nil {
		t.Fatalf("open the state root: %v", err)
	}

	path, err := s.EventsPath(id)
	if err != nil {
		t.Fatalf("find the log of %q: %v", id, err)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("let go of the record: %v", err)
	}

	body := strings.Repeat(`{"kind":"already recorded"}`+"\n", len(recorded(t, repoDir, id)))
	body += strings.Join(lines, "\n") + "\n"

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write the log of %q: %v", id, err)
	}
}

// recorded is the kind of every event the record holds about one task, in
// order, which is how this package asks what a command wrote down.
//
// It goes through internal/task, the same door the commands go through, for
// the reason plant writes text: nothing here may read the record itself.
func recorded(t *testing.T, repoDir, id string) []string {
	t.Helper()

	s, err := store.Open()
	if err != nil {
		t.Fatalf("open the state root: %v", err)
	}

	defer func() {
		if err := s.Close(); err != nil {
			t.Fatalf("let go of the record: %v", err)
		}
	}()

	r, err := repo.Open(repoDir)
	if err != nil {
		t.Fatalf("open %q: %v", repoDir, err)
	}

	events, err := task.Events(s, task.Task{ID: id, Repo: r})
	if err != nil {
		t.Fatalf("read the record of %q: %v", id, err)
	}

	kinds := make([]string, len(events))
	for i, e := range events {
		kinds[i] = e.Kind
	}

	return kinds
}

// markerPath is where the run marker of one task goes, asked of the store
// rather than built from the -repo string as typed.
//
// The two are not the same path. On a machine where the temp directory is
// itself a symlink — macOS, /var -> /private/var — a marker written under the
// typed path is one task.Alive never finds, because every command resolves
// the repository through repo.Open first.
func markerPath(t *testing.T, id string) string {
	t.Helper()

	s, err := store.Open()
	if err != nil {
		t.Fatalf("open the state root: %v", err)
	}

	defer func() {
		if err := s.Close(); err != nil {
			t.Fatalf("let go of the record: %v", err)
		}
	}()

	path, err := s.RunPath(id)
	if err != nil {
		t.Fatalf("find the run marker of %q: %v", id, err)
	}

	return path
}

// breakRecord makes the record unreachable, by taking it away and leaving a
// directory where it goes.
//
// A directory rather than a permission bit because the bit depends on who is
// running the test, and root would not notice it. It breaks reading and
// writing together: the record is one file now, and there is no fault that
// stops a command writing to it while it still reads.
func breakRecord(t *testing.T) {
	t.Helper()

	s, err := store.Open()
	if err != nil {
		t.Fatalf("open the state root: %v", err)
	}

	path := s.DBPath()
	if err := s.Close(); err != nil {
		t.Fatalf("let go of the record: %v", err)
	}

	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("remove the record: %v", err)
	}

	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf("put a directory where the record goes: %v", err)
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
	root, _ := workspace(t)

	repoDir := filepath.Join(root, "payments")
	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "x"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	abandon(t, repoDir, "ACME-1")

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
