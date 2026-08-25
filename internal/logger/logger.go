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
	mu   sync.Mutex
	file *os.File
	path string
}

var (
	globalMu sync.RWMutex
	global   *Logger
)

// Init initializes the default global logger pointing to the given file path.
func Init(logPath string) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	if global != nil {
		_ = global.Close()
	}

	l, err := New(logPath)
	if err != nil {
		return fmt.Errorf("init logger at %q: %w", logPath, err)
	}
	global = l
	return nil
}

// New creates a new Logger writing to logPath.
func New(logPath string) (*Logger, error) {
	if logPath == "" {
		return nil, fmt.Errorf("log path cannot be empty")
	}

	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create log directory %q: %w", dir, err)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log file %q: %w", logPath, err)
	}

	return &Logger{
		file: f,
		path: logPath,
	}, nil
}

// Close closes the logger and its underlying file descriptor.
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
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
	if _, err := l.file.WriteString(entry); err != nil {
		return
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
