package task

// The alive.go branches task_alive_and_notes_coverage_test.go does not
// reach: the marker's own error paths, and Alive's stale-across-boot early
// return, which needs a marker whose "started" line the code itself never
// writes that old.

import (
	"os"
	"strconv"
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

	path, err := s.RunPath(r.Path, tk.ID)
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
// refuses, and a task directory that was never created so the marker has
// nowhere to land.
func TestMarkErrorPaths(t *testing.T) {
	s, r := fixture(t)

	// 1. An id containing a path separator is refused before anything is
	// written — the store's own validation, surfaced through RunPath.
	bad := Task{ID: "has/slash", Repo: r}
	if _, err := mark(s, bad, 123); err == nil {
		t.Error("mark with a slash in the id should have failed")
	}

	// 2. A well-formed id whose directory was never created: RunPath
	// resolves fine, but the write has nowhere to land.
	neverCreated := Task{ID: "NEVER-CREATED-1", Repo: r}
	if _, err := mark(s, neverCreated, 123); err == nil {
		t.Error("mark into a task directory that does not exist should have failed")
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
	dir, err := s.TaskDir(r.Path, tk.ID)
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
	path, err := s.RunPath(r.Path, tk.ID)
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
