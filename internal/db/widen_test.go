package db

// Coming up to this shape from the shapes that came before it.
//
// Every step so far has been a column added to a table somebody already had
// rows in, and the whole of what a migration promises is that those rows are
// still there afterwards. So each test here takes a column away, puts a row
// in the narrow table, and asks for the row back through the door that reads
// it — rather than asking the schema whether it looks right, which is the
// question a migration can pass while losing everything.

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// narrowed is a record taken back to an older shape: the columns dropped,
// the version stamped back, and the handle closed so the next Open is the
// migration under test.
//
// It is dropped rather than built from an old CREATE TABLE kept in a test,
// because a copy of the old schema here is a second place the shape lives
// and it goes stale the first time nobody remembers to change it.
func narrowed(t *testing.T, path string, back int, drop ...string) {
	t.Helper()

	d := openAt(t, path)

	for _, statement := range drop {
		if _, err := d.sql.Exec(statement); err != nil {
			t.Fatalf("take the record back with %q: %v", statement, err)
		}
	}

	if _, err := d.sql.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, back)); err != nil {
		t.Fatalf("stamp the older version: %v", err)
	}
}

// rowsBefore puts one row in each narrow table, through SQL rather than
// through a door: the doors of this package write the columns that are not
// there yet.
func rowsBefore(t *testing.T, d *DB) {
	t.Helper()

	writes := []struct {
		what string
		sql  string
		args []any
	}{
		{"a task", `INSERT INTO task(task_id, created_at) VALUES('ACME-1', ?)`, nil},
		{
			"a repository",
			`INSERT INTO repo(abs_path, name, first_seen)
			 VALUES('/src/acme', 'acme', ?)`,
			nil,
		},
		{
			"a pull request",
			`INSERT INTO pr(task_id, repo_id, url, opened_at)
			 SELECT (SELECT id FROM task WHERE task_id = 'ACME-1'),
			        (SELECT id FROM repo WHERE abs_path = '/src/acme'),
			        'https://github.test/acme/pull/1', ?`,
			nil,
		},
		{
			"a proposal",
			`INSERT INTO proposal(said_at, said)
			 VALUES(?, 'run the tests before you push')`,
			nil,
		},
		{"a rule turn", `INSERT INTO rule(rule_id, at, what) VALUES('R-1', ?, 'kept')`, nil},
	}

	at := time.Date(2026, 9, 17, 9, 0, 1, 0, time.UTC).Format(time.RFC3339Nano)

	for _, w := range writes {
		if _, err := d.sql.Exec(w.sql, append(w.args, at)...); err != nil {
			t.Fatalf("write %s into the narrow record: %v", w.what, err)
		}
	}
}

