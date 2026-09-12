package db

// The pull requests opened for a task, and what became of each.
//
// One row per opening, and rows are never removed: the record is
// append-only, and opening again after a close is a second row rather than
// a reopened one. What became of it is the state, marked where the merge
// or the close happened rather than inferred later from a branch that is
// gone for three other reasons.
//
// The subselects turn a task id and a repository path into the rows they
// name. Both name rows the events have already folded by the time a pull
// request is opened, merged or closed: a task is worked before it is
// delivered, and working is what joins it to the repository.

import (
	"fmt"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// What a pull request is: open, merged, or closed without merging.
const (
	PROpen   = "open"
	PRMerged = "merged"
	PRClosed = "closed"
)

// A PullRequest is one opening: where, when, and what became of it.
type PullRequest struct {
	Repo     string
	URL      string
	OpenedAt time.Time
	State    string
}

// OpenedPR writes down that a pull request was opened for a task.
func (d *DB) OpenedPR(taskID, repoAbs, url string) error {
	at := record.Stamp(time.Now().UTC())

	if _, err := d.sql.Exec(insertPR, taskID, repoAbs, url, at); err != nil {
		return fmt.Errorf("write down the pull request of %q in %q: %w", taskID, repoAbs, err)
	}

	return nil
}

// MarkPR says what became of every pull request a task has open in one
// repository. Merging and closing are per repository, and so is the mark.
func (d *DB) MarkPR(taskID, repoAbs, state string) error {
	if _, err := d.sql.Exec(markPR, state, taskID, repoAbs); err != nil {
		return fmt.Errorf("mark the pull requests of %q in %q %s: %w", taskID, repoAbs, state, err)
	}

	return nil
}

// PullRequests is every pull request opened for a task, newest first,
// across every repository it was worked in.
func (d *DB) PullRequests(taskID string) ([]PullRequest, error) {
	rows, err := d.sql.Query(selectPRs, taskID)
	if err != nil {
		return nil, fmt.Errorf("read the pull requests of %q: %w", taskID, err)
	}

	defer rows.Close()

	var out []PullRequest

	for rows.Next() {
		var one PullRequest

		var at string

		if err := rows.Scan(&one.Repo, &one.URL, &at, &one.State); err != nil {
			return nil, fmt.Errorf("read a pull request of %q: %w", taskID, err)
		}

		// An unreadable instant is still a row worth showing.
		one.OpenedAt, _ = time.Parse(time.RFC3339Nano, at) //nolint:errcheck

		out = append(out, one)
	}

	return out, rows.Err()
}
