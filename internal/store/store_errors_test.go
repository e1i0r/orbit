package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreTaskIDValidationComprehensive(t *testing.T) {
	// Valid IDs
	for _, id := range []string{"ACME-1", "feature_branch", "123", "PAY-404-retry"} {
		if err := ValidTaskID(id); err != nil {
			t.Errorf("ValidTaskID(%q) = %v, want nil", id, err)
		}
	}

	// Invalid IDs
	for _, id := range []string{"", ".", "..", "a/b", "../escape", "foo..bar"} {
		if err := ValidTaskID(id); err == nil {
			t.Errorf("ValidTaskID(%q) should have failed", id)
		}
	}
}

func TestStorePathsAndCreation(t *testing.T) {
	root := t.TempDir()
	s, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	repoPath := filepath.Join(t.TempDir(), "payments")

	// Store directories
	if s.Root() != root {
		t.Errorf("Root() = %q, want %q", s.Root(), root)
	}
	if s.FlowDir() != filepath.Join(root, "flows") {
		t.Errorf("FlowDir() = %q, want %q", s.FlowDir(), filepath.Join(root, "flows"))
	}
	if s.LogDir() != filepath.Join(root, "logs") {
		t.Errorf("LogDir() = %q, want %q", s.LogDir(), filepath.Join(root, "logs"))
	}
	if s.LogPath() != filepath.Join(root, "logs", "orbit.log") {
		t.Errorf("LogPath() = %q, want %q", s.LogPath(), filepath.Join(root, "logs", "orbit.log"))
	}

	// Task paths
	taskDir, err := s.TaskDir(repoPath, "TASK-1")
	if err != nil || !strings.Contains(taskDir, "TASK-1") {
		t.Fatalf("TaskDir: %v, dir=%q", err, taskDir)
	}

	ctrlPath, err := s.ControlPath(repoPath, "TASK-1")
	if err != nil || !strings.HasSuffix(ctrlPath, "control") {
		t.Fatalf("ControlPath: %v, path=%q", err, ctrlPath)
	}

	runPath, err := s.RunPath(repoPath, "TASK-1")
	if err != nil || !strings.HasSuffix(runPath, "run") {
		t.Fatalf("RunPath: %v, path=%q", err, runPath)
	}

	eventsPath, err := s.EventsPath(repoPath, "TASK-1")
	if err != nil || !strings.HasSuffix(eventsPath, "events.jsonl") {
		t.Fatalf("EventsPath: %v, path=%q", err, eventsPath)
	}

	// Worktree paths
	wtDir, err := s.WorktreeDir(repoPath, "TASK-1")
	if err != nil || !strings.Contains(wtDir, "TASK-1") {
		t.Fatalf("WorktreeDir: %v, dir=%q", err, wtDir)
	}

	// Create directories on disk
	createdTaskDir, err := s.CreateTaskDir(repoPath, "TASK-1")
	if err != nil {
		t.Fatalf("CreateTaskDir failed: %v", err)
	}
	if _, err := os.Stat(createdTaskDir); err != nil {
		t.Errorf("createdTaskDir does not exist on disk: %v", err)
	}

	createdWtParent, err := s.CreateWorktreeParent(repoPath, "TASK-1")
	if err != nil {
		t.Fatalf("CreateWorktreeParent failed: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(createdWtParent)); err != nil {
		t.Errorf("createdWtParent parent directory does not exist on disk: %v", err)
	}

	// Error on invalid task ID
	if _, err := s.TaskDir(repoPath, "../escape"); err == nil {
		t.Error("expected TaskDir to fail on traversal ID")
	}
	if _, err := s.ControlPath(repoPath, ""); err == nil {
		t.Error("expected ControlPath to fail on empty ID")
	}
}

func TestStoreSettingsSaveAndReadDefaults(t *testing.T) {
	root := t.TempDir()
	s, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// 1. Fresh store has defaults
	cfg, err := s.Settings()
	if err != nil {
		t.Fatalf("Settings fresh: %v", err)
	}
	if cfg.UnreadCap != defaultUnreadCap || cfg.Flow != defaultFlow {
		t.Errorf("unexpected default settings: %+v", cfg)
	}

	// 2. Save custom settings
	custom := Settings{
		Language:  "es",
		Autopilot: true,
		UnreadCap: 10,
		Engine:    "codex",
		Model:     "o3-mini",
		Flow:      "quick",
		Theme:     "frauddi",
	}
	if err := s.SaveSettings(custom); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	// 3. Read back saved settings
	readBack, err := s.Settings()
	if err != nil {
		t.Fatalf("Settings readback: %v", err)
	}
	if readBack != custom {
		t.Errorf("readback = %+v, want %+v", readBack, custom)
	}

	// 4. Corrupted settings file returns defaults without failing
	settingsFile := filepath.Join(root, "settings.json")
	if err := os.WriteFile(settingsFile, []byte("broken json content"), 0o644); err != nil {
		t.Fatalf("WriteFile broken: %v", err)
	}
	recovered, err := s.Settings()
	if err != nil {
		t.Fatalf("Settings over broken JSON returned error: %v", err)
	}
	if recovered.UnreadCap != defaultUnreadCap {
		t.Errorf("recovered.UnreadCap = %d, want default %d", recovered.UnreadCap, defaultUnreadCap)
	}
}

func TestStoreReposError(t *testing.T) {
	inner := errors.New("disk failure")
	rErr := &ReposError{Dir: "/path/to/repos", Err: inner}
	if !strings.Contains(rErr.Error(), "/path/to/repos") || !strings.Contains(rErr.Error(), "disk failure") {
		t.Errorf("unexpected ReposError text: %s", rErr.Error())
	}
	if !errors.Is(rErr, inner) {
		t.Errorf("expected ReposError to unwrap to inner error")
	}
}

func TestStoreRootPathFallbackAndOpen(t *testing.T) {
	t.Setenv("ORBIT_HOME", "")
	r, err := RootPath()
	if err != nil || r == "" {
		t.Errorf("RootPath() failed when ORBIT_HOME is unset: %v, got %q", err, r)
	}

	// Open() with default root
	s, err := Open()
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil Store from Open()")
	}
}

