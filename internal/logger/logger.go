// Package logger provides structured internal diagnostic logging for Orbit.
//
// All logs are written to an internal file (e.g. ~/.orbit/logs/orbit.log)
// rather than polluting terminal standard output, ensuring clean TUI rendering.
//
// Format:
//
//	[2026-08-25T00:54:12.123Z] [INFO] [cli/run] starting task ACME-1 in repo payments on flow careful
package logger

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level indicates the severity of a log entry.
type Level int

// Log level constants.
const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

const timeFormat = "2006-01-02T15:04:05.000Z07:00"

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

// Logger handles structured logging to a dedicated file.
type Logger struct {
	mu sync.Mutex
	// file takes every line; errFile takes the ones at LevelError, which
	// are therefore written twice on purpose. An error read in isolation
	// says what broke; the same error read in the log says what the
	// program was doing when it broke, and both questions get asked.
	file    *os.File
	errFile *os.File
	path    string
	errPath string
	// failed is the first write that did not land, kept because Log has
	// nowhere to return one: it is called from everywhere, on paths whose
	// callers have no business handling a logging fault, and the two
	// descriptors that are left — a cockpit redrawing the terminal, an mcp
	// server whose stdout is a client's JSON-RPC stream — are ones a stray
	// line would corrupt. So the failure waits here and leaves through
	// Close, which is somebody asking.
	failed error
}

var (
	globalMu sync.RWMutex
	global   *Logger
)

// Init initializes the default global logger pointing to the given file paths.
func Init(logPath, errPath string) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	// What the log being replaced has to say is said into the log replacing
	// it, below. Dropping it here would throw away, one line before the new
	// file is open, the very failures the field above exists to keep.
	var previous error
	if global != nil {
		previous = global.Close()
	}

	l, err := New(logPath, errPath)
	if err != nil {
		return fmt.Errorf("init logger at %q: %w", logPath, err)
	}

	global = l

	if previous != nil {
		l.Log(LevelWarn, "logger", "the log this one replaces did not close cleanly: %v", previous)
	}

	return nil
}

// New creates a new Logger writing every line to logPath and the errors
// again to errPath.
//
// An empty errPath is a logger with no error file, which is what a caller
// that only has one path gets. It is not an error: the errors still reach
// logPath, which is the file that must exist.
func New(logPath, errPath string) (*Logger, error) {
	f, err := openLog(logPath)
	if err != nil {
		return nil, err
	}

	l := &Logger{file: f, path: logPath}

	if errPath == "" {
		return l, nil
	}

	ef, err := openLog(errPath)
	if err != nil {
		// The log that did open is closed again rather than leaked. A
		// constructor that answers with an error and a descriptor nobody
		// holds is how a process runs out of them overnight.
		return nil, errors.Join(err, f.Close())
	}

	l.errFile, l.errPath = ef, errPath

	return l, nil
}

// openLog opens one append-only log file, creating its directory.
func openLog(path string) (*os.File, error) {
	if path == "" {
		return nil, fmt.Errorf("log path cannot be empty")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create log directory %q: %w", dir, err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log file %q: %w", path, err)
	}

	return f, nil
}

// Close closes the logger and its underlying file descriptor, and answers
// what went wrong on the way: the first entry that could not be written, as
// well as the close itself. A caller that only wanted the descriptor back
// gets told, at the one moment it can still be told, that the log it has
// been writing all along has been going nowhere.
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	failed := l.failed
	l.failed = nil

	if l.file == nil {
		return failed
	}

	err := l.file.Close()
	l.file = nil

	var errErr error
	if l.errFile != nil {
		errErr = l.errFile.Close()
		l.errFile = nil
	}

	return errors.Join(failed, err, errErr)
}

// Log writes a leveled log entry with timestamp, level, module tag, and formatted message.
func (l *Logger) Log(lvl Level, module, format string, args ...any) {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return
	}

	if module == "" {
		module = "orbit"
	}

	ts := time.Now().UTC().Format(timeFormat)
	msg := fmt.Sprintf(format, args...)

	entry := fmt.Sprintf("[%s] [%s] [%s] %s\n", ts, lvl.String(), module, msg)
	l.write(l.file, l.path, entry)

	if lvl == LevelError && l.errFile != nil {
		l.write(l.errFile, l.errPath, entry)
	}
}

// write puts one entry in one file and keeps the first refusal.
//
// The first failure is the one kept: a log that has stopped working fails
// on every line after it, and the hundredth message would say nothing the
// first does not.
func (l *Logger) write(f *os.File, path, entry string) {
	if _, err := f.WriteString(entry); err != nil && l.failed == nil {
		l.failed = fmt.Errorf("write to the log at %q: %w", path, err)
	}
}

// Debug logs a debug message with module tag to the default logger.
func Debug(module, format string, args ...any) {
	globalMu.RLock()
	defer globalMu.RUnlock()

	if global != nil {
		global.Log(LevelDebug, module, format, args...)
	}
}

// Info logs an informational message with module tag to the default logger.
func Info(module, format string, args ...any) {
	globalMu.RLock()
	defer globalMu.RUnlock()

	if global != nil {
		global.Log(LevelInfo, module, format, args...)
	}
}

// Warn logs a warning message with module tag to the default logger.
func Warn(module, format string, args ...any) {
	globalMu.RLock()
	defer globalMu.RUnlock()

	if global != nil {
		global.Log(LevelWarn, module, format, args...)
	}
}

// Error logs an error message with module tag to the default logger.
func Error(module, format string, args ...any) {
	globalMu.RLock()
	defer globalMu.RUnlock()

	if global != nil {
		global.Log(LevelError, module, format, args...)
	}
}

// CloseGlobal closes the active global logger.
func CloseGlobal() error {
	globalMu.Lock()
	defer globalMu.Unlock()

	if global != nil {
		err := global.Close()
		global = nil

		return err
	}

	return nil
}
