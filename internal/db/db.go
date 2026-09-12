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
	"errors"
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
	sql *sql.DB

	path string

	// ahead is set when the file was written by an orbit that knows more
	// than this one. The handle is open and every read answers; every write
	// is refused with this, and the refusal is the whole reason the field
	// exists. A zero AheadError means there is nothing ahead.
	ahead AheadError
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

	handle, err := connect(path, dsn)
	if err != nil {
		return nil, err
	}

	d := &DB{sql: handle, path: path}

	// The database's own mode goes on before the migration writes anything
	// into it. Ping is what makes the file: the handle above is lazy, and a
	// chmod of a path nothing has created yet is an ENOENT.
	if fresh {
		if err := handle.Ping(); err != nil {
			return nil, errors.Join(fmt.Errorf("open %q: %w", path, err), handle.Close())
		}

		if err := os.Chmod(path, fileMode); err != nil {
			return nil, errors.Join(fmt.Errorf("set the mode of %q: %w", path, err), handle.Close())
		}
	}

	if err := d.migrate(); err != nil {
		var ahead AheadError
		if !errors.As(err, &ahead) {
			return nil, errors.Join(err, handle.Close())
		}

		// The record was written by an orbit that knows more than this
		// one, and no migration can bring it back. What is left is most
		// of the program: reopen it read-only and let every read answer.
		//
		// The handle has to be let go of rather than kept, because the
		// one above is open for writing and a write is what must not be
		// possible.
		if err := handle.Close(); err != nil {
			return nil, fmt.Errorf("close %q: %w", path, err)
		}

		reading, err := connect(path, readDSN(path))
		if err != nil {
			return nil, err
		}

		// The handle is lazy, so a record that cannot even be looked at
		// — a log orphaned without its index beside it, say — would only
		// fail at the first read, far from what caused it. Ask for the
		// connection now, while the cause is still in reach.
		if err := reading.Ping(); err != nil {
			return nil, errors.Join(fmt.Errorf("read %q with the schema ahead of this orbit: %w", path, err), reading.Close())
		}

		return &DB{sql: reading, path: path, ahead: ahead}, nil
	}

	// And the two files SQLite keeps beside it, which the first write is
	// what creates. The -wal holds committed events until the process that
	// wrote them closes, so a record at 0600 with its log at 0644 is the
	// record readable by anyone with an account on the machine.
	if fresh {
		for _, beside := range []string{path + "-wal", path + "-shm"} {
			if err := os.Chmod(beside, fileMode); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, errors.Join(fmt.Errorf("set the mode of %q: %w", beside, err), handle.Close())
			}
		}
	}

	return d, nil
}

// connect is one handle on the record, asked for either way round.
func connect(path, dsn string) (*sql.DB, error) {
	handle, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}

	// One process is one writer. A pool would have a task contending with
	// itself for the single write lock SQLite has, which is a queue behind a
	// queue and buys nothing: the events of one task are written in order by
	// one goroutine anyway.
	handle.SetMaxOpenConns(1)

	return handle, nil
}

// readDSN is what a process opens with when the record is ahead of it.
//
// mode=ro is why this exists at all: the refusal to write is SQLite's and
// not this package's, so a write that slipped past the guard below would
// still be refused by the file rather than landing in a shape nobody knows.
// busy_timeout stays, because a reader still has to wait its turn behind a
// writer that is in the middle of appending an event.
func readDSN(path string) string {
	return fmt.Sprintf("file:%s?mode=ro&_pragma=busy_timeout(%d)", path, busyTimeoutMS)
}

// Ahead answers whether the record was written by an orbit that knows more
// than this one, in which case the handle is read-only and every read
// answers. The refusal a write gets carries the same news and names the way
// out: `orbit upgrade`.
func (d *DB) Ahead() bool {
	return d.ahead.Found != 0
}

// writable is what a write in this package asks for first, before it begins
// anything. A record ahead of the binary answers every read, and this is the
// line that keeps that from ever becoming a write.
func (d *DB) writable() error {
	if d.Ahead() {
		return d.ahead
	}

	return nil
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
