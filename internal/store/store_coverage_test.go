package store

import (
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
	settingsPath := filepath.Join(root, "settings.json")
	if err := os.WriteFile(settingsPath, []byte("broken json content"), 0o644); err != nil {
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

func TestStoreRootPathEnv(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("ORBIT_HOME", tempRoot)

	r, err := RootPath()
	if err != nil || r != tempRoot {
		t.Errorf("RootPath() = %q, want %q", r, tempRoot)
	}
}
