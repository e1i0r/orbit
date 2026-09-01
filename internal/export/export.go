// Package export writes the record back out as the JSONL an older Orbit
// kept: one events.jsonl per task, the supervisor thread beside them, and
// the marker saying which repositories a task was worked in.
//
// It is internal/migrate read backwards, and it exists for what the move to
// SQLite took away. A tree of text files was a backup that made itself —
// every event was a line, in a file, that `cat` and `grep` and any program
// ever written could read, and copying the folder was copying the record.
// One binary file is none of that. This is how those properties come back:
// on demand, run before an upgrade, rather than paid for on every write.
//
// What it writes is a tree the migration reads. That is the whole promise —
// an export restored into an empty $ORBIT_HOME comes back as the record it
// was taken from — and it is why the destination is named through
// internal/store rather than built here: the shape has to be the shape that
// package reads, and two descriptions of one layout drift.
package export

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/e1i0r/orbit/internal/db"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// Result is what one export wrote, for the line a reader is shown.
type Result struct {
	Tasks    int
	Events   int
	Messages int
}

// Run writes the record out into dir, which must not already hold anything.
//
// The handle is the store's own rather than one of this package's, for the
// reason migrate.Run gives: one process is one writer, and a second handle
// would have this contending with the command it was typed at.
func Run(s *store.Store, dir, only string) (Result, error) {
	from, err := s.Record()
	if err != nil {
		return Result{}, err
	}

	if err := empty(dir); err != nil {
		return Result{}, err
	}

	to, err := store.New(dir)
	if err != nil {
		return Result{}, err
	}

	return Records(from, to, only)
}

// empty refuses a destination that already holds something.
//
// Both ways of writing an export into a directory with files in it are
// wrong. It either lands on top of a backup somebody took earlier — and half
// of yesterday's record under half of today's is a file nobody can reason
// about — or it lands in a state root, where tasks/<id>/events.jsonl is
// exactly the shape the migration reads, and the next command would take the
// export for a log an older Orbit wrote and copy it back in.
func empty(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("look in %q: %w", dir, err)
	}

	if len(entries) > 0 {
		return fmt.Errorf("%q already holds something, and an export is a whole record: name a directory that is empty or does not exist yet", dir)
	}

	return nil
}

// Records writes what the record holds into the tree rooted at to.
//
// A task that will not read does not stop the ones after it, and what went
// wrong is collected and answered at the end. The reason to run this is
// often that the record is already doubted — `orbit check` says so and
// points here — and on a damaged file the first unreadable task would
// otherwise take every readable one with it, which is the whole of what
// there was left to save.
//
// It still never reports success. The error is what keeps a partial export
// from passing for a whole one: a backup that is missing three tasks and
// says nothing is worse than no backup, because somebody is counting on it.
func Records(from *db.DB, to *store.Store, only string) (Result, error) {
	var (
		out    Result
		failed []error
	)

	ids, err := chosen(from, only)
	if err != nil {
		return out, err
	}

	for _, id := range ids {
		n, err := log(from, to, id)
		if err != nil {
			failed = append(failed, err)
			continue
		}

		out.Tasks++
		out.Events += n

		if err := joins(from, to, id); err != nil {
			failed = append(failed, err)
		}
	}

	if only != "" {
		// A task's export is that task. The supervisor thread belongs to no
		// task — it is one conversation about all of them — so carrying it
		// along would put the whole of it in a directory named after one.
		return out, errors.Join(failed...)
	}

	n, err := thread(from, to)
	if err != nil {
		failed = append(failed, err)
	}

	out.Messages = n

	return out, errors.Join(failed...)
}

// chosen is which tasks are being written out.
//
// A named task the record has never heard of is refused rather than written
// out as an empty file. The reason to name one is that you know it is there,
// so a mistyped id answered with a directory holding nothing would be a
// backup of nothing that looks exactly like a backup.
func chosen(from *db.DB, only string) ([]string, error) {
	ids, err := from.Tasks()
	if err != nil {
		return nil, err
	}

	if only == "" {
		return ids, nil
	}

	if !slices.Contains(ids, only) {
		return nil, fmt.Errorf("the record has no task %q to write out", only)
	}

	return []string{only}, nil
}

// log writes one task's events, and answers how many there were.
//
// A task with no events gets its file all the same. It is a task somebody
// wrote down and has not run, the tree this shape came from had a directory
// for it, and a restore that dropped it would lose the one thing it says:
// that the id is taken.
func log(from *db.DB, to *store.Store, id string) (int, error) {
	events, err := from.Events(id)
	if err != nil {
		return 0, err
	}

	path, err := to.EventsPath(id)
	if err != nil {
		return 0, err
	}

	if err := record.Write(path, events); err != nil {
		return 0, err
	}

	return len(events), nil
}

// joins writes down which repositories a task was worked in.
//
// The events do not always say. A task written by an Orbit older than the
// one that put the repository into task.created has the link nowhere but in
// the row this reads, and that is precisely the task whose only remaining
// copy of it is the export. Without this, a restore comes back as a record
// whose tasks belong to no repository — which is a board with nothing on it.
func joins(from *db.DB, to *store.Store, id string) error {
	paths, err := from.ReposOfTask(id)
	if err != nil {
		return err
	}

	for _, abs := range paths {
		if err := to.JoinRepo(id, abs); err != nil {
			return err
		}
	}

	return nil
}

// thread writes the supervisor conversation, and answers how many turns it
// held.
func thread(from *db.DB, to *store.Store) (int, error) {
	turns, err := from.Messages()
	if err != nil {
		return 0, err
	}

	if err := record.Write(to.SupervisorLogPath(), turns); err != nil {
		return 0, err
	}

	return len(turns), nil
}
