package board

// The two questions the task view asks that the board itself does not
// answer: one task's whole record, and where that task's worktree is.
//
// They live here because of the layering rather than because they belong to
// the polling design. internal/ui may not import internal/record,
// internal/store or internal/engine — the window is not allowed to know the
// record's own shape, because a window that can read it is a window one
// commit away from writing it — and this package already holds the store and
// already imports the record, the task and the view. So the fact travels out
// through a method here, in the type the window is already given, and no row
// of the architecture's import table has to widen for it.

import (
	"fmt"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
)

// Log is one task's record, folded into the entries the task view draws.
//
// A task the poller already knows is answered from the events it has read,
// which is the whole of the offsets' value here: the log tab of a running
// task is redrawn twice a second and re-reading a 200 KB log at that rate to
// learn that four lines were appended is exactly what this package exists
// not to do. A task it does not know — one whose repository has not been
// enumerated yet, or one the reader opened in the same tick it was written —
// is read from disk once, and the next refresh picks it up the cheap way.
func (r *Reader) Log(repoPath, id string) ([]view.Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// The lock is held across the fallback read, and that is a considered
	// trade rather than an oversight: the alternative is releasing it and
	// racing a rescan that is removing this very task's state. One read of
	// one log delays one 500 ms poll; a use-after-free does worse.
	if st, ok := r.index[id]; ok && st.seen {
		return view.Log(st.events), nil
	}

	events, err := task.Events(r.store, task.Task{ID: id, Repo: repo.Repo{Path: repoPath}})
	if err != nil {
		return nil, fmt.Errorf("read the record of task %s: %w", id, err)
	}

	return view.Log(events), nil
}

// Worktree is where a task's throwaway checkout lives, for the one caller
// that has to run a command inside it.
//
// It answers the path and does not create it, which matters to both callers
// the window has: the diff of a task that has never run must fail as a
// missing directory rather than quietly succeed against an empty one, and
// the editor must refuse to open a file in a worktree that was cleaned up.
// The path is a pure function of the repository and the id, so no lock is
// taken — the store this is asked of does not change after NewReader.
func (r *Reader) Worktree(repoPath, id string) (string, error) {
	path, err := r.store.WorktreeDir(repoPath, id)
	if err != nil {
		return "", fmt.Errorf("locate the worktree of task %s: %w", id, err)
	}

	return path, nil
}
