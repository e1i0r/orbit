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
// One size, taken once, decides all three things: which bytes are read, what
// the last byte is checked to be, and what offset comes back. Two sizes —
// the returned one measured before the last byte is looked at, the scan then
// running to whatever the end has become — return an offset behind what has
// already been handed out for a log appended to in between, and the next
// call reads those events a second time. A poller that folds what it is
// given would see one attempt twice.
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

	whole, err := endsWithNewline(f, size)
	if err != nil {
		return nil, 0, fmt.Errorf("read %q: %w", path, err)
	}

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, 0, fmt.Errorf("seek %q: %w", path, err)
	}

	s, err := scanEvents(io.LimitReader(f, size-offset), offset)
	if err != nil {
		return nil, 0, fmt.Errorf("read %q: %w", path, err)
	}

	if whole {
		if s.hasPending {
			s.events = append(s.events, unreadable(s.pending))
		}

		return s.events, size, nil
	}
	// The final line has no newline yet. Whatever it produced — a parsed
	// event, or nothing because it was left pending — is withheld until a
	// later call sees it finished, and the offset stays at where it began.
	if s.lastWasEvent {
		s.events = s.events[:len(s.events)-1]
	}

	return s.events, s.lastStart, nil
}
