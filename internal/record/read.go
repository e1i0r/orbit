package record

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
)

// Unreadable is the kind of the event Read puts where a line it could not
// parse used to be. It is synthesised, never written: nothing appends it.
const Unreadable = "record.unreadable"

// Read returns every event in the log, oldest first.
//
// A log that does not exist yet is empty, not an error: a task that has not
// started has nothing to say, and that is a normal state rather than a fault.
//
// A line that will not parse is not silently dropped. Returning fewer events
// than the file holds, with nothing to say so, is a reader lying about its
// own completeness — so the gap is filled with a record.unreadable event
// carrying the line number, and whoever is looking can go and read that line
// with their own eyes. The one exception is a last line with no newline
// after it: that is a write interrupted mid-flight rather than damage, and
// it stays dropped.
func Read(path string) ([]Event, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	whole, err := endsWithNewline(f)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}

	var events []Event
	// pending holds the line number of a line that would not parse, kept
	// back one turn because a bad line is only forgivable when it is the
	// last one in an unterminated file. The next line to arrive proves it
	// was not.
	pending := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), MaxLine)
	line := 0
	for scanner.Scan() {
		line++
		if pending > 0 {
			events = append(events, unreadable(pending))
			pending = 0
		}
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var e Event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			pending = line
			continue
		}
		events = append(events, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	if pending > 0 && whole {
		events = append(events, unreadable(pending))
	}
	return events, nil
}

// unreadable is the event that stands in for a line nobody can parse.
func unreadable(line int) Event {
	return Event{
		Kind: Unreadable,
		Text: "this line of the record is not valid JSON and was skipped",
		Data: map[string]string{"line": strconv.Itoa(line)},
	}
}

// endsWithNewline reports whether the last line of the file was finished.
//
// It reads one byte at the end and leaves the read offset alone, so the
// scanner still starts from the beginning.
func endsWithNewline(f *os.File) (bool, error) {
	info, err := f.Stat()
	if err != nil {
		return false, err
	}
	if info.Size() == 0 {
		return false, nil
	}
	last := make([]byte, 1)
	if _, err := f.ReadAt(last, info.Size()-1); err != nil {
		return false, err
	}
	return last[0] == '\n', nil
}
