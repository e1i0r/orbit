package db

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/e1i0r/orbit/internal/record"
)

// The supervisor thread: the one conversation in Orbit that belongs to no
// task. It had a file of its own beside the tasks and it has a table of its
// own here, for the same reason — it is global, and hanging it off a task
// would mean inventing a task for it.
//
// A turn is taken back rather than erased. supervisor.retracted is appended
// like any other turn, naming the one it withdraws by record.Stamp, and it
// stamps retracted_at on that row in the same transaction. The withdrawn
// line stays exactly where it is: what changes is whether it is put in front
// of the model again, never whether it was said.

// AppendMessage writes one turn of the supervisor thread.
func (d *DB) AppendMessage(e record.Event) error {
	e = stamped(e)

	if err := tooBig(e); err != nil {
		return fmt.Errorf("append %q to the supervisor thread: %w", e.Kind, err)
	}

	if err := keepTrying(func() error { return d.messageOnce(e) }); err != nil {
		return fmt.Errorf("append %q to the supervisor thread: %w", e.Kind, err)
	}

	return nil
}

// messageOnce is one attempt: the turn, and the retraction it carries.
func (d *DB) messageOnce(e record.Event) error {
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}

	if err := writeMessage(tx, e); err != nil {
		return errors.Join(err, tx.Rollback())
	}

	return tx.Commit()
}

// writeMessage is the row and, when this turn takes one back, the stamp on
// the row it withdraws — both inside the caller's transaction, so a
// retraction that could not land leaves no turn claiming to have retracted.
func writeMessage(tx *sql.Tx, e record.Event) error {
	data, err := encode(e.Data)
	if err != nil {
		return err
	}

	// source, who, task_id and repo_id are the parts of the data a reader
	// filters on, lifted into columns so that "everything the supervisor did
	// about ACME-1" is a query. The map is still stored whole: the columns
	// are a way in, not a second copy to keep in step.
	if _, err := tx.Exec(
		insertMessage,
		record.Stamp(e.At), e.Kind, e.Data["channel"], e.Data["by"],
		e.Data["task_id"], e.Data["repo"], e.Text, data,
	); err != nil {
		return fmt.Errorf("insert the message row: %w", err)
	}

	if e.Kind != record.SupervisorRetracted {
		return nil
	}

	taken := e.Data["at"]
	if taken == "" {
		return nil
	}

	if _, err := tx.Exec(retractMessage, record.Stamp(e.At), taken); err != nil {
		return fmt.Errorf("retract the turn of %q: %w", taken, err)
	}

	return nil
}

// Messages is the whole supervisor thread, oldest first.
//
// Retractions come back as the turns they are. A reader folds them the same
// way it folded them out of the file — record.Retracted — and gets the same
// answer; retracted_at is a way to ask the question in SQL, not a second
// answer to it.
func (d *DB) Messages() ([]record.Event, error) {
	rows, err := d.sql.Query(selectMessages)
	if err != nil {
		return nil, fmt.Errorf("read the supervisor thread: %w", err)
	}

	defer rows.Close()

	var thread []record.Event

	for rows.Next() {
		var (
			e        record.Event
			at, data string
		)

		if err := rows.Scan(&e.Kind, &at, &e.Text, &data); err != nil {
			return nil, fmt.Errorf("read a turn of the supervisor thread: %w", err)
		}

		if e.At, err = moment(at); err != nil {
			return nil, fmt.Errorf("a %q turn: %w", e.Kind, err)
		}

		if e.Data, err = decode(data); err != nil {
			return nil, fmt.Errorf("a %q turn: %w", e.Kind, err)
		}

		thread = append(thread, e)
	}

	return thread, rows.Err()
}

// MessageCount is how many turns the thread already holds.
//
// It is what a migration reads to know where it got to. The thread is
// appended to and never reordered, so a count is a position in it.
func (d *DB) MessageCount() (int, error) {
	var n int
	if err := d.sql.QueryRow(countMessages).Scan(&n); err != nil {
		return 0, fmt.Errorf("count the supervisor thread: %w", err)
	}

	return n, nil
}

// EventCount is how many events of one task the record already holds, read
// for the same reason and with the same property behind it.
func (d *DB) EventCount(taskID string) (int, error) {
	var n int
	if err := d.sql.QueryRow(countEvents, taskID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count the events of %q: %w", taskID, err)
	}

	return n, nil
}
