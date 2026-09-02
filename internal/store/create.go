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
//
// An empty repoPath is a task that is not worked in any repository yet, and
// it gets the directory and nothing else: no repository is registered and no
// link is written. The task's own directory is under tasks/ and has never
// been under a repository's, so there is nothing here that a task with no
// repository is missing — which is what makes writing one down a matter of
// leaving a step out rather than a second way of writing a task.
func (s *Store) CreateTaskDir(repoPath, taskID string) (string, error) {
	dir, err := s.TaskDir(taskID)
	if err != nil {
		return "", err
	}

	if repoPath != "" {
		if _, err := s.createRepoDir(repoPath); err != nil {
			return "", err
		}
	}

	if err := os.MkdirAll(dir, dirMode); err != nil {
		return "", fmt.Errorf("create %q: %w", dir, err)
	}

	if repoPath == "" {
		return dir, nil
	}

	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", repoPath, err)
	}

	if err := s.JoinRepo(taskID, abs); err != nil {
		return "", err
	}

	return dir, nil
}

// CreateWorkDir makes the directory a phase runs in when the task has no
// repository, and returns it.
//
// A run needs somewhere to be, and a task that has reached into nothing has
// no checkout to offer. It gets a directory of its own beside its record —
// tasks/<id>/work — where the engine can read what it was given, write
// scratch, and run the command that opens the first real checkout. Nothing
// is committed from here and nothing is delivered from here: the moment the
// work is about a repository, `orbit join` answers with a worktree, and that
// is where the code goes.
func (s *Store) CreateWorkDir(taskID string) (string, error) {
	dir, err := s.TaskDir(taskID)
	if err != nil {
		return "", err
	}

	work := filepath.Join(dir, "work")
	if err := os.MkdirAll(work, dirMode); err != nil {
		return "", fmt.Errorf("create %q: %w", work, err)
	}

	return work, nil
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
// its own. `cat repos/<hash>/repo` answers the only one that matters.
//
// And when that file already names a different repository, this refuses.
// Twelve hex characters is 48 bits, which will not collide over a handful of
// paths — but "will not" is not "cannot", and the cost of being wrong is
// silent: two repositories filed under one record, each shown the other's
// tasks, with the marker still naming whichever of them got there first.
// Writing the marker only when it was missing never protected against that;
// it is what made it quiet. The refusal names both paths, which is what
// whoever reads it needs in order to move one of them.
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

	body, readErr := os.ReadFile(marker)
	switch {
	case errors.Is(readErr, os.ErrNotExist):
		if err := os.WriteFile(marker, []byte("path: "+abs+"\n"), fileMode); err != nil {
			return "", fmt.Errorf("write %q: %w", marker, err)
		}
	case readErr != nil:
		return "", fmt.Errorf("read %q: %w", marker, readErr)
	default:
		// A marker that will not parse is damage, and Repos already
		// reports it as damage; rewriting it here would erase the only
		// trace of it. Only a marker that reads cleanly and names
		// somewhere else is a collision.
		if known, ok := parseRepoMarker(string(body)); ok && known != abs {
			return "", fmt.Errorf("%q and %q both hash to the store key %q, so orbit would file them under one record: move one of the two, or remove %q if it is stale", abs, known, filepath.Base(dir), dir)
		}
	}

	return dir, nil
}
