package task

// The alive.go branches task_alive_and_notes_coverage_test.go does not
// reach: the marker's own error paths, and Alive's stale-across-boot early
// return, which needs a marker whose "started" line the code itself never
// writes that old.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestAliveReturnsDeadForAMarkerWrittenBeforeThisBoot plants a marker whose
// pid is very much alive (this test process) but whose started line predates
// the machine's own boot by a wide margin, so Alive must answer false without
// ever asking the kernel about the pid.
func TestAliveReturnsDeadForAMarkerWrittenBeforeThisBoot(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "STALE-BOOT-1", "stale boot test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, ok := bootTime(); !ok {
		t.Skip("this machine cannot report a boot time")
	}

	path, err := s.RunPath(tk.ID)
	if err != nil {
		t.Fatalf("RunPath: %v", err)
	}

	ancient := time.Unix(0, 0).UTC().Format(time.RFC3339)

	body := "pid: " + strconv.Itoa(os.Getpid()) + "\nstarted: " + ancient + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	pid, ok, err := Alive(s, tk)
	if err != nil {
		t.Fatalf("Alive: %v", err)
	}

	if ok {
		t.Error("Alive said a marker from before this boot is still running")
	}

	if pid != os.Getpid() {
		t.Errorf("Alive pid = %d, want %d — the pid is still reported even when stale", pid, os.Getpid())
	}
}

