package store

// Which repositories a task is worked in, now that its directory no longer
// says. The path used to answer it — repos/<key>/tasks/<id> — and a task
// that reaches into two checkouts had nowhere to be filed. So the answer
// moved inside the task, to a file that can hold more than one line.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// joinRepo writes down that a task is worked in a repository, once. The file
// is a list because a task may reach into several, and it is append-only for
// the same reason the record is: a repository that took part took part, and
// a later line cannot make that untrue.
func (s *Store) joinRepo(taskID, abs string) error {
	joined, err := s.TaskRepos(taskID)
	if err != nil {
		return err
	}

	for _, p := range joined {
		if p == abs {
			return nil
		}
	}

	dir, err := s.TaskDir(taskID)
	if err != nil {
		return err
	}

	marker := filepath.Join(dir, "repos")

	f, err := os.OpenFile(marker, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileMode)
	if err != nil {
		return fmt.Errorf("open %q: %w", marker, err)
	}

	if _, err := f.WriteString("path: " + abs + "\n"); err != nil {
		return errors.Join(fmt.Errorf("write %q: %w", marker, err), f.Close())
	}

	return f.Close()
}

// TaskRepos is every repository one task has been worked in, in the order
// they joined. A task whose marker is missing answers nothing rather than
// failing: that is a task directory made by a hand or an older Orbit, and
// the listing that calls this is better short than stopped.
func (s *Store) TaskRepos(taskID string) ([]string, error) {
	dir, err := s.TaskDir(taskID)
	if err != nil {
		return nil, err
	}

	marker := filepath.Join(dir, "repos")

	body, err := os.ReadFile(marker)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read %q: %w", marker, err)
	}

	var paths []string

	for _, line := range strings.Split(string(body), "\n") {
		if line == "" {
			continue
		}

		if path, ok := parseRepoMarker(line + "\n"); ok {
			paths = append(paths, path)
		}
	}

	return paths, nil
}

// TaskIDs is every task the state root holds, sorted.
func (s *Store) TaskIDs() ([]string, error) {
	entries, err := os.ReadDir(s.TasksDir())
	if errors.Is(err, os.ErrNotExist) {
		// Nobody has written a task yet. Reading a path creates nothing,
		// so the directory is genuinely absent until the first one.
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read %q: %w", s.TasksDir(), err)
	}

	ids := make([]string, 0, len(entries))

	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}

	sort.Strings(ids)

	return ids, nil
}

// TaskIDsOfRepo is every task that has been worked in one repository.
//
// A task that names no repository at all is left out rather than shown
// everywhere: the caller asked which tasks belong to this checkout, and
// "we do not know" is not an answer to that question.
func (s *Store) TaskIDsOfRepo(repoPath string) ([]string, error) {
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve %q: %w", repoPath, err)
	}

	all, err := s.TaskIDs()
	if err != nil {
		return nil, err
	}

	var (
		ids    []string
		failed []error
	)

	for _, id := range all {
		joined, err := s.TaskRepos(id)
		if err != nil {
			failed = append(failed, err)
			continue
		}

		for _, p := range joined {
			if p == abs {
				ids = append(ids, id)
				break
			}
		}
	}

	return ids, errors.Join(failed...)
}
