package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// Reading back what was written. The statements are in queries.go; what is
// here is turning their rows into the types a caller folds.

// Run is one attempt at a task, and the phases it went through.
type Run struct {
	N       int
	Started time.Time
	// Ended is the zero time while the attempt is still running.
	Ended   time.Time
	Outcome string
	Phases  []Phase
}

// Phase is one step of one attempt.
type Phase struct {
	N       int
	Name    string
	Engine  string
	Model   string
	Started time.Time
	Ended   time.Time
	Outcome string
}

// Events is everything recorded about one task, oldest first.
//
// A task nobody has heard of reads as no events rather than as a failure,
// which is the answer the file behind this gave and the one every caller
// already handles: a task written down and never run has nothing to show.
func (d *DB) Events(taskID string) ([]record.Event, error) {
	rows, err := d.sql.Query(selectEvents, taskID)
	if err != nil {
		return nil, fmt.Errorf("read the events of %q: %w", taskID, err)
	}

	defer rows.Close()

	var events []record.Event

	for rows.Next() {
		var (
			n                           int64
			kind, at, phase, text, data string
		)

		if err := rows.Scan(&n, &kind, &at, &phase, &text, &data); err != nil {
			return nil, fmt.Errorf("read an event of %q: %w", taskID, err)
		}

		events = append(events, eventOf(n, kind, at, phase, text, data))
	}

	return events, rows.Err()
}

// Tasks is every task id the record knows, in the order the tasks were first
// written down.
func (d *DB) Tasks() ([]string, error) {
	rows, err := d.sql.Query(selectTasks)
	if err != nil {
		return nil, fmt.Errorf("read the tasks: %w", err)
	}

	defer rows.Close()

	var ids []string

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read a task: %w", err)
		}

		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// Runs is every attempt at a task, oldest first, each carrying its phases.
//
// One query and not one per run: a task retried through the day has as many
// runs as somebody had patience, and a query per row is how a board drawing
// thirty tasks makes a hundred round trips. One query is also one snapshot —
// a retry starting while this reads cannot land half in the answer.
//
// The join is a left one because a run whose first phase has not started yet
// is a run, and an inner join would leave it out of the board at exactly the
// moment somebody is watching for it.
func (d *DB) Runs(taskID string) ([]Run, error) {
	rows, err := d.sql.Query(selectRuns, taskID)
	if err != nil {
		return nil, fmt.Errorf("read the runs of %q: %w", taskID, err)
	}

	defer rows.Close()

	var runs []Run

	for rows.Next() {
		r, p, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("read a run of %q: %w", taskID, err)
		}

		// The run repeats down the rows, once per phase of it.
		if len(runs) == 0 || runs[len(runs)-1].N != r.N {
			runs = append(runs, r)
		}

		if p != nil {
			last := &runs[len(runs)-1]
			last.Phases = append(last.Phases, *p)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the runs of %q: %w", taskID, err)
	}

	return runs, nil
}

// scanRun reads one row of the join: the attempt, and the phase of it this
// row is about, which is nothing at all for an attempt with no phases yet.
func scanRun(rows *sql.Rows) (Run, *Phase, error) {
	var (
		r        Run
		p        Phase
		started  string
		ended    sql.NullString
		outcome  sql.NullString
		pn       sql.NullInt64
		pName    sql.NullString
		pEngine  sql.NullString
		pModel   sql.NullString
		pStarted sql.NullString
		pEnded   sql.NullString
		pOutcome sql.NullString
	)

	if err := rows.Scan(
		&r.N, &started, &ended, &outcome,
		&pn, &pName, &pEngine, &pModel, &pStarted, &pEnded, &pOutcome,
	); err != nil {
		return r, nil, err
	}

	var err error
	if r.Started, r.Ended, err = span(started, ended); err != nil {
		return r, nil, fmt.Errorf("run %d: %w", r.N, err)
	}

	r.Outcome = outcome.String

	if !pn.Valid {
		return r, nil, nil
	}

	p.N = int(pn.Int64)
	p.Name, p.Engine, p.Model, p.Outcome = pName.String, pEngine.String, pModel.String, pOutcome.String

	if p.Started, p.Ended, err = span(pStarted.String, pEnded); err != nil {
		return r, nil, fmt.Errorf("phase %d of run %d: %w", p.N, r.N, err)
	}

	return r, &p, nil
}

// span is when something began and when it ended, with a NULL end reading as
// the zero time: still running.
func span(started string, ended sql.NullString) (time.Time, time.Time, error) {
	from, err := moment(started)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	if !ended.Valid {
		return from, time.Time{}, nil
	}

	to, err := moment(ended.String)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return from, to, nil
}

// eventOf turns one row into the event it holds, and a row that will not
// read into the record.unreadable event that says so.
//
// It cannot fail, and that is deliberate. A reader answering with fewer
// events than the record holds and nothing to say so is lying about its own
// completeness, and a board drawing twenty tasks would go blank because one
// row of one of them had been hand-edited. internal/record does exactly this
// for a line of a file that will not parse; this is the same fact about a
// row.
func eventOf(n int64, kind, at, phase, text, data string) record.Event {
	when, err := moment(at)
	if err != nil {
		return unreadable(n, "the time it was written at is not a time")
	}

	held, err := decode(data)
	if err != nil {
		return unreadable(n, "what it carries is not the object it should be")
	}

	return record.Event{Kind: kind, At: when, Phase: phase, Text: text, Data: held}
}

// unreadable names the row rather than counting rows, for the reason
// internal/record names a byte rather than a line: a count is only true of
// the reader that made it, and a row id is the same fact for everybody.
// Whoever is looking can go and read it with their own eyes.
func unreadable(n int64, why string) record.Event {
	return record.Event{
		Kind: record.Unreadable,
		Text: "this row of the record could not be read: " + why,
		Data: map[string]string{"row": strconv.FormatInt(n, 10)},
	}
}

// moment reads a column back into the time it was written from.
func moment(s string) (time.Time, error) {
	at, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("read %q as a time: %w", s, err)
	}

	// Stored in UTC, answered where the reader is. The instant is the same
	// either way; what changes is what `orbit show` prints, and a reader
	// asking what happened at four o'clock means four o'clock on the clock in
	// front of them. The log kept the writer's own offset in the line and
	// this brings the same answer back out of a column that cannot.
	return at.Local(), nil
}

// decode is encode backwards: an empty column is an event with no data,
// which is most of them.
func decode(data string) (map[string]string, error) {
	if data == "" {
		return nil, nil
	}

	var m map[string]string
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return nil, fmt.Errorf("read the event data: %w", err)
	}

	return m, nil
}
