package store

// Which repositories a task is worked in, now that its directory no longer
// says. The path used to answer it — repos/<key>/tasks/<id> — and a task
// that reaches into two checkouts had nowhere to be filed. So the answer
// moved inside the task, to a file that can hold more than one line.

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// JoinRepo writes down that a task is worked in a repository, once. The file
// is a list because a task may reach into several, and it is append-only for
// the same reason the record is: a repository that took part took part, and
// a later line cannot make that untrue.
//
// It is exported for the one caller outside this package that writes a task
// directory rather than reads one: `orbit export` builds the tree the record
// used to live in, and the shape of the file it builds is this package's to
// say. A second writer of the same three words is how the two halves of a
// format drift apart.
func (s *Store) JoinRepo(taskID, abs string) error {
	joined, err := s.TaskRepos(taskID)
	if err != nil {
		return err
	}

	for _, p := range joined {
		if p == abs {
			return nil
		}
	}

	marker, err := s.TaskReposPath(taskID)
	if err != nil {
		return err
	}

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
	marker, err := s.TaskReposPath(taskID)
	if err != nil {
		return nil, err
	}

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
