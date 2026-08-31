package index

// Building the whole index out of the record, which is also what keeping it
// up to date is: there is one operation, and doing it twice in a row is the
// second one folding nothing.

import (
	"errors"
	"fmt"
)

// Source is where the projector reads from. It is an interface and not
// *store.Store so that this package imports internal/record and nothing else
// of Orbit's — the index is derived from the record, and a package that had
// to know the shape of the state tree to fold an event would be deriving it
// from something else as well.
type Source interface {
	TaskIDs() ([]string, error)
	EventsPath(taskID string) (string, error)
}

// Build folds every task's log into the index, from wherever each one was
// left, and answers how many events that was in total.
//
// A task whose log will not fold costs itself and not the build: the rest
// of the index is still worth having, and what went wrong comes back joined
// so that a caller can say so. This is the whole of keeping the index
// current — it is called on start, it reads the tail of what changed, and on
// a tree nobody has touched since the last run it reads nothing at all.
func Build(x *Index, src Source) (int, error) {
	ids, err := src.TaskIDs()
	if err != nil {
		return 0, err
	}

	var (
		folded int
		failed []error
	)

	for _, id := range ids {
		path, err := src.EventsPath(id)
		if err != nil {
			failed = append(failed, err)
			continue
		}

		n, err := x.Project(id, path)
		if err != nil {
			failed = append(failed, err)
			continue
		}

		folded += n
	}

	return folded, errors.Join(failed...)
}

// Rebuild throws the index away and folds it again from the logs.
//
// It is the repair for an index somebody has reason to distrust, and it is
// cheap in the way only a derived thing can be: the record is untouched, so
// the worst this can cost is the time it takes to read the logs again.
func Rebuild(x *Index, src Source) (int, error) {
	if err := x.empty(); err != nil {
		return 0, err
	}

	if _, err := x.db.Exec(schema); err != nil {
		return 0, fmt.Errorf("create the tables of %q: %w", x.path, err)
	}

	return Build(x, src)
}
