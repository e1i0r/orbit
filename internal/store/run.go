package store

// RunPath is where a run's liveness marker lives: two lines, "pid: <n>" and
// "started: <rfc3339>", written by `orbit run` while it holds a task and
// removed on every exit path. Reading it, and checking whether that pid is
// still alive, is how the window tells a task that is still running from
// one whose process is gone — a later task does the writing and the
// checking; this is only the path.

import "path/filepath"

// RunPath is where one task's run marker lives.
func (s *Store) RunPath(repoPath, taskID string) (string, error) {
	dir, err := s.TaskDir(repoPath, taskID)
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "run"), nil
}