// TestMarkErrorPaths covers mark's two error returns: an id the store
// refuses, and a task directory it cannot make.
func TestMarkErrorPaths(t *testing.T) {
	s, r := fixture(t)

	// 1. An id containing a path separator is refused before anything is
	// written — the store's own validation, surfaced through RunPath.
	bad := Task{ID: "has/slash", Repo: r}
	if _, err := mark(s, bad, 123); err == nil {
		t.Error("mark with a slash in the id should have failed")
	}

	// 2. A regular file where the task's directory would go. mark makes the
	// directory itself now — the marker is the first thing a run writes —
	// so the failure to reach is the one where making it is impossible.
	blocked := Task{ID: "BLOCKED-1", Repo: r}

	dir, err := s.TaskDir(blocked.ID)
	if err != nil {
		t.Fatalf("TaskDir: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(dir), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(dir, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := mark(s, blocked, 123); err == nil {
		t.Error("mark should have failed where the task directory cannot be made")
	}
}

// TestMarkMakesTheTaskDirectoryItself pins the other half of that change.
// The marker goes down before task.started, so a run whose task directory
// is not there yet has to be able to make it — otherwise no run could ever
// claim a task before writing to its log.
func TestMarkMakesTheTaskDirectoryItself(t *testing.T) {
	s, r := fixture(t)
	fresh := Task{ID: "NEVER-CREATED-1", Repo: r}

	release, err := mark(s, fresh, os.Getpid())
	if err != nil {
		t.Fatalf("mark into a task directory that does not exist yet: %v", err)
	}
	defer release()

	if _, alive, err := Alive(s, fresh); err != nil || !alive {
		t.Errorf("Alive = (_, %v, %v), want the marker mark just wrote", alive, err)
	}
}

// TestHoldRefusesATaskAnotherRunIsWalking.
//
// Two runs of one task share a worktree, a branch and a log. With nothing to
// stop the second one, hold overwrites the marker, both walk the flow, and
// whichever finishes first takes the other's claim off on its way out.
func TestHoldRefusesATaskAnotherRunIsWalking(t *testing.T) {
	s, r := fixture(t)
	held := Task{ID: "HELD-1", Repo: r}
	// A marker naming this process, which is alive by construction.
	release, err := mark(s, held, os.Getpid())
	if err != nil {
		t.Fatalf("mark: %v", err)
	}
	defer release()

	if _, err := hold(s, held); err == nil {
		t.Fatal("hold claimed a task another live run is already walking")
	} else if !strings.Contains(err.Error(), held.ID) {
		t.Errorf("the refusal reads %q; it has to name the task", err)
	}
}

// TestHoldClaimsATaskWhoseRunIsGone is the other side of it: a marker left
// behind by a run that died is not a reason to refuse for ever.
func TestHoldClaimsATaskWhoseRunIsGone(t *testing.T) {
	s, r := fixture(t)
	stale := Task{ID: "STALE-1", Repo: r}
	// A pid no process can hold: mark refuses nothing, and Alive reports it
	// as a claim whose process is gone.
	if _, err := mark(s, stale, deadPid(t)); err != nil {
		t.Fatalf("mark: %v", err)
	}

	release, err := hold(s, stale)
	if err != nil {
		t.Fatalf("hold over a stale marker: %v", err)
	}
	defer release()

	pid, alive, err := Alive(s, stale)
	if err != nil {
		t.Fatalf("Alive: %v", err)
	}

	if !alive || pid != os.Getpid() {
		t.Errorf("Alive = (%d, %v), want this process holding it", pid, alive)
	}
}

// TestReleaseLeavesAMarkerThatNamesSomebodyElse is the second half of the
// same bug. A run asked to stop takes as long to die as its engine does, and
// the run that replaces it claims the task in the meantime; a release that
// removed the marker blindly took that claim off instead, leaving a live run
// nothing claimed and nobody could cancel.
func TestReleaseLeavesAMarkerThatNamesSomebodyElse(t *testing.T) {
	s, r := fixture(t)
	shared := Task{ID: "SHARED-1", Repo: r}

	// The dying run's claim, and the release it is holding.
	release, err := mark(s, shared, os.Getpid())
	if err != nil {
		t.Fatalf("mark: %v", err)
	}
	// The replacement run claims it while the first is still unwinding.
	successor := os.Getpid() + 1
	if _, err := mark(s, shared, successor); err != nil {
		t.Fatalf("mark successor: %v", err)
	}

	release()

	pid, _, found, err := readMarker(s, shared)
	if err != nil {
		t.Fatalf("readMarker: %v", err)
	}

	if !found {
		t.Fatal("the first run's release took the successor's claim off with it")
	}

	if pid != successor {
		t.Errorf("the marker names %d, want the successor %d", pid, successor)
	}
}

// TestRemoveMarkerErrorPaths covers removeMarker's two error returns.
func TestRemoveMarkerErrorPaths(t *testing.T) {
	s, r := fixture(t)

	// 1. Bad id.
	bad := Task{ID: "has/slash", Repo: r}
	if err := removeMarker(s, bad); err == nil {
		t.Error("removeMarker with a slash in the id should have failed")
	}

	// 2. A marker that cannot be removed because its directory has lost its
	// write bit — Remove fails for a reason other than "already gone".
	tk, err := Create(s, r, "RM-MARKER-1", "remove marker test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := mark(s, tk, os.Getpid()); err != nil {
		t.Fatalf("mark: %v", err)
	}

	dir, err := s.TaskDir(tk.ID)
	if err != nil {
		t.Fatalf("TaskDir: %v", err)
	}

	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) //nolint:errcheck

	if err := removeMarker(s, tk); err == nil {
		t.Error("removeMarker on a read-only directory should have failed")
	}
}

// TestReadMarkerErrorPaths covers readMarker's two error returns.
func TestReadMarkerErrorPaths(t *testing.T) {
	s, r := fixture(t)

	// 1. Bad id.
	bad := Task{ID: "has/slash", Repo: r}
	if _, _, _, err := readMarker(s, bad); err == nil {
		t.Error("readMarker with a slash in the id should have failed")
	}

	// 2. Something other than "not exist" reading the marker: a directory
	// sitting where the marker file should be.
	tk, err := Create(s, r, "READ-MARKER-1", "read marker test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	path, err := s.RunPath(tk.ID)
	if err != nil {
		t.Fatalf("RunPath: %v", err)
	}

	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	if _, _, _, err := readMarker(s, tk); err == nil {
		t.Error("readMarker over a directory should have failed")
	}
}
