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
// has not started. A directory whose marker is missing is a repository
// directory that was made but never finished (createRepoDir makes the
// directory before it writes the file), and it is skipped rather than
// reported: a half-created directory is not an error this reader can do
// anything about.
//
// A directory whose marker exists but will not parse — including one that
// names a path that is not absolute — is a different thing from a missing
// marker: it is damage, not a write still in flight, and it is actionable.
// Repos never lets one damaged marker blank the rest of the list: every
// repository it could read still comes back in the returned slice. But when
// it meets a marker like that, the returned error is non-nil and names the
// offending directory, joined with any others found the same way. A caller
// that only wants the list can use the slice as-is; a caller that wants to
// know whether something needs attention checks the error.
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
	var damaged []error
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
			return repos, fmt.Errorf("read %q: %w", marker, err)
		}
		path, ok := parseRepoMarker(string(body))
		if !ok {
			damaged = append(damaged, fmt.Errorf("repo marker %q is damaged", marker))
			continue
		}
		repos = append(repos, RepoRef{Key: entry.Name(), Path: path})
	}
	return repos, errors.Join(damaged...)
}

// parseRepoMarker reads back what createRepoDir wrote: "path: /abs/path\n".
// Anything else — including a path that is not absolute, which createRepoDir
// never writes — is reported to Repos as a damaged marker.
func parseRepoMarker(body string) (string, bool) {
	rest, ok := strings.CutPrefix(body, "path: ")
	if !ok {
		return "", false
	}
	rest = strings.TrimSuffix(rest, "\n")
	if rest == "" || !filepath.IsAbs(rest) {
		return "", false
	}
	return rest, true
}
