package db

// Whether the file is still the file.
//
// A log made of lines breaks one line at a time, and every reader Orbit has
// already knows what to do about that: internal/record answers a line it
// cannot parse with an event saying so, and the rest of the log is read. A
// database breaks differently. A page torn mid-write takes whatever else
// that page held with it — rows of another task, half an index — and nothing
// about reading it afterwards says anything is missing. So it has to be
// asked, and this is what asks.

import (
	"errors"
	"fmt"

	"modernc.org/sqlite"
)

// sound is what integrity_check answers when it found nothing wrong: one
// row, one word.
const sound = "ok"

// The two result codes that are about the file rather than about the
// statement: SQLITE_CORRUPT and SQLITE_NOTADB. They are written here as
// numbers because the names live in modernc.org/sqlite/lib, a hundred
// thousand generated lines to import for two integers, and because these two
// are part of SQLite's interface — the file format is documented as
// backwards compatible, and so are the codes that describe it.
const (
	codeCorrupt = 11
	codeNotADB  = 26
)

// Check is everything SQLite finds wrong with the record, and nothing at all
// when it finds nothing.
//
// What it finds comes back as an answer and not as an error. Damage is a
// fact about the file rather than a failure of the asking, and the caller is
// a command that prints the list; an error here is reserved for not having
// been able to look.
//
// It reads every page, so it costs what the file weighs. That is why it is a
// question somebody asks rather than something every open does, and why the
// setting that makes every open ask it is off until somebody turns it on.
//
// SQLite stops the list at a hundred problems of its own accord, which is
// far past the point where the answer stops changing: a record with two
// torn pages and a record with two hundred are both restored from the last
// export.
func (d *DB) Check() ([]string, error) {
	found, err := d.integrity()
	if malformed(err) {
		// SQLite's own sentence, handed on as the one thing it found. It
		// arrives as an error rather than as a row because the pragma could
		// not finish reading the file it was asked about, which is the news
		// itself and not a failure to deliver it.
		return []string{err.Error()}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("check %q: %w", d.path, err)
	}

	return found, nil
}

// malformed says whether an error is SQLite's answer about the file rather
// than about the asking.
//
// Damage large enough to stop the pragma mid-page comes back this way, and
// damage small enough for it to finish comes back as rows; a reader is owed
// the same words either way. Everything else — a handle that is closed, a
// disk that will not read — is a failure to look, and stays an error, so the
// command can tell a broken record from an unreachable one.
func malformed(err error) bool {
	var e *sqlite.Error
	if !errors.As(err, &e) {
		return false
	}

	// An extended code carries its primary code in the low byte, so
	// SQLITE_CORRUPT_INDEX and the rest of that family are corruption like
	// any other.
	switch e.Code() & 0xff {
	case codeCorrupt, codeNotADB:
		return true
	default:
		return false
	}
}

// integrity asks the pragma and collects what it answered, with SQLite's
// errors left exactly as they arrived so that Check can tell them apart.
func (d *DB) integrity() ([]string, error) {
	rows, err := d.sql.Query(integrityCheck)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var found []string

	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return nil, err
		}

		if line == sound {
			continue
		}

		found = append(found, line)
	}

	return found, rows.Err()
}

// Totals is how much the record holds.
type Totals struct {
	Tasks    int
	Events   int
	Messages int
}

// Totals counts what is in the record, for a reader who asked after its
// health and is owed more than a word.
func (d *DB) Totals() (Totals, error) {
	var t Totals

	if err := d.sql.QueryRow(countAll).Scan(&t.Tasks, &t.Events, &t.Messages); err != nil {
		return Totals{}, fmt.Errorf("count what %q holds: %w", d.path, err)
	}

	return t, nil
}
