package repo

import (
	"fmt"
	"path/filepath"
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
