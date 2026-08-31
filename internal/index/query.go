package index

// The questions the index is for: the ones that span tasks, which a tree of
// logs answers only by opening all of them.

import (
	"fmt"
)

// TaskIDsOfRepo is every task that has been worked in one repository, in the
// order they joined it.
func (x *Index) TaskIDsOfRepo(repoPath string) ([]string, error) {
	rows, err := x.db.Query(`
		SELECT tasks.task_id
		FROM task_repos
		JOIN tasks ON tasks.id = task_repos.task_id
		JOIN repos ON repos.id = task_repos.repo_id
		WHERE repos.abs_path = ?
		ORDER BY task_repos.joined_at, tasks.task_id`, repoPath)
	if err != nil {
		return nil, fmt.Errorf("ask which tasks are worked in %q: %w", repoPath, err)
	}

	defer rows.Close()

	var ids []string

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read a task worked in %q: %w", repoPath, err)
		}

		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// ReposOfTask is every repository one task was worked in.
func (x *Index) ReposOfTask(taskID string) ([]string, error) {
	rows, err := x.db.Query(`
		SELECT repos.abs_path
		FROM task_repos
		JOIN tasks ON tasks.id = task_repos.task_id
		JOIN repos ON repos.id = task_repos.repo_id
		WHERE tasks.task_id = ?
		ORDER BY task_repos.joined_at, repos.abs_path`, taskID)
	if err != nil {
		return nil, fmt.Errorf("ask where %q was worked: %w", taskID, err)
	}

	defer rows.Close()

	var paths []string

	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("read a repository of %q: %w", taskID, err)
		}

		paths = append(paths, path)
	}

	return paths, rows.Err()
}

// CountOfKind is how many events of one kind the whole index holds. It is
// the smallest question that is genuinely across tasks, and the one the
// supervisor's later questions are all shaped like.
func (x *Index) CountOfKind(kind string) (int, error) {
	var n int
	if err := x.db.QueryRow(`SELECT count(*) FROM events WHERE kind = ?`, kind).Scan(&n); err != nil {
		return 0, fmt.Errorf("count %q: %w", kind, err)
	}

	return n, nil
}
