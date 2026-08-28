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

	st.modTime = info.ModTime()

	size := info.Size()
	if size == st.size {
		// The one stat this whole design is built on. Nothing was appended,
		// so there is nothing to open, seek, read or parse, and the last
		// verdict stands — which is also what stops a log nobody can read
		// from being re-read twice a second and from quietly stopping being
		// reported.
		return nil, st.err
	}

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
		// A read that failed gets exactly one more try, on the next
		// refresh, and then the size is committed and the verdict stands.
		//
		// Both halves are needed and they pull opposite ways. The size used
		// to be committed before the read was attempted, so a failure was
		// never retried at all: the bytes it did not read stayed stranded
		// until something else was appended, and if the write that failed
		// was a run's last one, its ending never appeared — the row went on
		// saying "running" over a log that says "finished", and the error
		// beside it went on blaming a fault that may have lasted a
		// millisecond.
		//
		// Retrying forever is the other mistake. A line longer than
		// record.MaxLine — four megabytes — is a log no later read will get
		// past, and re-reading it twice a second is four megabytes of
		// pointless I/O per second per damaged task, for as long as the
		// window is open. One retry recovers a filesystem that blinked and
		// bounds the other case at two.
		if !st.retried {
			st.retried = true
			return nil, err
		}

		st.size, st.retried = size, false

		return nil, err
	}

	st.size, st.offset, st.retried = size, next, false

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
