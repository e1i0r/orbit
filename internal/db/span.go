package db

import (
	"database/sql"
	"fmt"

	"github.com/e1i0r/orbit/internal/record"
)

// A run and a phase are the two spans an event can sit inside, and unlike the
// band a task is in, each of them begins and ends on exactly one event. That
// is what makes them safe to keep as rows: there is no moment where the
// record says a phase started and the table does not, because the row and
// the event are written by the same transaction.
//
// The band stays out for the opposite reason. It changes for reasons no
// single event announces — a process that died, a deadline that passed — so
// a column holding it would be a guess that ages.

// opensRun and closesRun are the events a run begins and ends on.
//
// task.started is the boundary between one attempt and the next, which is
// why a second one does not reopen the first.
var (
	closesRun = map[string]bool{
		record.TaskFinished:      true,
		record.TaskFailed:        true,
		record.TaskCancelled:     true,
		record.TaskTimedOut:      true,
		record.TaskAbandoned:     true,
		record.TaskStuck:         true,
		record.TaskOverBudget:    true,
		record.TaskOverDiff:      true,
		record.TaskNewDependency: true,
		record.TaskContradicts:   true,
	}

	// closesPhase includes phase.retried because an attempt that a gate
	// refused is a phase that ended: the next attempt opens a row of its
	// own. Left open, the row would be closed by whatever ended the run —
	// so a task that went on to finish would carry attempts reading
	// "task.finished", stamped at a time none of them ran.
	closesPhase = map[string]bool{
		record.PhaseFinished:  true,
		record.PhaseFailed:    true,
		record.PhaseCancelled: true,
		record.PhaseRetried:   true,
	}
)

// spanOf is the run and the phase this event belongs to, after the event has
// opened or closed either of them.
//
// The order matters. A phase.started belongs to the phase it opens, not to
// the one before it, and a phase.finished belongs to the phase it closes
// rather than to nothing.
func spanOf(tx *sql.Tx, task int64, e record.Event) (int64, int64, error) {
	at := record.Stamp(e.At)

	if e.Kind == record.TaskStarted {
		run, err := openRun(tx, task, at)

		return run, 0, err
	}

	run, err := runOf(tx, task)
	if err != nil || run == 0 {
		return 0, 0, err
	}

	if closesRun[e.Kind] {
		return run, 0, closeRun(tx, run, at, e.Kind)
	}

	return phaseOf(tx, run, at, e)
}

// phaseOf is the phase half of spanOf, split out because the two halves
// together read as one long list of cases and each of them is short.
func phaseOf(tx *sql.Tx, run int64, at string, e record.Event) (int64, int64, error) {
	if e.Kind == record.PhaseStarted {
		phase, err := openPhase(tx, run, at, e)

		return run, phase, err
	}

	phase, err := openPhaseOf(tx, run)
	if err != nil || phase == 0 {
		return run, 0, err
	}

	if closesPhase[e.Kind] {
		return run, phase, closePhase(tx, phase, at, e.Kind)
	}

	return run, phase, nil
}

// openRun starts an attempt, numbered after the ones before it.
func openRun(tx *sql.Tx, task int64, at string) (int64, error) {
	var n int
	if err := tx.QueryRow(countRuns, task).Scan(&n); err != nil {
		return 0, fmt.Errorf("count the attempts so far: %w", err)
	}

	res, err := tx.Exec(insertRun, task, n+1, at)
	if err != nil {
		return 0, fmt.Errorf("open the run row: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read back the run row: %w", err)
	}

	return id, nil
}

// closeRun ends an attempt. The outcome is the kind that ended it, so the
// column speaks the record's own vocabulary rather than a second one that
// would have to be kept in step with it.
func closeRun(tx *sql.Tx, run int64, at, kind string) error {
	if _, err := tx.Exec(endRun, at, kind, run); err != nil {
		return fmt.Errorf("close the run row: %w", err)
	}

	// A run that ends leaves no phase running. A phase still open here is one
	// whose own terminal event never arrived — the process was killed — and
	// leaving it open would have every later event of the task filed under a
	// phase that stopped.
	if _, err := tx.Exec(endOpenPhases, at, kind, run); err != nil {
		return fmt.Errorf("close the phases the run left open: %w", err)
	}

	return nil
}

// openPhase starts a phase inside the current attempt.
func openPhase(tx *sql.Tx, run int64, at string, e record.Event) (int64, error) {
	var n int
	if err := tx.QueryRow(countPhases, run).Scan(&n); err != nil {
		return 0, fmt.Errorf("count the phases so far: %w", err)
	}

	res, err := tx.Exec(insertPhase, run, n+1, e.Phase, e.Data["engine"], e.Data["model"], at)
	if err != nil {
		return 0, fmt.Errorf("open the phase row: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read back the phase row: %w", err)
	}

	return id, nil
}

// closePhase ends a phase, with the kind that ended it as the outcome.
func closePhase(tx *sql.Tx, phase int64, at, kind string) error {
	if _, err := tx.Exec(endPhase, at, kind, phase); err != nil {
		return fmt.Errorf("close the phase row: %w", err)
	}

	return nil
}

// runOf is the attempt a task is on, which is the last one started.
//
// No run at all is not a failure: events arrive before the first
// task.started — task.created is one — and they belong to no attempt.
func runOf(tx *sql.Tx, task int64) (int64, error) {
	var id int64

	err := tx.QueryRow(latestRun, task).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}

	if err != nil {
		return 0, fmt.Errorf("find the run of the task: %w", err)
	}

	return id, nil
}

// openPhaseOf is the phase currently running inside an attempt, if one is.
func openPhaseOf(tx *sql.Tx, run int64) (int64, error) {
	var id int64

	err := tx.QueryRow(runningPhase, run).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}

	if err != nil {
		return 0, fmt.Errorf("find the open phase of the run: %w", err)
	}

	return id, nil
}
