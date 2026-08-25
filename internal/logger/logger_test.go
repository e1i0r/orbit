package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoggerLifecycleAndLevels(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	l, err := New(logPath)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	l.Log(LevelDebug, "debug event %d", 1)
	l.Log(LevelInfo, "info event %s", "started")
	l.Log(LevelWarn, "warn event")
	l.Log(LevelError, "error occurred: %v", "fatal")

	if err := l.Close(); err != nil {
		t.Errorf("error closing logger: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	for _, expected := range []string{"[DEBUG]", "[INFO]", "[WARN]", "[ERROR]", "debug event 1", "info event started"} {
		if !strings.Contains(content, expected) {
			t.Errorf("expected log file to contain %q, got:\n%s", expected, content)
		}
	}
}

func TestGlobalLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "global.log")

	if err := Init(logPath); err != nil {
		t.Fatalf("failed to init global logger: %v", err)
	}
	defer func() {
		if err := CloseGlobal(); err != nil {
			t.Errorf("error closing global logger in defer: %v", err)
		}
	}()

	Debug("global debug")
	Info("global info")
	Warn("global warn")
	Error("global error")

	if err := CloseGlobal(); err != nil {
		t.Errorf("error closing global logger: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read global log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "global info") || !strings.Contains(content, "global error") {
		t.Errorf("global logger missing entries, got:\n%s", content)
	}
}

func TestLevelStrings(t *testing.T) {
	tests := []struct {
		lvl  Level
		want string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{Level(99), "INFO"},
	}

	for _, tt := range tests {
		if got := tt.lvl.String(); got != tt.want {
			t.Errorf("Level(%d).String() = %q, want %q", tt.lvl, got, tt.want)
		}
	}
}

func TestNilLoggerSafety(t *testing.T) {
	var l *Logger
	if err := l.Close(); err != nil {
		t.Errorf("nil logger close should not error: %v", err)
	}
	l.Log(LevelInfo, "should not panic")
}

func TestLoggerErrorPaths(t *testing.T) {
	if _, err := New(""); err == nil {
		t.Error("expected error creating logger with empty path")
	}
	if err := Init(""); err == nil {
		t.Error("expected error initializing global logger with empty path")
	}

	l, err := New(filepath.Join(t.TempDir(), "temp.log"))
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Errorf("failed to close logger: %v", err)
	}
	// Log on closed file should be safe
	l.Log(LevelInfo, "entry on closed file")
}
