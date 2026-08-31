package board

// The two loops, on the two clocks: refresh every task the last enumeration
// found and coalesce what moved into one answer, and, more slowly, walk the
// tree again to find what has been written down since. What either of them
// does to a single task or repository is poll.go.

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
)

// Refresh polls every task the last enumeration found and returns the
// board, what changed since the previous call, and an error only when there
// is nothing at all to draw.
//
// It is an ordinary blocking function and not a tea.Cmd on purpose. Every
// subtle behaviour of the window lives in here, and in here it is testable
// against a temporary state root with no terminal anywhere in the picture;
// the window's command is three lines that call this.
//
// The first call enumerates, so a reader is useful before the first two
// second tick arrives. After that only Rescan enumerates, which is what
// makes the half-second clock cheap enough to run.
//
// Per-task and per-repository failures are carried in Board.Errs rather
// than returned: one task whose log is unreadable must not blank the other
// nineteen. The returned error is reserved for a root that cannot be walked
// at all, where an empty board would be indistinguishable from an empty
// directory.
func (r *Reader) Refresh() (Board, Changed, error) {
	start := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	root := ""
	if r.store != nil {
		root = r.store.Root()
	}

	if !r.scanned {
		if err := r.rescan(); err != nil {
			return Board{
				Health: Health{
					Root:     root,
					Duration: time.Since(start),
					Errs:     1,
				},
			}, Changed{}, err
		}
	}

	repoList := make([]RepoInfo, len(r.repos))
	for i, rs := range r.repos {
		repoList[i] = RepoInfo{Name: rs.name, Path: rs.path}
	}

	b := Board{
		Tasks:    make([]view.Task, 0, len(r.tasks)),
		Repos:    len(r.repos),
		RepoList: repoList,
		ReadAt:   time.Now(),
	}
	b.Errs = append(b.Errs, r.scanErrs...)

	var (
		changed    Changed
		totalBytes int64
		eventsRead int
		lastWrite  time.Time
	)

	for _, st := range r.tasks {
		fresh, err := r.poll(st)
		totalBytes += st.size
		eventsRead += len(fresh)

		if st.modTime.After(lastWrite) {
			lastWrite = st.modTime
		}

		moved := len(fresh) > 0
		// A log that has just started or just stopped being readable is a
		// row that changes even though no event arrived.
		flipped := (err != nil) != (st.err != nil)

		st.err = err
		if err != nil {
			b.Errs = append(b.Errs, &TaskError{Repo: st.repo.name, ID: st.id, Err: err})
		}

		if moved || !st.seen {
			st.events = append(st.events, fresh...)
			st.task = view.Fold(st.events)
		}
		// Where the log was found is the caller's knowledge and not the
		// log's, so Fold leaves these three empty and they are stamped
		// here — on every refresh rather than only on a fold, so that a
		// repository whose name has just become readable reaches the rows
		// it owns without waiting for their logs to move.
		//
		st.task.ID, st.task.Repo, st.task.RepoPath = st.id, st.repo.name, st.repo.path

		// Live is the one field on the row that does not come from the log,
		// and cannot: a run killed with SIGKILL writes nothing on its way
		// out, so a log ending at phase.started is either a run in flight or
		// a run that died on Tuesday. The run marker under the task's
		// directory, and a signal 0 to the pid it names, are the only things
		// that tell those apart, and task.Alive is the very function `orbit
		// reconcile` asks — one reader of that file, not two.
		//
		// It is asked on every refresh, including for a log that has not
		// moved, because a process dying changes no file the poll above
		// would notice. It costs one open that usually fails with ENOENT,
		// which is the same order of cost as the stat the poll is built on.
		//
		// Believing it is as far as this package goes: the band still comes
		// from the record. A run whose process is gone is not abandoned
		// until somebody appends task.abandoned — task.Reconcile's job, and
		// the window's to call once when it opens — because a board that
		// banded on liveness would be the only reader of the record that
		// knew, and `orbit show` and `cat` would still say the run was
		// going. That drift is the thing the record exists to prevent.
		_, alive, aliveErr := task.Alive(r.store, task.Task{ID: st.id, Repo: repo.Repo{Path: st.repo.path}})
		if aliveErr != nil {
			// Not a *TaskError: that type says the task's log could not be
			// read, which drives the flipped test above and would be a lie
			// here — the log is fine, the marker beside it is not. The row
			// stays, showing what the record says, and the fault is
			// reported where a reader can see it.
			b.Errs = append(b.Errs, fmt.Errorf("task %s in %s: %w", st.id, st.repo.name, aliveErr))
		}

		// Three answers out of two returns and an error. A marker that could
		// not be read is not "nothing holds this": it is nobody knowing, and
		// the window refuses both starting and stopping on it rather than
		// guessing which of the two would do less harm.
		switch {
		case aliveErr != nil:
			st.task.Live = view.LiveUnknown
		case alive:
			st.task.Live = view.LiveHeld
		default:
			st.task.Live = view.LiveFree
		}

		// One call to view.BandOf, answering both the crossing and — via
		// counts below, over these very tasks — the number in the header.
		band := view.BandOf(st.task)
		if r.baseline && band == view.NeedsYou && st.band != view.NeedsYou {
			changed.Entered = append(changed.Entered, st.id)
		}

		if moved || flipped || !st.seen {
			changed.Tasks = append(changed.Tasks, st.id)
		}

		st.band, st.seen = band, true
		b.Tasks = append(b.Tasks, st.task)
	}

	b.Counts = counts(b.Tasks)
	b.Health = Health{
		Root:       root,
		Repos:      len(r.repos),
		Tasks:      len(r.tasks),
		Bytes:      totalBytes,
		EventsRead: eventsRead,
		LastWrite:  lastWrite,
		Duration:   time.Since(start),
		Errs:       len(b.Errs),
	}
	r.baseline = true

	return b, changed, nil
}

