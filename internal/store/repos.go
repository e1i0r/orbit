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

// ReposError is the error Repos returns when the repos/ directory itself
// could not be listed. It is the only failure that leaves a caller with no
// list at all rather than a shorter one: every other fault costs a single
// directory and is reported alongside the repositories that did read.
//
// It is a distinct type because the length of the returned slice cannot tell
// the two apart. A root holding one repository whose marker is damaged
// returns no repositories and a non-nil error, and so does a root that
// cannot be listed at all — but the first is one repository to be repaired
// and the second is the whole listing gone. A caller deciding whether to
// keep drawing asks with errors.As rather than counting.
type ReposError struct {
	Dir string // the repos/ directory that could not be listed
	Err error  // what os.ReadDir said about it
}

func (e *ReposError) Error() string {
	return fmt.Sprintf("read %q: %v", e.Dir, e.Err)
}

func (e *ReposError) Unwrap() error { return e.Err }

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
//
// The one failure that is not per-directory is repos/ itself refusing to be
// listed, and that comes back as a *ReposError with no repositories at all.
// It is the only error this function returns that means the listing did not
// happen, as against happening and finding damage.
func (s *Store) Repos() ([]RepoRef, error) {
	dir := filepath.Join(s.root, "repos")

	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, &ReposError{Dir: dir, Err: err}
	}

	var (
		repos  []RepoRef
		failed []error
	)

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

// ForgetRepo removes one repository's record from the state root, and
// answers the directory it removed.
//
// What goes is the record and only the record: every task's events.jsonl,
// task.md, run marker and control file for that repository. It is the one
// operation in Orbit that deletes from the append-only log, so it is spelled
// as its own verb rather than reached by removing a directory, and the
// caller is the one that has to decide there is nothing there worth keeping.
//
// What stays is the worktrees. They live under worktrees/ rather than under
// repos/ — see WorktreeDir — and they are checkouts git itself has
// registered in the repository; removing them from underneath it would leave
// the repository's worktree list naming directories that are not there. A
// caller that wants them gone runs `orbit cancel` first, which is what
// removes a worktree through git.
//
// A repository the root has no record of is an error rather than a silent
// success: "forgotten" and "never known" are different answers, and a caller
// that misspelled a path deserves to hear which one it got.
func (s *Store) ForgetRepo(repoPath string) (string, error) {
	dir, err := s.RepoDir(repoPath)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("orbit has no record of a repository at %q", repoPath)
		}

		return "", fmt.Errorf("read %q: %w", dir, err)
	}

	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("remove %q: %w", dir, err)
	}

	return dir, nil
}
