package board

// The two loops, on the two clocks: refresh every task the last enumeration
// found and coalesce what moved into one answer, and, more slowly, walk the
// tree again to find what has been written down since. What either of them
// does to a single task or repository is poll.go.

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/store"
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
// nineteen. The returned error is reserved for a state root whose
// repositories cannot be listed at all, where an empty board would be
// indistinguishable from an empty root.
func (r *Reader) Refresh() (Board, Changed, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.scanned {
		if err := r.rescan(); err != nil {
			return Board{}, Changed{}, err
		}
	}

	b := Board{
		Tasks:  make([]view.Task, 0, len(r.tasks)),
		Repos:  len(r.repos),
		ReadAt: time.Now(),
	}
	b.Errs = append(b.Errs, r.scanErrs...)

	var changed Changed
	for _, st := range r.tasks {
		fresh, err := r.poll(st)
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
		// Live is left exactly as Fold set it. Deciding whether a process
		// still holds a task means reading the run marker and asking the
		// operating system about a pid, and that marker is task 6's; until
		// it exists the record's answer is the only honest one.
		st.task.ID, st.task.Repo, st.task.RepoPath = st.id, st.repo.name, st.repo.path

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
	r.baseline = true
	return b, changed, nil
}

// Rescan walks the tree again: every repository in the state root and every
// task in each of them.
//
// It is separate from Refresh, and slower, because a new event is common
// and a new task is rare. What it does that Refresh does not is find and
// forget: a task written down since the window opened gains a row here and
// nowhere else, and one whose directory has gone loses its row here. A task
// that was already known keeps everything Refresh remembers about it — its
// offset above all, so a rescan costs no re-reading.
//
// A failure that concerns one repository is kept for the next board's Errs
// and does not stop the rest of the walk — a damaged marker included, even
// when it was the only repository there was. The one error returned is the
// listing not happening at all: repos/ itself refusing to be read, as
// against being read and found damaged. In that case the previous
// enumeration is left standing rather than replaced with nothing.
func (r *Reader) Rescan() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rescan()
}

// rescan is Rescan with the lock already held, so Refresh can enumerate on
// its first call without taking it twice.
func (r *Reader) rescan() error {
	// Repos always returns every repository whose marker was readable
	// alongside its error, so a damaged marker costs one entry in Errs and
	// never the board — including when it was the only repository there was,
	// which is why this asks the error what happened rather than counting
	// what came back. An empty list with a fault behind it is the ordinary
	// shape of one damaged marker, and a board that went blank until someone
	// repaired a file would be the fault made worse. Only repos/ itself
	// refusing to be listed leaves nothing to enumerate, and store says so by
	// type.
	refs, err := r.store.Repos()
	var unlistable *store.ReposError
	if errors.As(err, &unlistable) {
		return fmt.Errorf("list the repositories under %q: %w", r.store.Root(), err)
	}
	var errs []error
	if err != nil {
		errs = append(errs, err)
	}

	repos := make([]*repoState, 0, len(refs))
	for _, ref := range refs {
		rs, openErr := r.open(ref)
		if openErr != nil {
			errs = append(errs, openErr)
		}
		repos = append(repos, rs)
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
				path, pathErr := r.store.EventsPath(rs.path, id)
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
