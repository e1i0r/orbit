package db

import (
	"errors"
	"fmt"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// Which tasks belong to which repository.
//
// The link is the row task_repo, and it is the one thing in the record that
// no fold can answer: a task's own events say what happened to it, and they
// are read a task at a time, so a question that starts from the repository
// has to be asked of something that crosses tasks. That is what the index is
// for. Both questions are answered by the same table read from either end.

// TasksOfRepo is every task written against one repository, by id.
//
// The path is the absolute one repo.Open resolved, because that is what the
// events carry and two spellings of the same directory are two repositories
// to a string comparison.
//
// The order is by id rather than by when the task was written, which is what
// the directory listing this replaced gave and what every caller of it
// expects to print.
//
// A deleted task is not one of them. Deleting is an event rather than a row
// removed — see record.TaskDeleted — and this is the one enumeration in the
// program: the board, `orbit list` and the reconcile sweep all arrive here,
// so leaving them out once leaves them out everywhere, and no caller can
// forget the rule because no caller states it. The kind is passed rather
// than written into the statement so that it is spelled in exactly one
// place, which is the constant.
func (d *DB) TasksOfRepo(abs string) ([]string, error) {
	rows, err := d.sql.Query(selectTasksOfRepo, abs, record.TaskDeleted)
	if err != nil {
		return nil, fmt.Errorf("read the tasks of %q: %w", abs, err)
	}

	defer rows.Close()

	var ids []string

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read a task of %q: %w", abs, err)
		}

		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// ReposOfTask is the other end: every repository one task reaches into,
// oldest join first.
//
// A task usually has one, and the plural is not hypothetical — a task that
// opens a worktree in a second checkout joins it, and the migration reads
// this to know which joins it has already carried across.
func (d *DB) ReposOfTask(taskID string) ([]string, error) {
	rows, err := d.sql.Query(selectReposOfTask, taskID)
	if err != nil {
		return nil, fmt.Errorf("read the repositories of %q: %w", taskID, err)
	}

	defer rows.Close()

	var paths []string

	for rows.Next() {
		var abs string
		if err := rows.Scan(&abs); err != nil {
			return nil, fmt.Errorf("read a repository of %q: %w", taskID, err)
		}

		paths = append(paths, abs)
	}

	return paths, rows.Err()
}

// Worked is one task and one repository it is worked in.
type Worked struct {
	Task string // the task's id
	Path string // the repository's absolute path
	Name string // the repository's name, as the row that made the link spelled it
}

// TasksAndRepos is every task the record holds, with every repository it is
// worked in, by id and then oldest join first.
//
// It is one question rather than the same question asked once per
// repository, because a task that reaches into four checkouts is answered by
// four of those and the board has to put the four answers back together to
// draw one row. Asked from this end the row is already whole, and the join
// order — which repository the task was written in, and which ones the work
// went on to — survives, where a walk of the repositories would have
// returned them in whatever order the repositories were walked in.
//
// A deleted task is left out here for the reason it is left out of
// TasksOfRepo, and by the same subselect.
func (d *DB) TasksAndRepos() ([]Worked, error) {
	rows, err := d.sql.Query(selectTasksAndRepos, record.TaskDeleted)
	if err != nil {
		return nil, fmt.Errorf("read the tasks and their repositories: %w", err)
	}

	defer rows.Close()

	var worked []Worked

	for rows.Next() {
		var w Worked
		if err := rows.Scan(&w.Task, &w.Path, &w.Name); err != nil {
			return nil, fmt.Errorf("read a task and its repository: %w", err)
		}

		worked = append(worked, w)
	}

	return worked, rows.Err()
}

// Join links a task to a repository without an event saying so.
//
// It is what the migration writes, and it is the one place a task_repo row
// is made by anything but an event. The older Orbit kept the link in a file
// beside the task — tasks/<id>/repos, an absolute path a line — and never
// recorded it as something that happened, so there is no event to carry
// across. Appending one now would put a thing in the record that was never
// in it, dated from whenever the upgrade was run, and every reader of that
// task's history would show it.
//
// at is when the link was made as well as anybody knows, which for a
// migrated state root is when its file was last written.
//
// It is harmless twice: both inserts already have to be, because a
// repository joins again on every retry.
func (d *DB) Join(taskID, abs, name string, at time.Time) error {
	e := record.Event{
		Kind: record.RepoJoined,
		At:   at,
		Data: map[string]string{"path": abs, "repo": name},
	}

	if err := keepTrying(func() error { return d.joinOnce(taskID, e) }); err != nil {
		return fmt.Errorf("join %q to %q: %w", abs, taskID, err)
	}

	return nil
}

// joinOnce is one attempt: the task's row if it has none yet, then the link.
func (d *DB) joinOnce(taskID string, e record.Event) error {
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}

	task, err := taskRow(tx, taskID, e)
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}

	if err := joined(tx, task, e); err != nil {
		return errors.Join(err, tx.Rollback())
	}

	return tx.Commit()
}
