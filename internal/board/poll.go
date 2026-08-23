package board

// The two things a poll does to one thing: read the tail of one task's log,
// and find out what one repository is called and what is in it. Refresh and
// Rescan are the loops; these are what they call.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// poll reads whatever has been appended to one task's log since the last
// refresh, and returns the verdict on having done so.
func (r *Reader) poll(st *taskState) ([]record.Event, error) {
	info, err := os.Stat(st.path)
	if errors.Is(err, os.ErrNotExist) {
		// A task written down whose run has never started has no log at
		// all, and that is an answer rather than a fault — the same
		// treatment record.Read gives it.
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", st.path, err)
	}

	size := info.Size()
	if size == st.size {
		// The one stat this whole design is built on. Nothing was appended,
		// so there is nothing to open, seek, read or parse, and the last
		// verdict stands — which is also what stops a log nobody can read
		// from being re-read twice a second and from quietly stopping being
		// reported.
		return nil, st.err
	}
	st.size = size

	if size < st.offset {
		// Shorter than what has already been read, so this log was replaced
		// rather than appended to. ReadFrom answers that on its own by
		// starting from the top; what it cannot know is that the events read
		// from the log it replaced are no longer this task's history, and
		// folding both together would show one task attempted twice. So they
		// go with the offset. Clearing seen is what makes the caller fold
		// again even if the replacement has nothing in it yet.
		st.offset, st.events, st.seen = 0, nil, false
	}

	// The offset is only ever what ReadFrom answered, never arithmetic of
	// this package's own: ReadFrom advances past a complete,
	// newline-terminated line and no further, and a torn final line is a
	// write in flight rather than damage. Tracking bytes here would defeat
	// exactly that property.
	events, next, err := record.ReadFrom(st.path, st.offset)
	if err != nil {
		return nil, err
	}
	st.offset = next
	return events, nil
}

// open turns a marker into the repository the rows name, calling repo.Open
// once per repository for as long as this reader lives.
//
// A repository's name does not change while the window is open, and
// repo.Open is three git subprocesses; answering it again every two seconds
// would cost more than the whole polling design saves. Successes are
// therefore kept and failures are not, so a repository that comes back — a
// volume remounted, a checkout restored — is picked up on the next
// enumeration rather than at the next restart.
//
// A repository that will not open is reported and keeps its tasks. The
// record lives in the state root and the checkout does not, so a checkout
// that has been moved or deleted takes nothing on screen with it: the rows
// fold exactly as before, under the name the marker's path ends in.
func (r *Reader) open(ref store.RepoRef) (*repoState, error) {
	if rs, ok := r.opened[ref.Path]; ok {
		return rs, nil
	}
	opened, err := repo.Open(ref.Path)
	if err != nil {
		return &repoState{path: ref.Path, name: filepath.Base(ref.Path)},
			fmt.Errorf("open the repository at %q: %w", ref.Path, err)
	}
	// The marker's path is kept and git's top level is discarded, because
	// the store files this repository's record under a hash of the former;
	// see repoState.
	rs := &repoState{path: ref.Path, name: opened.Name}
	r.opened[ref.Path] = rs
	return rs, nil
}

// list is every task directory of one repository, in the order the rows are
// drawn — os.ReadDir answers in filename order, which for task ids is the
// order they are listed in everywhere else.
func (r *Reader) list(rs *repoState) ([]string, error) {
	dir, err := r.store.TasksDir(rs.path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		// A repository nobody has written a task against yet. Computing a
		// path creates nothing, so this directory is genuinely absent until
		// the first task, and "no tasks" is an answer rather than a fault.
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list the tasks of %q: %w", rs.name, err)
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	return ids, nil
}
