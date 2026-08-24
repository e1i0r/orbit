package board

// The two things a poll does to one thing: read the tail of one task's log,
// and find out what tasks one repository has. Refresh and Rescan are the
// loops; these are what they call.

import (
	"errors"
	"fmt"
	"os"

	"github.com/e1i0r/orbit/internal/record"
)

// poll reads whatever has been appended to one task's log since the last
// refresh, and returns the verdict on having done so.
func (r *Reader) poll(st *taskState) ([]record.Event, error) {
	info, err := os.Stat(st.path)
	if errors.Is(err, os.ErrNotExist) {
		// A task written down whose run has never started has no log at
		// all, and that is an answer rather than a fault — the same
		// treatment record.Read gives it.
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", st.path, err)
	}

	size := info.Size()
	if size == st.size {
		// The one stat this whole design is built on. Nothing was appended,
		// so there is nothing to open, seek, read or parse, and the last
		// verdict stands — which is also what stops a log nobody can read
		// from being re-read twice a second and from quietly stopping being
		// reported.
		return nil, st.err
	}
	st.size = size

	if size < st.offset {
		// Shorter than what has already been read, so this log was replaced
		// rather than appended to. ReadFrom answers that on its own by
		// starting from the top; what it cannot know is that the events read
		// from the log it replaced are no longer this task's history, and
		// folding both together would show one task attempted twice. So they
		// go with the offset. Clearing seen is what makes the caller fold
		// again even if the replacement has nothing in it yet.
		st.offset, st.events, st.seen = 0, nil, false
	}

	// The offset is only ever what ReadFrom answered, never arithmetic of
	// this package's own: ReadFrom advances past a complete,
	// newline-terminated line and no further, and a torn final line is a
	// write in flight rather than damage. Tracking bytes here would defeat
	// exactly that property.
	events, next, err := record.ReadFrom(st.path, st.offset)
	if err != nil {
		return nil, err
	}
	st.offset = next
	return events, nil
}

// list is every task directory of one repository, in the order the rows are
// drawn — os.ReadDir answers in filename order, which for task ids is the
// order they are listed in everywhere else.
func (r *Reader) list(rs *repoState) ([]string, error) {
	dir, err := r.store.TasksDir(rs.path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		// A repository nobody has written a task against yet. Computing a
		// path creates nothing, so this directory is genuinely absent until
		// the first task, and "no tasks" is an answer rather than a fault.
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list the tasks of %q: %w", rs.name, err)
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	return ids, nil
}
