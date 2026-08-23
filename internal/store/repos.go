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
// A directory whose marker exists but could not be turned into a RepoRef —
// whether because reading it hit a real I/O error (permission denied, or the
// marker path turning out to be a directory) or because it read fine but
// would not parse, including one that names a path that is not absolute —
// is a different thing from a missing marker: it is damage or a fault, not
// a write still in flight, and it is actionable. Either way, Repos never
// lets that one directory stop it from reading the rest: it always finishes
// the whole listing, and the returned slice is always every repository
// whose marker was readable and parseable, regardless of what else failed.
// When something did fail, the returned error is non-nil and joins one
// entry per failing directory, each naming the directory and what went
// wrong. A caller that only wants the list can use the slice as-is; a
// caller that wants to know whether something needs attention checks the
// error.
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
	var failed []error
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
			failed = append(failed, fmt.Errorf("read %q: %w", marker, err))
			continue
		}
		path, ok := parseRepoMarker(string(body))
		if !ok {
			failed = append(failed, fmt.Errorf("repo marker %q is damaged", marker))
			continue
		}
		repos = append(repos, RepoRef{Key: entry.Name(), Path: path})
	}
	return repos, errors.Join(failed...)
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
