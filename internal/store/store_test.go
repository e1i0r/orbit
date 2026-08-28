package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenPrefersOrbitHome(t *testing.T) {
	want := filepath.Join(t.TempDir(), "elsewhere")
	t.Setenv("ORBIT_HOME", want)

	s, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if s.Root() != want {
		t.Errorf("Root() = %q, want %q", s.Root(), want)
	}
}

func TestRepoDirIsStableAndDistinct(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	a1, err := s.RepoDir("/tmp/one")
	if err != nil {
		t.Fatalf("RepoDir: %v", err)
	}

	a2, err := s.RepoDir("/tmp/one")
	if err != nil {
		t.Fatalf("RepoDir: %v", err)
	}

	b, err := s.RepoDir("/tmp/two")
	if err != nil {
		t.Fatalf("RepoDir: %v", err)
	}

	if a1 != a2 {
		t.Errorf("same path gave two directories: %q and %q", a1, a2)
	}

	if a1 == b {
		t.Error("two different repositories share one directory")
	}

	if !strings.HasPrefix(a1, s.Root()) {
		t.Errorf("%q is outside the root %q", a1, s.Root())
	}
}

func TestComputingAPathCreatesNothing(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	paths := map[string]func() (string, error){
		"RepoDir":      func() (string, error) { return s.RepoDir("/tmp/one") },
		"TasksDir":     func() (string, error) { return s.TasksDir("/tmp/one") },
		"TaskDir":      func() (string, error) { return s.TaskDir("/tmp/one", "ACME-1") },
		"TaskFilePath": func() (string, error) { return s.TaskFilePath("/tmp/one", "ACME-1") },
		"WorktreeDir":  func() (string, error) { return s.WorktreeDir("/tmp/one", "ACME-1") },
		"EventsPath":   func() (string, error) { return s.EventsPath("/tmp/one", "ACME-1") },
	}
	for name, compute := range paths {
		path, err := compute()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		if _, statErr := os.Stat(path); statErr == nil {
			t.Errorf("%s created %q merely by being asked where it would be", name, path)
		} else if !os.IsNotExist(statErr) {
			t.Errorf("%s: stat %q: %v", name, path, statErr)
		}

		if _, statErr := os.Stat(filepath.Dir(path)); statErr == nil {
			t.Errorf("%s created the parent of %q merely by being asked where it would be", name, path)
		}
	}
}

func TestTaskFilePathSitsInsideTheTask(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	dir, err := s.TaskDir("/tmp/one", "ACME-1")
	if err != nil {
		t.Fatalf("TaskDir: %v", err)
	}

	path, err := s.TaskFilePath("/tmp/one", "ACME-1")
	if err != nil {
		t.Fatalf("TaskFilePath: %v", err)
	}

	if filepath.Dir(path) != dir {
		t.Errorf("task file at %q, expected inside %q", path, dir)
	}

	if filepath.Base(path) != "task.md" {
		t.Errorf("task file is %q, want task.md", filepath.Base(path))
	}
}

func TestEventsPathSitsInsideTheTask(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	dir, err := s.TaskDir("/tmp/one", "ACME-1")
	if err != nil {
		t.Fatalf("TaskDir: %v", err)
	}

	path, err := s.EventsPath("/tmp/one", "ACME-1")
	if err != nil {
		t.Fatalf("EventsPath: %v", err)
	}

	if filepath.Dir(path) != dir {
		t.Errorf("events at %q, expected inside %q", path, dir)
	}

	if filepath.Base(path) != "events.jsonl" {
		t.Errorf("events file is %q, want events.jsonl", filepath.Base(path))
	}
}

func TestWorktreeDirLeavesTheLeafForGitToCreate(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	path, err := s.WorktreeDir("/tmp/one", "ACME-1")
	if err != nil {
		t.Fatalf("WorktreeDir: %v", err)
	}

	if _, err := os.Stat(path); err == nil {
		t.Error("worktree leaf exists but must not; git worktree add will fail")
	} else if !os.IsNotExist(err) {
		t.Errorf("stat: %v", err)
	}
}

