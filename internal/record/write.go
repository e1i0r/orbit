package record

// Writing a whole log at once, which is what an export is: rows of the
// database turned back into the lines they were written as.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Write puts a whole log down at path, replacing whatever was there.
//
// It is Append's bytes and not its file. The same marshalling, so a log
// written here and a log appended to a line at a time are the same file —
// and one write and one flush for the lot, where going through Append would
// be an open, a write and an fsync per event. A record of a hundred thousand
// events is a hundred thousand fsyncs that way round, and the point of an
// export is that it is run before an upgrade rather than overnight.
//
// Nothing here refuses a line for its length the way Append does, and that
// is not an omission: the record refuses an event over MaxLine on the way
// in, so no row it holds can come back out as a line too long to read.
func Write(path string, events []Event) (err error) {
	var body []byte

	for _, e := range events {
		line, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("encode event %q: %w", e.Kind, err)
		}

		body = append(body, line...)
		body = append(body, '\n')
	}

	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return fmt.Errorf("create %q: %w", filepath.Dir(path), err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileMode)
	if err != nil {
		return fmt.Errorf("open %q: %w", path, err)
	}
	// A copy of the record that was not flushed is the copy somebody made
	// before the upgrade that then went wrong, so the close is not dropped
	// on the floor either: it becomes the error when nothing worse happened
	// first.
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close %q: %w", path, closeErr)
		}
	}()

	if _, err := f.Write(body); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("flush %q: %w", path, err)
	}

	return nil
}
