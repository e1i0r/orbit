package db

import (
	"database/sql"
	"errors"
	"fmt"
)

// version is the shape of the record on disk. It goes up by one whenever a
// table changes, and every step up needs a migration beside it in migrate.
//
// It is not the disposable version an index carries. Nothing here can be
// rebuilt from anywhere else, so a schema that turns out wrong is migrated
// forward against live data and never dropped and remade.
const version = 1

// busyTimeoutMS is how long SQLite waits for its turn at the write lock
// before refusing. Five seconds is far past any transaction this package
// opens — every one of them is a single insert — so a refusal at this
// timeout means somebody is holding the lock across work they should not be.
const busyTimeoutMS = 5000

// schema is the record.
//
// Two things are deliberately absent. There is no state column on a task:
// the band a task sits in is folded from its events, and a stored band is a
// second answer to a question that already has one, disagreeing with the
// first the moment a process is killed. And there are no settings and no
// flows: those are files a person edits by hand, and hand-edited
// configuration in a database is configuration nobody can edit by hand.
const schema = `
CREATE TABLE IF NOT EXISTS repo(
  id         INTEGER PRIMARY KEY,
  abs_path   TEXT NOT NULL UNIQUE,
  name       TEXT NOT NULL DEFAULT '',
  key        TEXT NOT NULL DEFAULT '',
  first_seen TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS task(
  id         INTEGER PRIMARY KEY,
  task_id    TEXT NOT NULL UNIQUE,
  text       TEXT NOT NULL DEFAULT '',
  flow       TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS task_repo(
  task_id   INTEGER NOT NULL REFERENCES task(id),
  repo_id   INTEGER NOT NULL REFERENCES repo(id),
  joined_at TEXT NOT NULL,
  worktree  TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (task_id, repo_id)
);

CREATE TABLE IF NOT EXISTS run(
  id         INTEGER PRIMARY KEY,
  task_id    INTEGER NOT NULL REFERENCES task(id),
  n          INTEGER NOT NULL,
  started_at TEXT NOT NULL,
  ended_at   TEXT,
  outcome    TEXT
);

CREATE TABLE IF NOT EXISTS phase(
  id         INTEGER PRIMARY KEY,
  run_id     INTEGER NOT NULL REFERENCES run(id),
  n          INTEGER NOT NULL,
  name       TEXT NOT NULL,
  engine     TEXT NOT NULL DEFAULT '',
  model      TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  ended_at   TEXT,
  outcome    TEXT,
  usd        REAL,
  tokens_in  INTEGER,
  tokens_out INTEGER
);

CREATE TABLE IF NOT EXISTS event(
  id       INTEGER PRIMARY KEY,
  task_id  INTEGER NOT NULL REFERENCES task(id),
  run_id   INTEGER REFERENCES run(id),
  phase_id INTEGER REFERENCES phase(id),
  repo_id  INTEGER REFERENCES repo(id),
  kind     TEXT NOT NULL,
  at       TEXT NOT NULL,
  phase    TEXT NOT NULL DEFAULT '',
  text     TEXT NOT NULL DEFAULT '',
  data     TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS decision(
  id            INTEGER PRIMARY KEY,
  task_id       INTEGER NOT NULL REFERENCES task(id),
  decision_id   TEXT NOT NULL DEFAULT '',
  made_at       TEXT NOT NULL,
  scope         TEXT NOT NULL DEFAULT '',
  text          TEXT NOT NULL DEFAULT '',
  superseded_by INTEGER REFERENCES decision(id)
);

CREATE TABLE IF NOT EXISTS pr(
  id        INTEGER PRIMARY KEY,
  task_id   INTEGER NOT NULL REFERENCES task(id),
  repo_id   INTEGER NOT NULL REFERENCES repo(id),
  url       TEXT NOT NULL,
  opened_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS message(
  id           INTEGER PRIMARY KEY,
  at           TEXT NOT NULL,
  kind         TEXT NOT NULL,
  source       TEXT NOT NULL DEFAULT '',
  who          TEXT NOT NULL DEFAULT '',
  task_id      INTEGER REFERENCES task(id),
  repo_id      INTEGER REFERENCES repo(id),
  text         TEXT NOT NULL DEFAULT '',
  data         TEXT NOT NULL DEFAULT '',
  retracted_at TEXT
);

CREATE INDEX IF NOT EXISTS event_by_task ON event(task_id, id);
CREATE INDEX IF NOT EXISTS event_by_kind ON event(kind);
CREATE INDEX IF NOT EXISTS run_by_task   ON run(task_id, n);
CREATE INDEX IF NOT EXISTS phase_by_run  ON phase(run_id, n);
`

// migrate brings the file up to the version this binary knows.
//
// A record newer than the binary is refused rather than opened: an older
// Orbit writing into a shape it does not know is how a column silently stops
// being filled, and the record is the one thing here that cannot be rebuilt.
func (d *DB) migrate() error {
	var found int
	if err := d.sql.QueryRow(readVersion).Scan(&found); err != nil {
		return fmt.Errorf("read the schema version of %q: %w", d.path, err)
	}

	if found > version {
		return fmt.Errorf("%q is at schema version %d and this orbit knows %d: upgrade orbit rather than letting an older one write into it", d.path, found, version)
	}

	if found == version {
		// There is nothing to do, so nothing is written. Every command opens
		// the record before it does anything, and re-stamping the version it
		// already has would make opening it a write: a transaction and a
		// growing write-ahead log for each `orbit list`, and a state root
		// whose file is read-only refusing to be read at all.
		return nil
	}

	tx, err := d.sql.Begin()
	if err != nil {
		return fmt.Errorf("begin the migration of %q: %w", d.path, err)
	}

	if err := stepsFrom(tx, found); err != nil {
		return errors.Join(err, tx.Rollback())
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit the migration of %q: %w", d.path, err)
	}

	return nil
}

// stepsFrom runs every migration between the version on disk and this one.
//
// Version 1 is the whole schema, so a file at 0 gets it and there is nothing
// else to do yet. The second entry here will be the first real migration, and
// the shape it goes in is: a case that alters what exists and leaves the
// rows alone.
func stepsFrom(tx *sql.Tx, found int) error {
	if found < 1 {
		if _, err := tx.Exec(schema); err != nil {
			return fmt.Errorf("create the schema: %w", err)
		}
	}

	// What fills stampVersion is a constant of this package and never
	// anything a caller can reach.
	if _, err := tx.Exec(fmt.Sprintf(stampVersion, version)); err != nil {
		return fmt.Errorf("stamp the schema version: %w", err)
	}

	return nil
}
