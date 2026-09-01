// Package migrate fills the record from the files an older Orbit wrote.
//
// It reads and it never deletes. Every tasks/<id>/events.jsonl and the
// supervisor.jsonl beside them stay exactly where they are, so the previous
// binary keeps working and there is no moment at which the only copy of a
// run is the new one. Removing them is a later version's job, done once,
// when the database has been the record for long enough to trust.
//
// It is safe to run before every command. What it copies is the part of each
// file the record does not have yet, which on the second run is nothing.
package migrate

import (
	"errors"
	"fmt"

	"github.com/e1i0r/orbit/internal/db"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// Result is what one pass moved, for the line that gets logged about it.
type Result struct {
	Tasks    int
	Events   int
	Messages int
}

// Moved reports whether this pass had anything to do.
func (r Result) Moved() bool { return r.Events > 0 || r.Messages > 0 }

// String is the sentence a reader sees in the log.
func (r Result) String() string {
	return fmt.Sprintf("%d event(s) from %d task(s) and %d supervisor turn(s)", r.Events, r.Tasks, r.Messages)
}

// Records copies whatever the files hold and the record does not.
//
// A task that cannot be read does not stop the ones after it. The whole
// point of running this before every command is that it finishes; a
// migration that gives up on the first damaged file leaves a state root
// half moved, and half moved is the one shape nobody can reason about.
// What went wrong is collected and answered at the end, after the work.
func Records(s *store.Store, d *db.DB) (Result, error) {
	var (
		out    Result
		failed []error
	)

	ids, err := s.TaskIDs()
	if err != nil {
		return out, fmt.Errorf("list the tasks to migrate: %w", err)
	}

	for _, id := range ids {
		n, err := task(s, d, id)
		if err != nil {
			failed = append(failed, err)
			continue
		}

		if n > 0 {
			out.Tasks++
			out.Events += n
		}
	}

	n, err := thread(s, d)
	if err != nil {
		failed = append(failed, err)
	}

	out.Messages = n

	return out, errors.Join(failed...)
}

// task copies the part of one task's log the record has not got.
//
// Where it got to is a count rather than a mark left in the file. The log is
// appended to and never reordered, so the first n lines of it are the n
// events already in the record, and the rest are what is left to do. A mark
// would be a second thing to keep in step with the first.
func task(s *store.Store, d *db.DB, id string) (int, error) {
	path, err := s.EventsPath(id)
	if err != nil {
		return 0, fmt.Errorf("find the log of %q: %w", id, err)
	}

	// A damaged line is read as record.unreadable and carried across as one,
	// which is what internal/record already does for every other reader: the
	// event that says a line could not be read is itself part of the record.
	events, err := record.Read(path)
	if err != nil {
		return 0, fmt.Errorf("read the log of %q: %w", id, err)
	}

	done, err := d.EventCount(id)
	if err != nil {
		return 0, err
	}

	if done >= len(events) {
		return 0, nil
	}

	for _, e := range events[done:] {
		if err := d.Append(id, e); err != nil {
			return 0, err
		}
	}

	return len(events) - done, nil
}

// thread copies the part of the supervisor conversation the record has not
// got, by the same count and for the same reason.
func thread(s *store.Store, d *db.DB) (int, error) {
	turns, err := record.Read(s.SupervisorLogPath())
	if err != nil {
		return 0, fmt.Errorf("read the supervisor thread: %w", err)
	}

	done, err := d.MessageCount()
	if err != nil {
		return 0, err
	}

	if done >= len(turns) {
		return 0, nil
	}

	for _, e := range turns[done:] {
		if err := d.AppendMessage(e); err != nil {
			return 0, err
		}
	}

	return len(turns) - done, nil
}

// Run opens the record, fills it from the files, and lets go of it again.
//
// The handle is opened here rather than handed in so that a caller wanting
// the migration does not have to know where the record lives or hold it open
// across a command it is not otherwise using.
func Run(s *store.Store) (Result, error) {
	d, err := db.Open(s.DBPath())
	if err != nil {
		return Result{}, err
	}

	out, err := Records(s, d)

	return out, errors.Join(err, d.Close())
}
