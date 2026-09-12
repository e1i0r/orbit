package db

// Coming up to the shape this binary knows, from whatever shape is there.

import (
	"path/filepath"
	"testing"
	"time"
)

// TestAVersionIsOnlyAPromiseInsideOneLineOfWork.
//
// A record was found stamped 2 with no proposal table in it: that 2 was
// written by a branch exploring something else, and it meant a different
// record than this 2 does. A step that trusted the number altered a table
// that was not there, and every command that opened the record said so.
//
// So what is there is read. This builds the shape that broke it — stamped 2,
// with no proposal table — and asks for it whole.
func TestAVersionIsOnlyAPromiseInsideOneLineOfWork(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orbit.db")

	d, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if _, err := d.sql.Exec(`DROP TABLE proposal`); err != nil {
		t.Fatalf("take the proposal table away: %v", err)
	}

	if _, err := d.sql.Exec(`PRAGMA user_version = 2`); err != nil {
		t.Fatalf("stamp the version the other branch wrote: %v", err)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	again, err := Open(path)
	if err != nil {
		t.Fatalf("a record at that 2 was refused: %v", err)
	}

	defer func() { _ = again.Close() }() //nolint:errcheck // the test is over

	if err := again.Propose(Proposal{SaidAt: time.Now().UTC(), Said: "always wrap errors"}); err != nil {
		t.Errorf("the table was not put back: %v", err)
	}
}

// TestTheColumnsAreNotAddedTwice, which is the other half: SQLite has no IF
// NOT EXISTS for a column, so a record already at this shape must be left
// alone rather than altered again.
func TestTheColumnsAreNotAddedTwice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orbit.db")

	for range 2 {
		d, err := Open(path)
		if err != nil {
			t.Fatalf("open: %v", err)
		}

		if err := d.Close(); err != nil {
			t.Fatalf("close: %v", err)
		}
	}
}
