package store

// A lock left behind is the footprint of a process that died holding it. It
// is broken and the change goes through, which is the right thing to do and
// no reason for it to happen quietly.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/logger"
)

func TestBreakingAStaleLockIsWrittenDown(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ORBIT_HOME", root)

	logs := t.TempDir()
	logPath := filepath.Join(logs, "orbit.log")

	if err := logger.Init(logPath, filepath.Join(logs, "errors.log")); err != nil {
		t.Fatalf("init the log: %v", err)
	}

	defer func() {
		if err := logger.CloseGlobal(); err != nil {
			t.Errorf("close the log: %v", err)
		}
	}()

	s, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// A lock older than lockStale: something took it and never gave it back.
	lock := s.settingsPath() + lockSuffix
	if err := os.WriteFile(lock, nil, fileMode); err != nil {
		t.Fatalf("write the lock: %v", err)
	}

	old := time.Now().Add(-2 * lockStale)
	if err := os.Chtimes(lock, old, old); err != nil {
		t.Fatalf("age the lock: %v", err)
	}

	if err := s.UpdateSettings(func(*Settings) error { return nil }); err != nil {
		t.Fatalf("a change over a stale lock was refused: %v", err)
	}

	body, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read the log: %v", err)
	}

	said := string(body)
	if !strings.Contains(said, "settings lock") || !strings.Contains(said, "died holding it") {
		t.Errorf("breaking a stale settings lock did not say a process had died holding it; "+
			"the log holds:\n%s", said)
	}
}