// Rescan walks the tree again: every repository under the root and every
// task the state root holds against each of them.
//
// It is separate from Refresh, and slower, because a new event is common
// and a new task is rare. What it does that Refresh does not is find and
// forget: a task written down since the window opened gains a row here and
// nowhere else, and one whose directory has gone loses its row here. A task
// that was already known keeps everything Refresh remembers about it — its
// offset above all, so a rescan costs no re-reading.
//
// A failure that concerns one repository is kept for the next board's Errs
// and does not stop the rest of the walk — a repository whose tasks cannot
// be listed included, even when it was the only repository there was. The
// one error returned is the walk not happening at all: a root that is not
// there, or one that cannot be read, as against one that was read and found
// empty. In that case the previous enumeration is left standing rather than
// replaced with nothing.
func (r *Reader) Rescan() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.rescan()
}

// rescan is Rescan with the lock already held, so Refresh can enumerate on
// its first call without taking it twice.
func (r *Reader) rescan() error {
	// The repositories are the ones under the root, and not the ones the
	// state root happens to hold a directory for. Those two are different
	// sets, and the difference is the whole of what a new reader sees: a
	// repository gains a directory under repos/ only when the first task is
	// written against it, so enumerating from there tells somebody who has
	// just cloned three projects that there are no repositories at all, and
	// offers them the one action — clone one — that would change nothing.
	//
	// repo.Discover is the walker, and it is the same one `orbit repos`
	// uses rather than a second: it stops at the first .git instead of
	// descending into it, skips dotted directories and the state root, and
	// does not follow symlinks. A directory that looks like a repository and
	// will not open is left out of the listing rather than failing the walk.
	//
	// The one error is the walk not happening: a root that is not there, or
	// one that cannot be read. An empty board and a root nobody could look
	// in are the same picture, and which one this is has to be said.
	found, err := repo.Discover(r.root)
	if err != nil {
		return fmt.Errorf("look for repositories under %q: %w", r.root, err)
	}

	var errs []error

	repos := make([]*repoState, 0, len(found))
	for _, rp := range found {
		// The name is taken from the walk rather than asked for again.
		// Discover has already run repo.Open on this path, and repo.Open is
		// three git subprocesses; asking a second time would pay for them
		// twice on every enumeration.
		repos = append(repos, &repoState{path: rp.Path, name: rp.Name})
	}
	// Sorted by name so the rows are in an order a reader can predict —
	// repos/ is named by hash and answers nothing on its own — and by path
	// after it, because two checkouts of the same project have one name.
	slices.SortFunc(repos, func(a, b *repoState) int {
		return cmp.Or(strings.Compare(a.name, b.name), strings.Compare(a.path, b.path))
	})

	index := make(map[taskKey]*taskState, len(r.index))

	tasks := make([]*taskState, 0, len(r.tasks))
	for _, rs := range repos {
		ids, listErr := r.list(rs)
		if listErr != nil {
			// The tasks of this one repository are not claimed this time
			// round. Every other repository still walks.
			errs = append(errs, listErr)
			continue
		}

		for _, id := range ids {
			st, keep := r.index[taskKey{rs.path, id}]
			if !keep {
				path, pathErr := r.store.EventsPath(id)
				if pathErr != nil {
					errs = append(errs, pathErr)
					continue
				}

				st = &taskState{id: id, path: path}
			}

			st.repo = rs
			index[taskKey{rs.path, id}] = st
			tasks = append(tasks, st)
		}
	}

	r.repos, r.tasks, r.index, r.scanErrs, r.scanned = repos, tasks, index, errs, true

	return nil
}
