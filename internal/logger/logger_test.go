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

	l, err := New(logPath, filepath.Join(tmpDir, "test-errors.log"))
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	l.Log(LevelDebug, "cli/test", "debug event %d", 1)
	l.Log(LevelInfo, "task/runner", "info event %s", "started")
	l.Log(LevelWarn, "board/poll", "warn event")
	l.Log(LevelError, "engine/claude", "error occurred: %v", "fatal")

	if err := l.Close(); err != nil {
		t.Errorf("error closing logger: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)

	expectedTokens := []string{
		"[DEBUG]", "[INFO]", "[WARN]", "[ERROR]",
		"[cli/test]", "[task/runner]", "[board/poll]", "[engine/claude]",
		"debug event 1", "info event started",
	}
	for _, expected := range expectedTokens {
		if !strings.Contains(content, expected) {
			t.Errorf("expected log file to contain %q, got:\n%s", expected, content)
		}
	}
}

func TestGlobalLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "global.log")

	if err := Init(logPath, filepath.Join(tmpDir, "global-errors.log")); err != nil {
		t.Fatalf("failed to init global logger: %v", err)
	}
	defer func() {
		if err := CloseGlobal(); err != nil {
			t.Errorf("error closing global logger in defer: %v", err)
		}
	}()

	Debug("cli/top", "global debug")
	Info("cli/run", "global info")
	Warn("task/gate", "global warn")
	Error("engine/exec", "global error")

	if err := CloseGlobal(); err != nil {
		t.Errorf("error closing global logger: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read global log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "[cli/run] global info") || !strings.Contains(content, "[engine/exec] global error") {
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

	l.Log(LevelInfo, "test", "should not panic")
}

func TestLoggerErrorPaths(t *testing.T) {
	if _, err := New("", ""); err == nil {
		t.Error("expected error creating logger with empty path")
	}

	if err := Init("", ""); err == nil {
		t.Error("expected error initializing global logger with empty path")
	}

	l, err := New(filepath.Join(t.TempDir(), "temp.log"), "")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	if err := l.Close(); err != nil {
		t.Errorf("failed to close logger: %v", err)
	}
	// Log on closed file should be safe
	l.Log(LevelInfo, "", "entry on closed file")
}

// readOnly replaces the logger's descriptor with one that cannot be written
// to and can still be closed. That combination is the point: a test that
// simply closes the file underneath the logger passes whether or not the
// failed write was kept, because closing an already-closed file fails on its
// own and the error looks the same from outside.
func readOnly(t *testing.T, l *Logger) {
	t.Helper()

	if err := l.file.Close(); err != nil {
		t.Fatalf("close the writable descriptor: %v", err)
	}

	f, err := os.Open(l.path)
	if err != nil {
		t.Fatalf("reopen %q read-only: %v", l.path, err)
	}

	l.file = f
}

// TestAWriteThatFailedIsNotSwallowed. Log cannot return an error, and the
// version this replaces therefore discarded one: a log whose file had gone
// away carried on looking healthy for the rest of the process, and every
// diagnostic written to it after the first failure was lost in silence.
// The failure is kept and handed to whoever closes the logger.
func TestAWriteThatFailedIsNotSwallowed(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "broken.log")

	l, err := New(logPath, "")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	readOnly(t, l)

	l.Log(LevelError, "cli/test", "an entry that cannot land")

	closeErr := l.Close()
	if closeErr == nil {
		t.Fatal("a write that failed was swallowed: Close answered nil")
	}

	if !strings.Contains(closeErr.Error(), "write to the log") || !strings.Contains(closeErr.Error(), logPath) {
		t.Errorf("the failure does not say a write to that log is what went wrong: %v", closeErr)
	}
}

// TestInitSaysWhatTheLogItReplacesCouldNot. Re-initialising is where a
// failed log is most likely to be noticed and was most likely to be lost:
// the old logger is closed on the way past, and its answer used to go
// straight into the blank identifier.
func TestInitSaysWhatTheLogItReplacesCouldNot(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.log")
	second := filepath.Join(dir, "second.log")

	if err := Init(first, ""); err != nil {
		t.Fatalf("init the first logger: %v", err)
	}

	globalMu.RLock()
	readOnly(t, global)
	globalMu.RUnlock()

	Error("cli/test", "an entry that cannot land")

	if err := Init(second, ""); err != nil {
		t.Fatalf("init the second logger: %v", err)
	}

	defer func() {
		if err := CloseGlobal(); err != nil {
			t.Errorf("error closing global logger: %v", err)
		}
	}()

	data, err := os.ReadFile(second)
	if err != nil {
		t.Fatalf("failed to read the second log: %v", err)
	}

	if !strings.Contains(string(data), first) {
		t.Errorf("the new log does not say the old one was in trouble, got:\n%s", data)
	}
}

// TestAnErrorIsWrittenTwiceAndEverythingElseOnce.
//
// The errors file earns its place only if it is exactly the errors: a copy
// of the whole log under a second name is a second thing to grep, and a
// file that drops errors is worse than not having it.
func TestAnErrorIsWrittenTwiceAndEverythingElseOnce(t *testing.T) {
	dir := t.TempDir()
	logPath, errPath := filepath.Join(dir, "orbit.log"), filepath.Join(dir, "errors.log")

	l, err := New(logPath, errPath)
	if err != nil {
		t.Fatalf("create the logger: %v", err)
	}

	l.Log(LevelDebug, "task/run", "a debug line")
	l.Log(LevelInfo, "task/run", "an info line")
	l.Log(LevelWarn, "task/run", "a warn line")
	l.Log(LevelError, "task/run", "the phase %s broke", "review")

	if err := l.Close(); err != nil {
		t.Fatalf("close the logger: %v", err)
	}

	read := func(path string) string {
		t.Helper()

		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %q: %v", path, err)
		}

		return string(b)
	}

	full, errs := read(logPath), read(errPath)

	for _, line := range []string{"a debug line", "an info line", "a warn line", "the phase review broke"} {
		if !strings.Contains(full, line) {
			t.Errorf("the log is missing %q", line)
		}
	}

	if !strings.Contains(errs, "the phase review broke") {
		t.Errorf("the errors file is missing the error: %q", errs)
	}

	for _, line := range []string{"a debug line", "an info line", "a warn line"} {
		if strings.Contains(errs, line) {
			t.Errorf("the errors file carries %q, which is not an error", line)
		}
	}
}

// TestALoggerWithNoErrorsFileStillLogsErrors. An empty errPath is a caller
// that has one path, not a caller that wants its errors dropped.
func TestALoggerWithNoErrorsFileStillLogsErrors(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "only.log")

	l, err := New(logPath, "")
	if err != nil {
		t.Fatalf("create the logger: %v", err)
	}

	l.Log(LevelError, "task/run", "the only error")

	if err := l.Close(); err != nil {
		t.Fatalf("close the logger: %v", err)
	}

	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read the log: %v", err)
	}

	if !strings.Contains(string(b), "the only error") {
		t.Errorf("the log is missing the error: %q", b)
	}
}
