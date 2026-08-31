// Package index is the derived index over the record: SQLite holding the
// relations the record has and a directory tree cannot keep — which
// repositories a task was worked in, what happened in which order, across
// every task at once.
//
// Nothing here is authoritative and none of it is a second truth. The record
// stays what internal/record says it is, and every row in this database was
// folded out of an event on the way past. The index holds nothing the record
// does not, which is why the repair for an index that is wrong is to delete
// the file: the next run builds it again from the logs.
//
// There is one write path. Nothing inserts a row because it felt like it —
// an event is appended to a task's log, and the projector folds that event
// in. Drift between the two is not a bug that has to be found and fixed; it
// is a state with no way to occur.
package index

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // the pure-Go driver: releases build with CGO_ENABLED=0
)

// Index is an open handle on the derived database.
type Index struct {
	db   *sql.DB
	path string
}

// Open opens the index at path, creating it, and brings it to the current
// schema — which, when the schema it finds is an older one, means throwing
// it away and starting empty. That is the whole benefit of the record being
// the truth: a schema change costs a rebuild and never a migration written
// against live data.
//
// The pragmas are the ones a file read by several processes at once needs.
// WAL lets the window keep reading while a run writes, and a busy timeout
// turns the contention that remains into a wait instead of an error: ten
// tasks folding a few hundred bytes each is human-paced, not machine-paced.
func Open(path string) (*Index, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}

	x := &Index{db: db, path: path}

	if err := x.migrate(); err != nil {
		return nil, err
	}

	return x, nil
}

// Close closes the database.
func (x *Index) Close() error {
	if err := x.db.Close(); err != nil {
		return fmt.Errorf("close %q: %w", x.path, err)
	}

	return nil
}

// Path is where the index file is.
func (x *Index) Path() string { return x.path }
