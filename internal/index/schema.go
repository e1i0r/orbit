package index

// The shape of the index, and what happens when it changes.

import (
	"fmt"
)

// version is the schema this build folds into. Raise it whenever the tables
// below change in any way at all: there is nothing to preserve, so there is
// nothing to think carefully about. An index at another version is deleted
// and folded again from the logs.
const version = 1

// schema is every table, and each one is a projection.
//
// Two things are deliberately absent. There is no state or band column on a
// task: which band a task sits in is folded out of its events by
// internal/view, and a stored answer to a question that already has one is
// two answers that disagree the first time a process is killed. And there
// are no settings and no flows: those are configuration a person edits by
// hand, and hand-edited configuration in a database is configuration that
// can no longer be edited by hand.
//
// folded_bytes is what makes the projector resumable and folding idempotent.
// It is how much of a task's log has already been folded, so a second pass
// over a log that has not grown folds nothing, and a pass over one that has
// folds only the tail.
const schema = `
CREATE TABLE repos (
	id        INTEGER PRIMARY KEY,
	abs_path  TEXT NOT NULL UNIQUE,
	first_at  TEXT NOT NULL
);

CREATE TABLE tasks (
	id           INTEGER PRIMARY KEY,
	task_id      TEXT NOT NULL UNIQUE,
	text         TEXT NOT NULL DEFAULT '',
	flow         TEXT NOT NULL DEFAULT '',
	created_at   TEXT NOT NULL DEFAULT '',
	folded_bytes INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE task_repos (
	task_id   INTEGER NOT NULL REFERENCES tasks(id),
	repo_id   INTEGER NOT NULL REFERENCES repos(id),
	joined_at TEXT NOT NULL,
	PRIMARY KEY (task_id, repo_id)
);

CREATE TABLE events (
	id      INTEGER PRIMARY KEY,
	task_id INTEGER NOT NULL REFERENCES tasks(id),
	at      TEXT NOT NULL,
	kind    TEXT NOT NULL,
	phase   TEXT NOT NULL DEFAULT '',
	text    TEXT NOT NULL DEFAULT '',
	data    TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX events_by_task ON events (task_id, id);
CREATE INDEX events_by_kind ON events (kind, at);
`

// migrate brings an open database to the current schema, emptying it when
// what it finds was folded by another version.
func (x *Index) migrate() error {
	var found int
	if err := x.db.QueryRow(`PRAGMA user_version`).Scan(&found); err != nil {
		return fmt.Errorf("read the schema version of %q: %w", x.path, err)
	}

	if found == version {
		return nil
	}

	if found != 0 {
		if err := x.empty(); err != nil {
			return err
		}
	}

	if _, err := x.db.Exec(schema); err != nil {
		return fmt.Errorf("create the tables of %q: %w", x.path, err)
	}

	// The version goes on last, so a build interrupted halfway through the
	// schema is found by the next run as a version it does not know and
	// emptied, rather than trusted for tables that are not all there.
	if _, err := x.db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, version)); err != nil {
		return fmt.Errorf("stamp the schema version of %q: %w", x.path, err)
	}

	return nil
}

// empty drops what an older version left. Dropping the tables rather than
// unlinking the file keeps every reader's open handle valid: the window may
// be holding one while the command that upgraded is running.
func (x *Index) empty() error {
	for _, table := range []string{"events", "task_repos", "tasks", "repos"} {
		if _, err := x.db.Exec(`DROP TABLE IF EXISTS ` + table); err != nil {
			return fmt.Errorf("drop %q of %q: %w", table, x.path, err)
		}
	}

	return nil
}
