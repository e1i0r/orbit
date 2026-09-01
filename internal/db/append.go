package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// retries is how many times a write refused for want of a turn asks again
// before the caller is told it failed.
//
// Four, backing off, covers the only case that produces a refusal at all: a
// process holding the write lock across work it should not be. Measured with
// one process holding for ten seconds, four attempts lost nothing and the
// same run without them lost every event nine other writers tried.
const retries = 4

// Append writes one event down.
//
// The whole of the work is one transaction: the event row, and the run or
// phase row that this event opens or closes. That is deliberate and it is
// the second half of what makes run and phase safe to store — a projector
// that walked the events afterwards would leave a window in which the event
// exists and the row derived from it does not, and that window is where two
// answers to one question come from.
func (d *DB) Append(taskID string, e record.Event) error {
	if e.At.IsZero() {
		e.At = time.Now()
	}

	var err error

	for try := 0; ; try++ {
		err = d.appendOnce(taskID, e)
		if err == nil || !refused(err) || try == retries {
			break
		}

		// Backing off rather than spinning: the lock is held by somebody
		// doing something, and asking again immediately only adds a wakeup
		// to whatever they are doing.
		time.Sleep(time.Duration(try+1) * 250 * time.Millisecond)
	}

	if err != nil {
		return fmt.Errorf("append %q to %q: %w", e.Kind, taskID, err)
	}

	return nil
}

// refused reports whether a write failed for want of a turn at the lock,
// which is the one failure worth asking about again. Anything else — a
// constraint, a broken file, a column that is not there — will fail the same
// way the second time.
func refused(err error) bool {
	s := err.Error()

	return strings.Contains(s, "SQLITE_BUSY") || strings.Contains(s, "database is locked")
}

// appendOnce is one attempt: open, write, commit.
func (d *DB) appendOnce(taskID string, e record.Event) error {
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}

	if err := writeEvent(tx, taskID, e); err != nil {
		// Without this the connection stays checked out and the next Begin
		// waits on it for as long as the process lives, which reads as a
		// database that has stopped answering and is not one.
		return errors.Join(err, tx.Rollback())
	}

	return tx.Commit()
}

// writeEvent is everything one event changes, and it runs inside the
// caller's transaction so that all of it lands or none of it does.
func writeEvent(tx *sql.Tx, taskID string, e record.Event) error {
	task, err := taskRow(tx, taskID, e)
	if err != nil {
		return err
	}

	run, phase, err := spanOf(tx, task, e)
	if err != nil {
		return err
	}

	data, err := encode(e.Data)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		insertEvent,
		task, null(run), null(phase), e.Kind, record.Stamp(e.At), e.Phase, e.Text, data,
	)
	if err != nil {
		return fmt.Errorf("insert the event row: %w", err)
	}

	return joined(tx, task, e)
}

// taskRow is the task's row id, made on the way past if this is the first
// anybody has heard of it.
//
// A task appearing because an event mentioned it is not a shortcut. The
// record is what happened, and an event about a task nobody wrote down first
// is still something that happened; refusing it here would lose the event to
// protect a foreign key.
func taskRow(tx *sql.Tx, taskID string, e record.Event) (int64, error) {
	at := record.Stamp(e.At)

	if _, err := tx.Exec(insertTask, taskID, at); err != nil {
		return 0, fmt.Errorf("insert the task row for %q: %w", taskID, err)
	}

	// task.created is the one event that says what a task is, and it can
	// arrive after the row exists — a log migrated out of order, or an event
	// that mentioned the task first.
	if e.Kind == record.TaskCreated {
		if _, err := tx.Exec(fillTask, e.Text, e.Data["flow"], at, taskID); err != nil {
			return 0, fmt.Errorf("fill in the task row for %q: %w", taskID, err)
		}
	}

	var id int64
	if err := tx.QueryRow(findTask, taskID).Scan(&id); err != nil {
		return 0, fmt.Errorf("find the task row for %q: %w", taskID, err)
	}

	return id, nil
}

// joined records the repository an event says the task is being worked in.
//
// The scope of a task is observed rather than declared: opening a worktree is
// what joining is, so repo.joined is the event that makes the row.
func joined(tx *sql.Tx, task int64, e record.Event) error {
	if e.Kind != record.RepoJoined {
		return nil
	}

	abs := e.Data["path"]
	if abs == "" {
		return nil
	}

	at := record.Stamp(e.At)

	if _, err := tx.Exec(insertRepo, abs, e.Data["repo"], at); err != nil {
		return fmt.Errorf("insert the repository row for %q: %w", abs, err)
	}

	if _, err := tx.Exec(joinTaskToRepo, task, at, abs); err != nil {
		return fmt.Errorf("join %q to the task: %w", abs, err)
	}

	return nil
}

// encode turns an event's data into the column, and an empty map into an
// empty string rather than the four characters "null".
func encode(data map[string]string) (string, error) {
	if len(data) == 0 {
		return "", nil
	}

	b, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("encode the event data: %w", err)
	}

	return string(b), nil
}

// null turns the zero row id into SQL NULL, because a foreign key of 0 names
// a row that is not there and a NULL says there is none.
func null(id int64) any {
	if id == 0 {
		return nil
	}

	return id
}
