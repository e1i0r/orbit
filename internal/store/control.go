package store

// ControlPath is where a reader leaves one word for a run in flight —
// `pause`, `resume`, `cancel`, `continue`, `skip` — written by `orbit pause`
// and by the window, read by the runner at its next phase boundary and taken
// off the moment it is read. A later task does the writing and the reading;
// this is only the path.
//
// It sits beside the run marker rather than inside it because the two say
// different things and are written by different people: the marker is the
// run's own claim on the task, and this is the reader's word to it. One file
// per fact keeps `cat` an answer to both questions.

import "path/filepath"

// ControlPath is where one task's control word lives.
func (s *Store) ControlPath(repoPath, taskID string) (string, error) {
	dir, err := s.TaskDir(repoPath, taskID)
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "control"), nil
}