// TestAnOlderRecordKeepsItsRowsThroughEveryWidening. One case per group of
// columns, because each arrived at a different time and a record holding any
// prefix of them is a record somebody was using in between.
func TestAnOlderRecordKeepsItsRowsThroughEveryWidening(t *testing.T) {
	cases := []struct {
		name string
		drop []string
	}{
		{"the pull request never said what became of it", []string{
			`ALTER TABLE pr DROP COLUMN state`,
		}},
		{"a proposal only ever came from the supervisor", []string{
			`ALTER TABLE proposal DROP COLUMN said_by`,
			`ALTER TABLE proposal DROP COLUMN about_task`,
			`ALTER TABLE proposal DROP COLUMN repo`,
			`ALTER TABLE proposal DROP COLUMN path`,
			`ALTER TABLE proposal DROP COLUMN topic`,
			`ALTER TABLE proposal DROP COLUMN habit`,
			`ALTER TABLE proposal DROP COLUMN gate`,
		}},
		{"a proposal knew where the work was but not which habit", []string{
			`ALTER TABLE proposal DROP COLUMN habit`,
			`ALTER TABLE proposal DROP COLUMN topic`,
			`ALTER TABLE proposal DROP COLUMN gate`,
		}},
		{"a proposal knew its habit but arrived with no command", []string{
			`ALTER TABLE proposal DROP COLUMN gate`,
		}},
		{"a rule turn never said where it happened", []string{
			`ALTER TABLE rule DROP COLUMN task_id`,
			`ALTER TABLE rule DROP COLUMN phase`,
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "orbit.db")

			before := openAt(t, path)
			rowsBefore(t, before)

			for _, statement := range c.drop {
				if _, err := before.sql.Exec(statement); err != nil {
					t.Fatalf("take the record back with %q: %v", statement, err)
				}
			}

			if _, err := before.sql.Exec(`PRAGMA user_version = 1`); err != nil {
				t.Fatalf("stamp the older version: %v", err)
			}

			if err := before.Close(); err != nil {
				t.Fatalf("close the narrow record: %v", err)
			}

			after := openAt(t, path)

			// The rows are still there, read through the doors that need
			// the new columns to answer at all.
			prs, err := after.PullRequests("ACME-1")
			if err != nil {
				t.Fatalf("read the pull requests: %v", err)
			}

			if len(prs) != 1 || prs[0].State != PROpen {
				t.Errorf("the pull request came through as %+v, want the one row still open", prs)
			}

			waiting, err := after.Waiting()
			if err != nil {
				t.Fatalf("read what is waiting: %v", err)
			}

			if len(waiting) != 1 || waiting[0].Said != "run the tests before you push" {
				t.Errorf("the proposal came through as %+v, want the one sentence", waiting)
			}

			history, err := after.RuleHistory("R-1")
			if err != nil {
				t.Fatalf("read the history of R-1: %v", err)
			}

			if len(history) != 1 || history[0].What != RuleKept {
				t.Errorf("the rule turn came through as %+v, want the one turn", history)
			}

			// And the widened record takes the writes the narrow one could
			// not, which is what the columns were added for.
			if err := after.MarkPR("ACME-1", "/src/acme", PRMerged); err != nil {
				t.Errorf("the widened record refused a mark: %v", err)
			}

			if err := after.Happened(RuleTurn{
				Rule: "R-1", At: time.Now().UTC(), What: RuleSkipped, Task: "ACME-1", Phase: "test",
			}); err != nil {
				t.Errorf("the widened record refused a turn with a phase on it: %v", err)
			}
		})
	}
}

// TestAnAlreadyWideRecordIsLeftAlone. SQLite has no IF NOT EXISTS for a
// column, so every widening asks the table about its own shape first. A
// record stamped back with every column already there is the case where
// asking is the only thing between it and an error on every open.
func TestAnAlreadyWideRecordIsLeftAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orbit.db")

	narrowed(t, path, 1)

	again, err := Open(path)
	if err != nil {
		t.Fatalf("a record stamped back with every column already there was refused: %v", err)
	}

	defer func() { _ = again.Close() }() //nolint:errcheck // the test is over

	var found int
	if err := again.sql.QueryRow(readVersion).Scan(&found); err != nil {
		t.Fatalf("read the version it came up to: %v", err)
	}

	if found != version {
		t.Errorf("it came up to version %d, want %d", found, version)
	}
}

// TestAMigrationThatFailsLeavesNothingBehind. Every step runs in one
// transaction, so a record that could not be brought all the way up is the
// record it was rather than half of two shapes.
func TestAMigrationThatFailsLeavesNothingBehind(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orbit.db")

	d := openAt(t, path)
	rowsBefore(t, d)

	// A rule table with task_id missing and phase already there: the
	// widening asks about task_id, is told no, and then fails adding phase
	// a second time. Half a step, which is what the transaction is for.
	for _, statement := range []string{
		`ALTER TABLE rule DROP COLUMN task_id`,
		`PRAGMA user_version = 1`,
	} {
		if _, err := d.sql.Exec(statement); err != nil {
			t.Fatalf("take the record back with %q: %v", statement, err)
		}
	}

	if err := d.Close(); err != nil {
		t.Fatalf("close the narrow record: %v", err)
	}

	broken, err := Open(path)
	if err == nil {
		_ = broken.Close() //nolint:errcheck // the test is over

		t.Fatal("a migration that could not run answered as if it had")
	}

	// The version on disk is untouched, which is what says the transaction
	// rolled back rather than stopping half way.
	raw, err := sql.Open("sqlite", joinDSN(path))
	if err != nil {
		t.Fatalf("reopen the file to look at it: %v", err)
	}

	defer func() { _ = raw.Close() }() //nolint:errcheck // the test is over

	var found int
	if err := raw.QueryRow(readVersion).Scan(&found); err != nil {
		t.Fatalf("read the version: %v", err)
	}

	if found != 1 {
		t.Errorf("the failed migration left the record at version %d, want the 1 it was", found)
	}
}
