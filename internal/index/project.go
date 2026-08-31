package index

// The one write path: a task's log is read from where the last fold stopped,
// and what it says becomes rows.

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/e1i0r/orbit/internal/record"
)

// Project folds whatever has been appended to one task's log since the last
// time it was folded, and answers how many events that was.
//
// It is safe to call as often as anybody likes, which is what makes it the
// only write path worth having: a log that has not grown produces nothing,
// and a log that has grown produces exactly its tail. The whole fold is one
// transaction with the new offset in it, so a fold that fails partway leaves
// the index at the last event it was sure of and folds the rest next time.
func (x *Index) Project(taskID, logPath string) (int, error) {
	tx, err := x.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("start folding %q: %w", taskID, err)
	}

	defer tx.Rollback() //nolint:errcheck // the commit below is what decides; a rollback after it is a no-op

	id, folded, err := taskRow(tx, taskID)
	if err != nil {
		return 0, err
	}

	events, at, err := record.ReadFrom(logPath, folded)
	if err != nil {
		return 0, fmt.Errorf("read the log of %q: %w", taskID, err)
	}

	if at < folded {
		// The log is shorter than what was folded out of it, so it was
		// replaced rather than appended to and the rows above it describe a
		// log that is gone. ReadFrom has already started over from the top;
		// what this drops is the projection of the log that used to be.
		if err := forget(tx, id); err != nil {
			return 0, err
		}
	}

	for _, e := range events {
		if err := foldEvent(tx, id, e); err != nil {
			return 0, fmt.Errorf("fold %q of %q: %w", e.Kind, taskID, err)
		}
	}

	if _, err := tx.Exec(`UPDATE tasks SET folded_bytes = ? WHERE id = ?`, at, id); err != nil {
		return 0, fmt.Errorf("mark %q folded: %w", taskID, err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("finish folding %q: %w", taskID, err)
	}

	return len(events), nil
}

// taskRow is the row for one task id, made if this is the first sight of it,
// with however much of its log has already been folded.
func taskRow(tx *sql.Tx, taskID string) (id int64, folded int64, err error) {
	err = tx.QueryRow(`SELECT id, folded_bytes FROM tasks WHERE task_id = ?`, taskID).Scan(&id, &folded)
	if err == nil {
		return id, folded, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return 0, 0, fmt.Errorf("read the task %q: %w", taskID, err)
	}

	res, err := tx.Exec(`INSERT INTO tasks (task_id) VALUES (?)`, taskID)
	if err != nil {
		return 0, 0, fmt.Errorf("write the task %q: %w", taskID, err)
	}

	id, err = res.LastInsertId()
	if err != nil {
		return 0, 0, fmt.Errorf("read back the task %q: %w", taskID, err)
	}

	return id, 0, nil
}

// forget takes out everything folded from a log that is no longer there.
func forget(tx *sql.Tx, id int64) error {
	for _, stmt := range []string{
		`DELETE FROM events WHERE task_id = ?`,
		`DELETE FROM task_repos WHERE task_id = ?`,
	} {
		if _, err := tx.Exec(stmt, id); err != nil {
			return fmt.Errorf("take back what was folded: %w", err)
		}
	}

	return nil
}

// foldEvent turns one event into rows: the event itself, always, and
// whatever the kinds that say something about the task's shape add to it.
func foldEvent(tx *sql.Tx, id int64, e record.Event) error {
	data, err := json.Marshal(e.Data)
	if err != nil {
		return fmt.Errorf("write down the data: %w", err)
	}

	at := record.Stamp(e.At)

	if _, err := tx.Exec(
		`INSERT INTO events (task_id, at, kind, phase, text, data) VALUES (?, ?, ?, ?, ?, ?)`,
		id, at, e.Kind, e.Phase, e.Text, string(data),
	); err != nil {
		return err
	}

	switch e.Kind {
	case record.TaskCreated:
		_, err = tx.Exec(`UPDATE tasks SET text = ?, flow = ?, created_at = ? WHERE id = ?`, e.Text, e.Data["flow"], at, id)
	case record.RepoJoined:
		err = joinRepo(tx, id, e.Data["path"], at)
	}

	return err
}

// joinRepo writes down that a task was worked in a repository, once. It is
// the row the whole index exists for: a task and a repository are many to
// many, and that is the one shape a directory tree has no way to hold.
func joinRepo(tx *sql.Tx, id int64, path, at string) error {
	if path == "" {
		// A repository that joined without saying where it is says nothing
		// this table can hold. The event itself is already written down.
		return nil
	}

	if _, err := tx.Exec(`INSERT OR IGNORE INTO repos (abs_path, first_at) VALUES (?, ?)`, path, at); err != nil {
		return err
	}

	var repoID int64
	if err := tx.QueryRow(`SELECT id FROM repos WHERE abs_path = ?`, path).Scan(&repoID); err != nil {
		return err
	}

	_, err := tx.Exec(`INSERT OR IGNORE INTO task_repos (task_id, repo_id, joined_at) VALUES (?, ?, ?)`, id, repoID, at)

	return err
}
