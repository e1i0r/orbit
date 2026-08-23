package board

// One poll: stat every task's log, read the tail of the ones that grew,
// refold what moved. And, on the slower clock, walk the tree again to find
// what has been written down since.

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
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
// and does not stop the rest of the walk. Only a state root whose
// repositories cannot be listed at all is returned, and in that case the
// previous enumeration is left standing rather than replaced with nothing.
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
	// never the board. An error with no repositories to show for it is a
	// different thing: there is no enumeration to install.
	refs, err := r.store.Repos()
	if err != nil && len(refs) == 0 {
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
