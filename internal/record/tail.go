package record

// ReadFrom is Read's incremental sibling. Read exists to show a whole
// task's history at once; ReadFrom exists so a poller can ask "what
// changed since I last looked" without re-scanning what it already saw —
// which is what makes polling cheap enough to prefer over watching the
// filesystem.

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// ReadFrom returns every event appended to path after offset, and the
// offset to pass back in on the next call.
//
// The returned offset only ever advances past a complete, newline-terminated
// line. A writer mid-append leaves the log's final line torn — no trailing
// newline yet — and that is a write in flight, not damage. endsWithNewline
// (read.go) is how Read already tells the two apart; ReadFrom asks it the
// same question and, when the last line is not finished, holds the offset
// at wherever that line began and drops any event it would otherwise have
// produced for it. The next call rereads it once the newline lands.
//
// offset == 0 on a file that does not exist yet returns (nil, 0, nil),
// matching Read: nothing has happened yet, which is not an error. An offset
// past the end of a file that has shrunk means the log was replaced under
// the reader rather than merely appended to; ReadFrom starts over from the
// top, so the caller can notice the returned offset moved backwards instead
// of silently missing everything that came before.
func ReadFrom(path string, offset int64) ([]Event, int64, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, 0, fmt.Errorf("stat %q: %w", path, err)
	}
	size := info.Size()
	if offset > size {
		offset = 0
	}
	if offset == size {
		return nil, offset, nil
	}

	whole, err := endsWithNewline(f)
	if err != nil {
		return nil, 0, fmt.Errorf("read %q: %w", path, err)
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, 0, fmt.Errorf("seek %q: %w", path, err)
	}

	events, pending, lastLen, lastWasEvent, err := scanEvents(f)
	if err != nil {
		return nil, 0, fmt.Errorf("read %q: %w", path, err)
	}

	if whole {
		if pending > 0 {
			events = append(events, unreadable(pending))
		}
		return events, size, nil
	}
	// The final line has no newline yet. Whatever it produced — a parsed
	// event, or nothing because it was left pending — is withheld until a
	// later call sees it finished, and the offset stays at where it began.
	//
	// size - int64(lastLen) assumes the line is terminated by a bare '\n',
	// which is the only terminator Append ever writes to this log; a '\r\n'
	// writer would need this arithmetic adjusted, because bufio.ScanLines
	// strips a trailing '\r' from the token before lastLen ever sees it.
	if lastWasEvent {
		events = events[:len(events)-1]
	}
	return events, size - int64(lastLen), nil
}
