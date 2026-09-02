package board

// What a poll asks: everything the record has gained since the last refresh,
// one task's whole history, and what tasks one repository has. Refresh and
// Rescan are the loops; these are what they call.

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// arrival is one event the stream brought, with the row it was written at.
//
// The row travels with the event because the reader has to be able to tell an
// event it has already been given from one it has not, and two events of a
// task can be told apart by nothing else: the same kind, at the same second,
// with the same text, is what a retried run writes.
type arrival struct {
	at    int64
	event record.Event
}

// follow reads everything written since the last refresh, whatever task it
// was written about, and files it under the tasks it belongs to.
//
// One query, and not one per task. That is what the byte offsets used to
// buy, bought again and more cheaply: a refresh over twenty quiet tasks is
// one statement answering no rows, where it used to be twenty stats.
//
// Each state carries the row it has read up to rather than trusting the
// reader's own, so an event this call has already handed to a task cannot be
// handed to it twice — which is what makes the catch-up below safe to run at
// any moment, against a record another process is writing to.
func (r *Reader) follow() (map[string][]arrival, error) {
	d, err := r.store.Record()
	if err != nil {
		return nil, err
	}

	changes, err := d.Since(r.at, "")
	if err != nil {
		return nil, err
	}

	fresh := make(map[string][]arrival, len(changes))

	for _, c := range changes {
		fresh[c.Task] = append(fresh[c.Task], arrival{at: c.N, event: c.Event})

		if c.N > r.at {
			r.at = c.N
		}
	}

	return fresh, nil
}

// history is everything the record holds about one task the reader has only
// just heard of.
//
// A task written down since the window opened has a whole history behind the
// row every other task has already reached, and the stream above starts
// where they are. So it is read once, from the top, exactly as opening its
// log from byte zero used to do.
func (r *Reader) history(st *taskState) ([]record.Event, error) {
	d, err := r.store.Record()
	if err != nil {
		return nil, err
	}

	changes, err := d.Since(0, st.id)
	if err != nil {
		return nil, fmt.Errorf("read the record of %q: %w", st.id, err)
	}

	events := make([]record.Event, 0, len(changes))

	for _, c := range changes {
		events = append(events, c.Event)

		if c.N > st.at {
			st.at = c.N
		}
	}

	return events, nil
}

// enumerate is every task this board holds, with the state each one carries
// between refreshes.
//
// One query, asked from the task's end rather than the repository's. A task
// worked in four checkouts comes back as four rows and leaves here as one
// state naming four repositories: asked the other way round it would be four
// tasks with the same id, which is how a band came to count pairs.
//
// The index is keyed by the id alone. Ids are the state root's to give out
// now and are unique across it, so a task is one thing whatever number of
// repositories it reaches into.
func (r *Reader) enumerate(repos []*repoState) ([]*taskState, map[string]*taskState, error) {
	d, err := r.store.Record()
	if err != nil {
		return nil, nil, err
	}

	worked, err := d.TasksAndRepos()
	if err != nil {
		return nil, nil, fmt.Errorf("the tasks under %q: %w", r.root, err)
	}

	under := make(map[string]*repoState, len(repos))
	for _, rs := range repos {
		under[rs.path] = rs
	}

	found := make([]*taskState, 0, len(r.tasks))
	index := make(map[string]*taskState, len(r.index))

	for _, w := range worked {
		st, held := index[w.Task]
		if !held {
			st = r.carried(w.Task)
			index[w.Task] = st
			found = append(found, st)
		}

		st.repos = append(st.repos, RepoInfo{Name: w.Name, Path: w.Path})
		// Filed under the first repository it joined that this board can
		// see, which is where its checkout and its diff are opened from.
		// The rest are on the row and nowhere else.
		if st.repo == nil {
			st.repo = under[w.Path]
		}
	}

	return onTheBoard(found, index), index, nil
}

// carried is what the previous enumeration remembered about a task, emptied
// of the repositories it named then.
//
// Everything else is kept — the offset above all, so a rescan costs no
// re-reading. The repositories are not, because a task that has joined one
// since would otherwise name it twice.
func (r *Reader) carried(id string) *taskState {
	st, kept := r.index[id]
	if !kept {
		return &taskState{id: id}
	}

	st.repo, st.repos = nil, nil

	return st
}

// onTheBoard drops the tasks this window is not of and puts the rest in the
// order they are drawn.
//
// A task none of whose repositories is under the root is not on this board.
// The walk that asked each repository what it held said that by never
// asking; asked from the task's end it has to be said out loud, and a task
// dropped here is dropped from the index with it so that nothing reaches it
// by id either.
//
// The order is the repository the task is filed under and then the id, which
// is the order the rows have always been in: predictable, and stable across
// refreshes so that a cursor resting on a row stays on that row.
func onTheBoard(found []*taskState, index map[string]*taskState) []*taskState {
	tasks := make([]*taskState, 0, len(found))

	for _, st := range found {
		if st.repo == nil {
			delete(index, st.id)
			continue
		}

		tasks = append(tasks, st)
	}

	slices.SortFunc(tasks, func(a, b *taskState) int {
		return cmp.Or(
			strings.Compare(a.repo.name, b.repo.name),
			strings.Compare(a.repo.path, b.repo.path),
			strings.Compare(a.id, b.id),
		)
	})

	return tasks
}

// arrivals is what one task gained this refresh: its whole history if this
// is the first the reader has heard of it, and otherwise the part of the
// stream that was written about it.
//
// The row each task has read up to is checked against even on the stream,
// and that is not belt and braces. A task caught up during the previous
// refresh read to the last row in the record, which is past the row the
// stream had reached before it — so the next stream carries rows that task
// has already been given, and taking them again would show a run attempted
// twice over a record that says once.
func (r *Reader) arrivals(st *taskState, arrived map[string][]arrival) ([]record.Event, error) {
	if !st.seen && st.at == 0 {
		return r.history(st)
	}

	var fresh []record.Event

	for _, a := range arrived[st.id] {
		if a.at <= st.at {
			continue
		}

		fresh = append(fresh, a.event)
		st.at = a.at
	}

	return fresh, nil
}

// newest is when the last of these events was written, which is what the
// health panel calls the last write.
//
// It comes from the events themselves rather than from a file's modification
// time, and it is the one number on that panel that is now a fact about the
// record instead of a fact about the filesystem holding it.
func newest(events []record.Event) time.Time {
	var at time.Time

	for _, e := range events {
		if e.At.After(at) {
			at = e.At
		}
	}

	return at
}
