package record

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

// Unreadable is the kind of the event Read puts in place of a line it could
// not parse. It is synthesised, never written: nothing appends it.
const Unreadable = "record.unreadable"

// Read returns every event in the log, oldest first.
//
// A log that does not exist yet is empty, not an error: a task that has not
// started has nothing to say, and that is a normal state rather than a fault.
//
// A line that will not parse is not silently dropped. Returning fewer events
// than the file holds, with nothing to say so, is a reader lying about its
// own completeness — so the gap is filled with a record.unreadable event
// carrying where in the file the line begins, and whoever is looking can go
// and read that line with their own eyes. The one exception is a last line
// with no newline after it: that is a write interrupted mid-flight rather
// than damage, and it stays dropped.
//
// Everything below works from one size, taken once. Measuring a log twice
// while it is being appended to — the last byte checked against the file as
// it was then, the scan running to whatever the end had become since — reads
// a write that landed in between as a torn line in a file that had already
// been declared whole, and synthesises a record.unreadable for a line that
// was merely still being written.
func Read(path string) ([]Event, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}

	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", path, err)
	}

	size := info.Size()

	whole, err := endsWithNewline(f, size)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}

	s, err := scanEvents(io.LimitReader(f, size), 0)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}

	if s.hasPending && whole {
		s.events = append(s.events, unreadable(s.pending))
	}

	return s.events, nil
}

// scan is what one pass over a log's lines produced.
//
// It is a struct rather than five return values because ReadFrom needs four
// of them and Read needs two, and a signature nobody can read at the call
// site is how the two sides start disagreeing about which is which.
type scan struct {
	events []Event
	// pending is where the last line that would not parse begins, held back
	// one turn because a bad line is only forgivable when it is the last one
	// in an unterminated file; the caller decides what that means once
	// scanning is done. hasPending is whether there is one, which a byte
	// offset of zero cannot say on its own.
	pending    int64
	hasPending bool
	// lastStart is where the final line scanned begins, and lastWasEvent
	// whether that line produced an event. Both exist only for ReadFrom's
	// offset arithmetic, and Read ignores them.
	lastStart    int64
	lastWasEvent bool
}

// scanEvents is the one place that turns a log's lines into events. Read and
// ReadFrom each finish the job differently — Read decides whether a still-
// pending line counts using whole-file knowledge, ReadFrom also needs to
// know how the offset should move — but the classification itself, the part
// most likely to drift if it existed twice, lives here and nowhere else.
//
// base is where in the file r starts, so that every offset this hands back
// names a place in the log rather than a place in whatever slice of it the
// caller happened to read. ReadFrom passes its offset; Read passes zero.
//
// The arithmetic assumes a line is terminated by a bare '\n', which is the
// only terminator Append ever writes to this log; a '\r\n' writer would need
// it adjusted, because bufio.ScanLines strips a trailing '\r' from the token
// before its length is ever measured.
func scanEvents(r io.Reader, base int64) (scan, error) {
	s := scan{lastStart: base}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), MaxLine)

	at := base
	for sc.Scan() {
		s.lastStart, s.lastWasEvent = at, false
		at += int64(len(sc.Bytes())) + 1

		if s.hasPending {
			s.events = append(s.events, unreadable(s.pending))
			s.hasPending = false
		}

		if len(sc.Bytes()) == 0 {
			continue
		}

		var e Event
		if jerr := json.Unmarshal(sc.Bytes(), &e); jerr != nil {
			s.pending, s.hasPending = s.lastStart, true
			continue
		}

		s.events = append(s.events, e)
		s.lastWasEvent = true
	}

	if serr := sc.Err(); serr != nil {
		return scan{}, serr
	}

	return s, nil
}

// unreadable is the event that stands in for a line nobody can parse.
//
// It names the byte the line begins at rather than counting lines, because a
// count is only true of a reader that started at the top. ReadFrom starts
// wherever it left off, so a line number from it meant "the third line of
// what I happened to read this time" — a number that pointed at nothing and
// looked like it pointed at something. A byte offset is the same fact for
// both readers, and `tail -c +N` opens on it.
func unreadable(at int64) Event {
	return Event{
		Kind: Unreadable,
		Text: "this line of the record is not valid JSON and was skipped",
		Data: map[string]string{"byte": strconv.FormatInt(at, 10)},
	}
}

// endsWithNewline reports whether the last line of the file was finished, as
// of the size the caller measured.
//
// The size is a parameter rather than a stat of its own so that every
// question this package asks of a log that is being appended to is asked
// about the same file: a second stat here is a second file, and the two
// answers together describe one that never existed.
//
// It reads one byte and leaves the read offset alone, so the scanner still
// starts from the beginning.
func endsWithNewline(f *os.File, size int64) (bool, error) {
	if size == 0 {
		return false, nil
	}

	last := make([]byte, 1)
	if _, err := f.ReadAt(last, size-1); err != nil {
		return false, err
	}

	return last[0] == '\n', nil
}
