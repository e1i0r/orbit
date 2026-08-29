package record

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MaxLine is the longest line the log will hold, and the same number the
// reader sizes its buffer from.
//
// Both sides have to agree on it. If Read capped a line at 4 MB while Append
// wrote anything it was handed — and task.Run hands it an engine's entire
// stdout — one oversized event would make the whole task's record
// unreadable: the scanner returns bufio.ErrTooLong and every event before
// the offending line is lost with it. A refused write the caller can see
// beats a log that cannot be read back.
const MaxLine = 4 << 20

// dirMode and fileMode match the state root the log lives in: the record is
// the whole truth about a task, including every word the engines printed,
// and it is nobody's business but its owner's. store states the same two
// modes for the same reason; this package takes a path, not a Store, so it
// cannot borrow them.
const (
	dirMode  os.FileMode = 0o700
	fileMode os.FileMode = 0o600
)

// Append adds one event to the log, creating it if needed.
//
// The whole line is marshalled before the file is opened so that a failure to
// encode cannot leave a half-written line behind, and the newline is part of
// the same write for the same reason.
func Append(path string, e Event) (err error) {
	if e.At.IsZero() {
		e.At = time.Now()
	}

	line, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("encode event %q: %w", e.Kind, err)
	}

	line = append(line, '\n')
	if len(line) > MaxLine {
		return fmt.Errorf("event %q is %d bytes, over the %d the record can read back: refusing to write a line that would make the whole log unreadable", e.Kind, len(line), MaxLine)
	}

	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return fmt.Errorf("create %q: %w", filepath.Dir(path), err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileMode)
	if err != nil {
		return fmt.Errorf("open %q: %w", path, err)
	}
	// The log is the only thing that survives the process, so a failure to
	// flush it is not a detail to drop on the floor: it is the difference
	// between a record and a rumour. The close error becomes the returned
	// error when nothing worse happened first.
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close %q: %w", path, closeErr)
		}
	}()

	if _, err := f.Write(line); err != nil {
		return fmt.Errorf("append to %q: %w", path, err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("flush %q: %w", path, err)
	}

	return nil
}
