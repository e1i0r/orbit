package db

// Asking SQLite whether the file is still the file, and asking the file how
// much of ours is in it.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// corrupt scribbles over the head of every page but the first.
//
// Page one holds the schema, which is why it is left alone: a file whose
// schema is gone will not open at all, and what is under test here is the
// record that opens, answers questions, and is broken anyway — the shape of
// damage a reader has no other way of finding out about. The database must
// be closed before this runs and reopened after it, because a handle holds
// pages in memory that this does not touch.
func corrupt(t *testing.T, path string) {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY, fileMode)
	if err != nil {
		t.Fatalf("open %s to break it: %v", path, err)
	}

	const page = 4096

	for at := page; at < len(body); at += page {
		if _, err := f.WriteAt([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, int64(at)); err != nil {
			t.Fatalf("scribble at %d: %v", at, err)
		}
	}

	if err := f.Close(); err != nil {
		t.Fatalf("close %s: %v", path, err)
	}
}

// fill puts enough events in a record to spread it over several pages, so
// that breaking one of them breaks something that was being kept.
func fill(t *testing.T, d *DB) {
	t.Helper()

	tick := clock()

	for i := range 40 {
		e := record.Event{
			At:   tick(),
			Kind: record.PhaseFinished,
			Text: strings.Repeat("the engine said something at length. ", 4),
		}

		if err := d.Append("T-1", e); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}
}

// TestASoundRecordHasNothingToSayAboutItself. The answer to a healthy file
// is an empty list rather than the word SQLite actually says: "ok" is a row
// this package reads and does not pass on, because a caller printing
// whatever came back would print it.
func TestASoundRecordHasNothingToSayAboutItself(t *testing.T) {
	d := open(t)
	fill(t, d)

	found, err := d.Check()
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	if len(found) != 0 {
		t.Fatalf("a record nothing has happened to answered %q", found)
	}
}

// TestDamageComesBackAsAnAnswerAndNotAsAFailure. SQLite reports a file torn
// this badly by refusing to finish the pragma, so the news arrives as an
// error where the rest of it arrives as rows. A caller that returned that
// error would print `check "/path/orbit.db": database disk image is
// malformed (11)` — the SQLite dump this command exists not to print.
func TestDamageComesBackAsAnAnswerAndNotAsAFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orbit.db")

	d, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	fill(t, d)

	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	corrupt(t, path)

	broken := openAt(t, path)

	found, err := broken.Check()
	if err != nil {
		t.Fatalf("check a broken record: %v", err)
	}

	if len(found) != 1 || !strings.Contains(found[0], "malformed") {
		t.Fatalf("a broken record answered %q, want SQLite's sentence about the image", found)
	}
}

// TestWhatWasFoundComesBackLineByLine. Damage small enough for the pragma
// to finish is reported as rows, and every one of them is a sentence the
// command prints — SQLite names the table and the column, which is the only
// part of this a reader can act on.
//
// The damage here is a column declared NOT NULL over rows that are null: an
// event that belongs to no run, under a schema that says every event belongs
// to one. It is what a half-applied migration leaves behind, and it is
// invisible to every query — the count below still answers — which is the
// whole reason a record has to be asked rather than watched.
func TestWhatWasFoundComesBackLineByLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orbit.db")

	d, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	tick := clock()
	if err := d.Append("T-1", record.Event{At: tick(), Kind: record.TaskCreated, Text: "no run has begun"}); err != nil {
		t.Fatalf("append: %v", err)
	}

	for _, stmt := range []string{
		`PRAGMA writable_schema=ON`,
		`UPDATE sqlite_schema SET sql = replace(sql, 'run_id   INTEGER REFERENCES', 'run_id   INTEGER NOT NULL REFERENCES')
		  WHERE type = 'table' AND name = 'event'`,
		`PRAGMA writable_schema=RESET`,
	} {
		if _, err := d.sql.Exec(stmt); err != nil {
			t.Fatalf("rewrite the schema: %v", err)
		}
	}

	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	unsound := openAt(t, path)

	found, err := unsound.Check()
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	if len(found) != 1 || !strings.Contains(found[0], "event.run_id") {
		t.Fatalf("the check answered %q, want the column SQLite objected to", found)
	}

	if n, err := unsound.EventCount("T-1"); err != nil || n != 1 {
		t.Errorf("the record answered %d events and %v, so the damage was not the invisible kind", n, err)
	}
}

// TestNotHavingBeenAbleToLookIsStillAFailure. The line above turns one kind
// of error into an answer, and it has to turn only that kind: a handle that
// is closed is a check that did not happen, and a command told the record
// was fine because nobody could ask is worse than one told nothing.
func TestNotHavingBeenAbleToLookIsStillAFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orbit.db")

	d, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	found, err := d.Check()
	if err == nil {
		t.Fatalf("a check through a closed handle answered %q and no error", found)
	}

	if !strings.Contains(err.Error(), path) {
		t.Errorf("the failure is %q, want the path of what could not be checked in it", err)
	}
}

// TestTotalsCountEveryThingSeparately. Three numbers from one statement:
// the tasks, the events under them, and the supervisor thread, which belongs
// to no task and is counted apart from them for that reason.
func TestTotalsCountEveryThingSeparately(t *testing.T) {
	d := open(t)
	tick := clock()

	for _, id := range []string{"T-1", "T-2"} {
		if err := d.Append(id, record.Event{At: tick(), Kind: record.TaskCreated, Text: id}); err != nil {
			t.Fatalf("append to %s: %v", id, err)
		}
	}

	if err := d.Append("T-1", record.Event{At: tick(), Kind: record.PhaseStarted, Phase: "plan"}); err != nil {
		t.Fatalf("append a second event: %v", err)
	}

	say(t, d, record.Event{At: tick(), Kind: record.SupervisorMessage, Text: "retry T-1"})

	held, err := d.Totals()
	if err != nil {
		t.Fatalf("totals: %v", err)
	}

	if held.Tasks != 2 || held.Events != 3 || held.Messages != 1 {
		t.Errorf("the record holds %+v, want 2 tasks, 3 events and 1 turn", held)
	}
}

// TestTotalsOfARecordThatCannotBeRead. The count is a query like any other,
// so it fails like one — and a command that asked how much is in the file
// has to be told it could not be counted rather than shown a zero.
func TestTotalsOfARecordThatCannotBeRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orbit.db")

	d, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	held, err := d.Totals()
	if err == nil {
		t.Fatalf("counting through a closed handle answered %+v", held)
	}

	if held != (Totals{}) {
		t.Errorf("a failed count answered %+v, want no numbers at all", held)
	}
}
