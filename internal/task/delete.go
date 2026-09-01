package task

// Taking a task off the board without unwriting what it did.

import (
	"errors"
	"os"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// Delete takes a task off every listing and gives up its checkout.
//
// It is a soft delete, and the asymmetry is deliberate: the worktree goes
// for real and the record stays. A checkout of a task nobody can see is a
// lie on the disk — and the only part of a deleted task that costs anything
// worth reclaiming — while the record is the only account there is of what
// an engine was asked, what it spent and what it changed. A reader tidying a
// board is not asking for that account to be destroyed. What they get is a
// row that is gone from the board, from `orbit list` and from the reconcile
// sweep, all three through the one enumeration that reads the event.
//
// The worktree goes through git rather than through os.RemoveAll, because
// the bookkeeping under .git/worktrees is the one thing Orbit writes into a
// repository it does not own, and a checkout removed by hand leaves an entry
// behind that only `git worktree prune` clears. It goes first, so that a
// task whose checkout could not be given up is still on the board and can be
// asked again.
//
// The path and not the name. Every directory the store keeps is under a hash
// of the repository's path, and this gesture was once given both by the name
// a row displays: filepath.Abs("payments") is whatever directory orbit was
// started in plus "payments", so it deleted the right task when the window
// happened to be opened from the workspace root and silently deleted nothing
// anywhere else. It said "task deleted" either way.
//
// What went wrong comes back, joined. The window puts it in the activity
// band, which is the difference between a row that stays for a reason and
// one that stays for none.
func Delete(s *store.Store, t Task) error {
	var errs []error

	wtDir, err := s.WorktreeDir(t.Repo.Path, t.ID)

	switch {
	case err != nil:
		errs = append(errs, err)
	case exists(wtDir):
		if err := t.Repo.RemoveWorktree(wtDir); err != nil {
			errs = append(errs, err)
		}
	}

	if err := emit(s, t, record.Event{Kind: record.TaskDeleted}); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// exists is whether there is anything at path. A stat that failed for any
// other reason answers false: what follows it is a removal, and git says
// what it could not read better than this line could.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
