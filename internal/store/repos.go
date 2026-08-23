package store

// Repos lists every repository the state root has record of, by reading the
// marker createRepoDir writes the first time a repository gets a task: the
// directory name is a hash and answers nothing on its own, and this is the
// reader that turns it back into a path.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RepoRef is one repository the store knows about: Key is the hashed
// directory name under repos/, Path is the absolute path createRepoDir
// wrote into that directory's marker.
type RepoRef struct {
	Key, Path string
}

// Repos lists every repository under repos/ that has a marker.
//
// A root with no repos/ directory at all has simply never held a task, and
// that is empty, not an error — the same treatment Read gives a log that
// has not started. A directory whose marker is missing or will not parse is
// a repository directory that was made but never finished (createRepoDir
// makes the directory before it writes the file), and it is skipped rather
// than reported: a half-created directory is not an error this reader can
// do anything about.
func (s *Store) Repos() ([]RepoRef, error) {
	dir := filepath.Join(s.root, "repos")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", dir, err)
	}

	var repos []RepoRef
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		marker := filepath.Join(dir, entry.Name(), "repo")
		body, err := os.ReadFile(marker)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", marker, err)
		}
		path, ok := parseRepoMarker(string(body))
		if !ok {
			continue
		}
		repos = append(repos, RepoRef{Key: entry.Name(), Path: path})
	}
	return repos, nil
}

// parseRepoMarker reads back what createRepoDir wrote: "path: /abs/path\n".
// Anything else is treated the same as a missing marker — not this reader's
// problem to fix.
func parseRepoMarker(body string) (string, bool) {
	rest, ok := strings.CutPrefix(body, "path: ")
	if !ok {
		return "", false
	}
	rest = strings.TrimSuffix(rest, "\n")
	if rest == "" {
		return "", false
	}
	return rest, true
}
