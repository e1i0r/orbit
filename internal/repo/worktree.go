package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AddWorktree creates a throwaway checkout of the base branch on a branch of
// its own.
//
// A worktree rather than a clone because it shares the object store, so it
// costs almost nothing and can be thrown away without losing anything. A
// branch of its own because a task must never be able to move the branch a
// human is standing on.
//
// The branch is created when it is not there and reused when it is. A
// worktree directory deleted by hand leaves its branch behind, and that is
// an ordinary thing for a person to do: the second run picks the branch up
// where the first one left it rather than failing on the collision.
func (r Repo) AddWorktree(dir, branch string) error {
	if r.Base == "" {
		return fmt.Errorf("%q is not on a branch — check one out before starting a task against it", r.Path)
	}

	if err := mkdir(filepath.Dir(dir)); err != nil {
		return err
	}

	args := []string{"worktree", "add", "-b", branch, dir, r.Base}
	if r.hasBranch(branch) {
		// Clear the bookkeeping git keeps for a worktree whose directory
		// is no longer there before checking the branch out again:
		// without it git refuses, saying the branch is already used by a
		// worktree that has not existed since the person deleted it.
		// prune only ever removes entries whose directory is gone.
		if _, err := git(r.Path, "worktree", "prune"); err != nil {
			return fmt.Errorf("prune worktrees of %q: %w", r.Path, err)
		}

		args = []string{"worktree", "add", dir, branch}
	}

	if _, err := git(r.Path, args...); err != nil {
		return fmt.Errorf("create a worktree for %q at %q: %w", branch, dir, err)
	}

	// What it was cut from, written beside the worktree's own bookkeeping.
	// Nothing else records it: the repository's own checkout moves on, and a
	// diff taken against wherever it stands now counts commits this task
	// never made — or, once that checkout is detached, counts nothing at all
	// and lets the over-diff gate pass anything.
	//
	// A file under the worktree's git directory and not `git config`. The
	// config is one file for the whole repository and git locks it to write:
	// two tasks starting at once in one repository collided there, and the
	// loser's run died on a lock rather than on anything about the work.
	// This file is the worktree's own, and git removes it with the worktree.
	//
	// A base that could not be written down is not worth a failed run: the
	// count falls back to the repository's own branch, which is what every
	// worktree was measured against before any of this was recorded.
	_ = writeBase(dir, r.Base) //nolint:errcheck // best effort: cutFrom falls back to r.Base

	// Orbit's own directory, kept out of what the task hands back. Said
	// rather than swallowed, unlike the base above: a base that could not be
	// written costs a count its accuracy, and an exclude that could not be
	// written puts Orbit's files in somebody's pull request.
	if err := excludeOrbit(dir); err != nil {
		return err
	}

	return nil
}

// baseFile is where a worktree records what it was cut from.
const baseFile = "orbitbase"

// writeBase puts the branch a worktree was cut from beside git's own
// bookkeeping for it.
func writeBase(dir, base string) error {
	gitDir, err := git(dir, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return fmt.Errorf("find the git directory of %q: %w", dir, err)
	}

	path := filepath.Join(strings.TrimSpace(gitDir), baseFile)
	if err := os.WriteFile(path, []byte(base), 0o600); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}

	return nil
}

// hasBranch reports whether a branch already exists.
func (r Repo) hasBranch(branch string) bool {
	_, err := git(r.Path, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

// RemoveWorktree deletes the checkout and the bookkeeping git keeps for it
// inside the repository.
//
// This bookkeeping under .git/worktrees is the one thing Orbit writes into a
// repository it does not own, so it is cleaned up rather than left behind.
//
// The prune after the removal is not redundant with it. `worktree remove`
// only knows about worktrees that are still there; prune is the only thing
// that clears the entry left behind when a person deletes the directory by
// hand, which is the same situation AddWorktree reuses a branch for.
func (r Repo) RemoveWorktree(dir string) error {
	if _, err := git(r.Path, "worktree", "remove", "--force", dir); err != nil {
		return fmt.Errorf("remove the worktree at %q: %w", dir, err)
	}

	if _, err := git(r.Path, "worktree", "prune"); err != nil {
		return fmt.Errorf("prune worktrees of %q: %w", r.Path, err)
	}

	return nil
}
