package db

import (
	"fmt"

	"github.com/e1i0r/orbit/internal/record"
)

// Following the record rather than asking it a question. A board watching
// twenty tasks wants what has been written since it last looked, across all
// of them, and that is one query — where asking each task in turn is twenty.

// Change is one event and the task it was written about.
//
// N is the row it was written into, and it is what a reader carries to the
// next call. Not the time it happened: two processes writing in the same
// millisecond have no order between their clocks, and a clock that stepped
// backwards would reorder history that had not changed. The row id is the
// order the record was written in, and it is the only one there is.
type Change struct {
	N     int64
	Task  string
	Event record.Event
}

// Since is every event written after a row, oldest first.
//
// task is the one it is about, or the empty string for all of them at once.
// A board following twenty tasks asks the second way and pays one query per
// refresh; the first way is for a task the reader has only just heard of,
// which has a history behind the row every other task has reached.
//
// A reader starts at zero and gets everything, which is the same cost as
// reading every log from the top and is paid once. After that it passes back
// the N of the last Change it saw and is given only what has arrived, which
// on most calls is nothing at all.
func (d *DB) Since(n int64, task string) ([]Change, error) {
	query, args := selectSince, []any{n}
	if task != "" {
		query, args = selectSinceOf, []any{n, task}
	}

	rows, err := d.sql.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("read what the record holds after %d: %w", n, err)
	}

	defer rows.Close()

	var changes []Change

	for rows.Next() {
		var (
			c                           Change
			kind, at, phase, text, data string
		)

		if err := rows.Scan(&c.N, &c.Task, &kind, &at, &phase, &text, &data); err != nil {
			return nil, fmt.Errorf("read an event written after %d: %w", n, err)
		}

		c.Event = eventOf(c.N, kind, at, phase, text, data)
		changes = append(changes, c)
	}

	return changes, rows.Err()
}