func TestStoreErrorsAndInvalidPaths(t *testing.T) {
	root := t.TempDir()
	s, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	repoPath := "/some/repo"

	// Invalid task ID tests across all methods
	for _, badID := range []string{"", "..", "../escape", "bad..id"} {
		if _, err := s.TaskDir(repoPath, badID); err == nil {
			t.Errorf("expected TaskDir to fail on %q", badID)
		}
		if _, err := s.ControlPath(repoPath, badID); err == nil {
			t.Errorf("expected ControlPath to fail on %q", badID)
		}
		if _, err := s.RunPath(repoPath, badID); err == nil {
			t.Errorf("expected RunPath to fail on %q", badID)
		}
		if _, err := s.EventsPath(repoPath, badID); err == nil {
			t.Errorf("expected EventsPath to fail on %q", badID)
		}
		if _, err := s.TaskFilePath(repoPath, badID); err == nil {
			t.Errorf("expected TaskFilePath to fail on %q", badID)
		}
		if _, err := s.WorktreeDir(repoPath, badID); err == nil {
			t.Errorf("expected WorktreeDir to fail on %q", badID)
		}
		if _, err := s.CreateTaskDir(repoPath, badID); err == nil {
			t.Errorf("expected CreateTaskDir to fail on %q", badID)
		}
		if _, err := s.CreateWorktreeParent(repoPath, badID); err == nil {
			t.Errorf("expected CreateWorktreeParent to fail on %q", badID)
		}
	}
}

func TestStoreSaveSettingsErrorAndRepoDirReuse(t *testing.T) {
	root := t.TempDir()
	s, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Create task dir twice to exercise existing marker check
	repoPath := filepath.Join(t.TempDir(), "project")
	dir1, err := s.CreateTaskDir(repoPath, "TASK-1")
	if err != nil {
		t.Fatalf("first CreateTaskDir failed: %v", err)
	}
	dir2, err := s.CreateTaskDir(repoPath, "TASK-2")
	if err != nil {
		t.Fatalf("second CreateTaskDir failed: %v", err)
	}
	if filepath.Dir(filepath.Dir(dir1)) != filepath.Dir(filepath.Dir(dir2)) {
		t.Errorf("expected same repo parent directory: %q vs %q", dir1, dir2)
	}

	// 2. SaveSettings error when settings.json is an unwriteable directory
	badStoreRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(badStoreRoot, "settings.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	badStore, err := New(badStoreRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := badStore.SaveSettings(Settings{Language: "es"}); err == nil {
		t.Error("expected error saving settings over directory blocker")
	}
}
