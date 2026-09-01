// Package db is the record: every event Orbit has ever written, and the
// relations folded out of them, in one SQLite file.
//
// It replaces a tree of append-only JSONL files. What that gave up is a
// record readable with `cat`, and what it bought is the ability to ask a
// question that crosses tasks without opening every log to answer it. The
// bytes are the same either way — `orbit export` writes them back out — and
// 95% of them are prose an engine printed that no query ever filters on.
//
// Two rules hold this together, and both were measured rather than assumed:
//
//   - A transaction covers one insert and nothing else. Never one held open
//     across the work an engine is doing. One process doing that stops every
//     other task for as long as it holds the lock.
//   - A write refused for want of a turn asks again. The caller has nowhere
//     else to put the event, so a SQLITE_BUSY that is merely reported is an
//     event that is gone with nothing to say so.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // the pure-Go driver, so releases stay CGO_ENABLED=0
)

// dirMode and fileMode match the state root the database lives in: the
// record is the whole truth about a task, including every word the engines
// printed, and it is nobody's business but its owner's.
const (
	dirMode  os.FileMode = 0o700
	fileMode os.FileMode = 0o600
)

// DB is an open handle on the record.
type DB struct {
	sql  *sql.DB
	path string
}

// Open opens the record, creating and migrating it if it is not there.
//
// Whether the file already existed decides which pragmas are asked for, and
// that distinction is load-bearing rather than an optimisation: see joinDSN.
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return nil, fmt.Errorf("create %q: %w", filepath.Dir(path), err)
	}

	_, statErr := os.Stat(path)

	fresh := os.IsNotExist(statErr)
	if statErr != nil && !fresh {
		return nil, fmt.Errorf("stat %q: %w", path, statErr)
	}

	dsn := joinDSN(path)
	if fresh {
		dsn = createDSN(path)
	}

	handle, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}

	// One process is one writer. A pool would have a task contending with
	// itself for the single write lock SQLite has, which is a queue behind a
	// queue and buys nothing: the events of one task are written in order by
	// one goroutine anyway.
	handle.SetMaxOpenConns(1)

	d := &DB{sql: handle, path: path}

	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("%w, and closing after it: %w", err, handle.Close())
	}

	if fresh {
		if err := os.Chmod(path, fileMode); err != nil {
			return nil, fmt.Errorf("%w, and closing after it: %w", err, handle.Close())
		}
	}

	return d, nil
}

// createDSN is what the process that makes the file opens with.
//
// journal_mode is here and nowhere else. Setting it is a write, so a process
// that asks for it on every connect cannot open the database at all while
// another one holds the write lock — the failure looks like corruption and is
// not. WAL is a property of the file and stays in it, so it is asked for once.
func createDSN(path string) string {
	return joinDSN(path) + "&_pragma=journal_mode(WAL)"
}

// joinDSN is what every process after the first opens with.
//
// synchronous is FULL rather than NORMAL because NORMAL does not flush on
// commit: a power cut takes the last transactions with it. Measured at ten
// parallel writers, FULL costs about half the throughput of a number already
// two orders of magnitude past anything Orbit produces, which makes
// durability free here.
func joinDSN(path string) string {
	return fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(%d)&_pragma=synchronous(FULL)&_pragma=foreign_keys(1)",
		path, busyTimeoutMS,
	)
}

// Close lets go of the record.
func (d *DB) Close() error {
	if err := d.sql.Close(); err != nil {
		return fmt.Errorf("close %q: %w", d.path, err)
	}

	return nil
}

// Path is where the record is, for a caller that has to name it to a human.
func (d *DB) Path() string { return d.path }
