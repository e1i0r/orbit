package db

// Every write, against a record an older orbit must not touch.
//
// Open reopens a record ahead of this binary read-only, so the file itself
// refuses a write that gets that far — but what the file says is "attempt to
// write a readonly database", which names neither the record nor the way
// out. AheadError names both, and these tests are what keep every door
// answering with it rather than with SQLite's sentence.
//
// Five writes did not ask. They are the five below that carry no event:
// pull requests, proposals and rule turns, which arrived after the guard was
// written and never met it.

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// ahead is a record stamped with a schema version no binary knows, reopened.
// What comes back is the read-only handle Open answers with.
func ahead(t *testing.T) *DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "orbit.db")

	first := openAt(t, path)
	if _, err := first.sql.Exec(`PRAGMA user_version = 99`); err != nil {
		t.Fatalf("stamp a later version: %v", err)
	}

	if err := first.Close(); err != nil {
		t.Fatalf("close the record before reopening it: %v", err)
	}

	d, err := Open(path)
	if err != nil {
		t.Fatalf("reopen a record at version 99: %v", err)
	}

	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Errorf("close %s: %v", path, err)
		}
	})

	if !d.Ahead() {
		t.Fatal("a record at version 99 does not read as ahead")
	}

	return d
}

// TestEveryWriteIsRefusedByNameOverARecordAhead. By name is the whole of it:
// the refusal has to be an AheadError, because that is the one that says
// which file, which version, and that the way out is to upgrade orbit.
func TestEveryWriteIsRefusedByNameOverARecordAhead(t *testing.T) {
	said := time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)

	writes := []struct {
		name  string
		write func(*DB) error
	}{
		{"Append", func(d *DB) error {
			return d.Append("ACME-1", record.Event{Kind: record.TaskCreated, Text: "Pay the thing"})
		}},
		{"AppendMessage", func(d *DB) error {
			return d.AppendMessage(record.Event{Kind: record.SupervisorMessage, Text: "carry on"})
		}},
		{"Join", func(d *DB) error {
			return d.Join("ACME-1", "/src/acme", "acme", said)
		}},
		{"Unjoin", func(d *DB) error {
			_, err := d.Unjoin("/src/acme")

			return err
		}},
		{"OpenedPR", func(d *DB) error {
			return d.OpenedPR("ACME-1", "/src/acme", "https://github.test/acme/pull/1")
		}},
		{"MarkPR", func(d *DB) error {
			return d.MarkPR("ACME-1", "/src/acme", PRMerged)
		}},
		{"Propose", func(d *DB) error {
			return d.Propose(Proposal{SaidAt: said, Said: "run the tests first", By: "operator"})
		}},
		{"Decide", func(d *DB) error {
			return d.Decide(said, Kept)
		}},
		{"Happened", func(d *DB) error {
			return d.Happened(RuleTurn{Rule: "R-1", At: said, What: "kept", By: "operator"})
		}},
	}

	for _, w := range writes {
		t.Run(w.name, func(t *testing.T) {
			d := ahead(t)

			err := w.write(d)
			if err == nil {
				t.Fatalf("%s wrote into a record ahead of this binary", w.name)
			}

			var was AheadError
			if !errors.As(err, &was) {
				t.Fatalf("%s was refused with %q, want the refusal that names the upgrade", w.name, err)
			}

			if was.Found != 99 || was.Known != version {
				t.Errorf("%s was refused with found %d known %d, want 99 and %d",
					w.name, was.Found, was.Known, version)
			}
		})
	}
}

// TestARuleWithNoNameIsStillSilentOverARecordAhead. Happened says nothing
// about a rule nobody named, and that answer comes before the guard: a
// silence is not a write, so there is nothing for the record to refuse.
func TestARuleWithNoNameIsStillSilentOverARecordAhead(t *testing.T) {
	d := ahead(t)

	if err := d.Happened(RuleTurn{What: "kept", By: "operator"}); err != nil {
		t.Errorf("a rule with no name was refused over a record ahead: %v", err)
	}
}
