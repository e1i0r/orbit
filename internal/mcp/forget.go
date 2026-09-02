package mcp

// What forgetting a repository takes with it, now that a task can be worked
// in more than one.
//
// The rule used to be simple because the shape was: a task lived under the
// repository it was written in, so removing that directory removed the task,
// and the tool refused until it was told to do it anyway. A task's record
// lives at the root of the state tree now and the link to a repository is a
// row beside it, so removing the directory removes neither — the tool went
// on answering `tasks_deleted: 1` for a task that was still on the board,
// which is the worst answer a tool can give: a caller reads it, believes the
// cleanup happened, and does not look again.
//
// So the deletion is made here, explicitly, and only of what forgetting
// really ends. A task worked in this repository and nowhere else has nothing
// left when it goes. One that was carried on into another checkout is still
// being worked, and taking it off the board because one of its repositories
// was forgotten would delete work nobody asked about.

import (
	"errors"
	"fmt"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
)

// leaving is what becomes of the tasks worked in a repository that is about
// to be forgotten.
type leaving struct {
	ending []string // worked here and nowhere else, so forgetting ends them
	kept   []string // worked here and elsewhere too, so they go on without it
}

// departing sorts the tasks of a repository into those two.
//
// It asks the record which repositories each task reaches into rather than
// reading the board, because a repository being forgotten is one a reader is
// cleaning up: the checkout may be gone, and a board that could not discover
// it would answer that the task is worked nowhere at all.
func departing(s *store.Store, ids []string, path string) (leaving, error) {
	d, err := s.Record()
	if err != nil {
		return leaving{}, err
	}

	var out leaving

	for _, id := range ids {
		paths, err := d.ReposOfTask(id)
		if err != nil {
			return leaving{}, err
		}

		if elsewhere(paths, path) {
			out.kept = append(out.kept, id)
			continue
		}

		out.ending = append(out.ending, id)
	}

	return out, nil
}

// elsewhere reports whether any of a task's repositories is not this one.
func elsewhere(paths []string, path string) bool {
	for _, p := range paths {
		if p != path {
			return true
		}
	}

	return false
}

// endTasks deletes the tasks forgetting the repository ends.
//
// task.Delete and not a removal of its own: it is the same gesture `orbit
// delete` makes, so a task deleted through this tool is deleted the way
// every other deleted task is — the worktree given up through git, the
// record left standing with task.deleted written into it, and the row gone
// from every listing that reads the event.
//
// The failures are joined rather than returned at the first one. A worktree
// that would not come away is a reason to tell the caller, not a reason to
// leave the other nineteen tasks half-forgotten.
func endTasks(s *store.Store, r repo.Repo, ids []string) error {
	var errs []error

	for _, id := range ids {
		if err := task.Delete(s, task.Task{ID: id, Repo: r}); err != nil {
			errs = append(errs, fmt.Errorf("delete task %s: %w", id, err))
		}
	}

	return errors.Join(errs...)
}

// names is a list of task ids that is never nil, so that a caller reading
// `"tasks_kept": null` does not have to decide whether that means none or
// unknown.
func names(ids []string) []string {
	if ids == nil {
		return []string{}
	}

	return ids
}
