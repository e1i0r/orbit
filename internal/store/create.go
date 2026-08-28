package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// CreateTaskDir makes the directory one task's record lives in, and every
// directory above it, and returns the path.
//
// Creation is a verb of its own so that the path methods can stay pure. Only
// a caller that is about to write something calls this.
func (s *Store) CreateTaskDir(repoPath, taskID string) (string, error) {
	if _, err := s.createRepoDir(repoPath); err != nil {
		return "", err
	}
	dir, err := s.TaskDir(repoPath, taskID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return "", fmt.Errorf("create %q: %w", dir, err)
	}
	return dir, nil
}

// RegisterRepo records a repository in the state root without writing a task
// against it.
//
// It exists because the repos/ listing was, until now, a side effect: a
// repository appeared in it the first time somebody ran a task there, and
// there was no way to say "orbit should know about this checkout" on its
// own. Anything that walks a directory finds repositories by itself, so this
// is for the callers that cannot walk — a server answering a tool call, told
// a path by a model that is not in that directory and cannot cd to it.
func (s *Store) RegisterRepo(repoPath string) (string, error) {
	return s.createRepoDir(repoPath)
}

// CreateWorktreeParent makes the directory a task's checkout will sit in and
// returns the path of the checkout itself, which it does not create: `git
// worktree add` insists on making that leaf for itself.
func (s *Store) CreateWorktreeParent(repoPath, taskID string) (string, error) {
	dir, err := s.WorktreeDir(repoPath, taskID)
	if err != nil {
		return "", err
	}
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, dirMode); err != nil {
		return "", fmt.Errorf("create %q: %w", parent, err)
	}
	return dir, nil
}

// createRepoDir makes a repository's directory and, the first time, writes
// the file that says which repository it is.
//
// The directory is named by a truncated hash, which answers no questions on
// its own. `cat repos/<hash>/repo` now answers the only one that matters —
// and if two repositories ever did land on one key, the mismatch would be
// visible in that file instead of silently sharing a record.
func (s *Store) createRepoDir(repoPath string) (string, error) {
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", repoPath, err)
	}
	dir, err := s.RepoDir(abs)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return "", fmt.Errorf("create %q: %w", dir, err)
	}
	marker := filepath.Join(dir, "repo")
	if _, statErr := os.Stat(marker); errors.Is(statErr, os.ErrNotExist) {
		if err := os.WriteFile(marker, []byte("path: "+abs+"\n"), fileMode); err != nil {
			return "", fmt.Errorf("write %q: %w", marker, err)
		}
	}
	return dir, nil
}