func TestWorktreeDirDiffersByRepository(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	one, err := s.WorktreeDir("/tmp/one", "ACME-1")
	if err != nil {
		t.Fatalf("WorktreeDir: %v", err)
	}

	two, err := s.WorktreeDir("/tmp/two", "ACME-1")
	if err != nil {
		t.Fatalf("WorktreeDir: %v", err)
	}

	if one == two {
		t.Errorf("two repositories share the worktree %q for the same task id — one would check out over the other", one)
	}
}

func TestWorktreeDirIsStableAndDistinct(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w1, err := s.WorktreeDir("/tmp/one", "ACME-1")
	if err != nil {
		t.Fatalf("WorktreeDir: %v", err)
	}

	w2, err := s.WorktreeDir("/tmp/one", "ACME-1")
	if err != nil {
		t.Fatalf("WorktreeDir: %v", err)
	}

	w3, err := s.WorktreeDir("/tmp/one", "ACME-2")
	if err != nil {
		t.Fatalf("WorktreeDir: %v", err)
	}

	if w1 != w2 {
		t.Errorf("same args gave two paths: %q and %q", w1, w2)
	}

	if w1 == w3 {
		t.Error("different task ids give the same path")
	}

	if !strings.HasPrefix(w1, s.Root()) {
		t.Errorf("%q is outside the root %q", w1, s.Root())
	}
}

func TestWorktreeDirRespectsRoot(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	path, err := s.WorktreeDir("/tmp/one", "ACME-1")
	if err != nil {
		t.Fatalf("WorktreeDir: %v", err)
	}

	if !strings.HasPrefix(path, s.Root()) {
		t.Errorf("%q is outside the root %q", path, s.Root())
	}
}

func TestTaskDirRejectsEscapingID(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = s.TaskDir("/tmp/one", "../../escape")
	if err == nil {
		t.Error("escaped path id was accepted")
	}
}

func TestWorktreeDirRejectsEscapingID(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = s.WorktreeDir("/tmp/one", "../../escape")
	if err == nil {
		t.Error("escaped path id was accepted")
	}
}

func TestOrdinaryIDWorks(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = s.TaskDir("/tmp/one", "ACME-1")
	if err != nil {
		t.Errorf("TaskDir rejected ordinary id: %v", err)
	}

	_, err = s.WorktreeDir("/tmp/one", "ACME-1")
	if err != nil {
		t.Errorf("WorktreeDir rejected ordinary id: %v", err)
	}
}

func TestTasksDirSitsUnderRepoDir(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	repoDir, err := s.RepoDir("/tmp/one")
	if err != nil {
		t.Fatalf("RepoDir: %v", err)
	}

	tasksDir, err := s.TasksDir("/tmp/one")
	if err != nil {
		t.Fatalf("TasksDir: %v", err)
	}

	want := filepath.Join(repoDir, "tasks")
	if tasksDir != want {
		t.Errorf("TasksDir() = %q, want %q", tasksDir, want)
	}
}

func TestWorktreeAndRepoAreSymmetric(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	w1, err := s.WorktreeDir("/tmp/one", "ACME-1")
	if err != nil {
		t.Fatalf("WorktreeDir: %v", err)
	}

	if !strings.HasPrefix(w1, s.Root()) {
		t.Errorf("worktree %q outside root %q", w1, s.Root())
	}

	relPath := "a/relative/path"

	absPath, err := filepath.Abs(relPath)
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}

	wRel, err := s.WorktreeDir(relPath, "ACME-1")
	if err != nil {
		t.Fatalf("WorktreeDir relative: %v", err)
	}

	wAbs, err := s.WorktreeDir(absPath, "ACME-1")
	if err != nil {
		t.Fatalf("WorktreeDir absolute: %v", err)
	}

	if wRel != wAbs {
		t.Errorf("relative %q and absolute %q gave different paths", relPath, absPath)
	}
}
