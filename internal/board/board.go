// Package board walks every repository under the directory a window was
// opened over, folds every task's record, and answers what is on screen
// right now and what changed since last time.
//
// It sits between the readers and the window. Everything it hands out is
// derived from the append-only record: it appends nothing and decides
// nothing the record does not already say. The one thing it remembers is
// where in each log it stopped reading, and that is an optimisation rather
// than a second copy of the truth — throw a Reader away and the next one
// reaches the same board by reading every log again from byte zero.
//
// # The window polls, and reads only the tail
//
// This is the design decision the package exists to hold, and it is written
// down here so that a later reader does not "fix" it by reaching for a
// filesystem watcher.
//
// An append-only file is the one case where polling is nearly free. Refresh
// calls os.Stat on each task's events.jsonl; a file whose size is unchanged
// costs one stat and nothing else. A file that grew costs a read from the
// stored offset and a parse of exactly the new bytes. At twenty running
// tasks and a 500 ms tick that is forty stats a second and a parse of
// whatever was actually written — where calling record.Read on the same
// twenty tasks would open and scan every log from byte zero forty times a
// second, which at a 200 KB log is eight megabytes a second of parsing to
// learn that nothing happened.
//
// fsnotify is refused for four reasons, and none of them is that it would
// not work: it is a dependency the allowlist would have to admit, it has
// documented deadlocks around removing watches from the consuming
// goroutine, it behaves inconsistently across macOS and Linux for appends,
// and — decisively — it gives no ordering guarantee against the writer's
// fsync, so offset tracking would still be required. It buys latency nobody
// can perceive in a terminal and pays for it in go.mod.
package board

import (
	"fmt"
	"sync"
	"time"

	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/view"
)

// The two clocks. A new event is common and a new task is rare, so events
// are polled at RefreshEvery and the tree is re-walked at RescanEvery;
// re-walking repos/*/tasks/* every 500 ms would be the one genuinely
// wasteful thing in this design. The window owns the tickers — nothing in
// this package starts a goroutine — and these are the numbers it ticks at,
// stated here because this is where the reasoning for them lives.
const (
	RefreshEvery = 500 * time.Millisecond
	RescanEvery  = 2 * time.Second
)

// Board is one answer to "what is on screen right now".
//
// It is a value and not a handle: the window keeps the one it was given
// until the next one arrives, and nothing here changes underneath it.
type Board struct {
	// Tasks is every task in every repository, ordered by repository name
	// and then by id. The order is stable across refreshes so that a cursor
	// resting on a row stays on that row.
	Tasks []view.Task
	// Repos is how many repositories were found under the root this reader
	// was opened over — the number in the header, not the number of rows.
	//
	// It counts a repository nobody has written a task against yet, and
	// that is the point of it: the count comes from the walk and the rows
	// come from the record, so a person who has just cloned three projects
	// is told there are three and no tasks in them, rather than that there
	// are no repositories at all.
	Repos int
	// Counts is how many tasks are in each band, indexed by the view.Band
	// value. It is filled by calling view.BandOf on the very tasks in
	// Tasks, so the number above a band and the rows inside it are one
	// answer rather than two rules that agree by inspection.
	Counts [4]int
	// ReadAt is when this board was read, so the window can age its elapsed
	// column against one time rather than against time.Now per row.
	ReadAt time.Time
	// Health is the status of the .jsonl database and store, measured during
	// the refresh.
	Health Health
	// Errs is what went wrong, per task and per repository, without the
	// board failing as a whole. One task whose log is unreadable must not
	// blank the other nineteen: it is a row that says the record could not
	// be read, which is a different sentence from "no events". A read
	// failure is a *TaskError so the window can find the row it belongs to
	// without matching on the words of a message.
	Errs []error
}

// Changed is what moved since the previous Refresh, coalesced into one
// answer. It is deliberately not one message per event: Bubble Tea's
// message channel is unbuffered and its loop is serial, and one message per
// record is how a backlog is built.
type Changed struct {
	// Tasks is the id of every task the window has reason to redraw: one
	// whose record grew, one the reader is seeing for the first time, and
	// one whose log started or stopped being readable. A task whose log did
	// not move is not in it, which is the whole point of the offsets.
	Tasks []string
	// Entered is the id of every task that crossed *into* view.NeedsYou
	// since the previous refresh, and it is the only thing that notifies.
	// The first refresh leaves it empty whatever it found: opening the
	// window on twelve historic failures must ring no bells, because a
	// notification channel that cries wolf on its first use is never
	// trusted again.
	Entered []string
}

// TaskError is one task whose record could not be read.
//
// It is a type rather than a formatted string because the window has to
// find the row it belongs to, and telling two failures apart by the words
// of their messages is a test of the message rather than of what happened.
type TaskError struct {
	Repo string // the repository's name, as the row shows it
	ID   string // the task's id
	Err  error
}

// Error says which task's record could not be read, and why.
func (e *TaskError) Error() string {
	return fmt.Sprintf("the record of task %s in %s could not be read: %s", e.ID, e.Repo, e.Err.Error())
}

// Unwrap gives up the cause, so errors.Is reaches os.ErrNotExist and the
// rest of what the readers return.
func (e *TaskError) Unwrap() error { return e.Err }

// Unread counts the finished work nobody has looked at, which is the number
// task.Start refuses at.
//
// It lives here rather than in internal/task because it is a question about
// the whole board and this package already imports internal/task: a count on
// the other side would be an import cycle, and — worse — a second fold of the
// record beside internal/view's. This one walks a Board that has already been
// folded, so the number and the rows on screen cannot disagree.
//
// It moved here from internal/cli when the window needed it: internal/ui
// draws the number in its header and cannot import internal/cli, and the one
// thing worse than a count in the wrong package is two counts in two.
//
// Cancelled is not unread. Its band is Done because the reader is the one
// who stopped it, and asking somebody to acknowledge the thing they just
// cancelled is how a brake earns its reputation for being in the way. A
// failed run is not counted either, and for a different reason: it is not in
// Done at all — internal/view bands it as NeedsYou, where it is already in
// front of the reader without a counter's help.
func Unread(b Board) int { return len(Unreads(b)) }

// Unreads is the same answer as a list, in the board's own order.
//
// It exists because a brake that says only "no" is a brake people route
// around: the window refuses to start a task at the cap and names the tasks
// that are waiting, and naming them needs the tasks rather than the count.
// Unread is defined as its length so that the number in the header and the
// ids in the refusal can never come from two different rules — which is the
// whole reason the count lives in this package at all.
func Unreads(b Board) []view.Task {
	var waiting []view.Task
	for _, t := range b.Tasks {
		if view.BandOf(t) != view.Done {
			continue
		}
		if t.Read || t.Reason.Key == view.ReasonCancelled {
			continue
		}
		waiting = append(waiting, t)
	}
	return waiting
}

// counts tallies the bands, and it is the only thing that does.
//
// view.BandOf is total — a Band no constant names comes back as NeedsYou —
// so every answer is one of the four and indexing a [4]int needs no guard.
func counts(tasks []view.Task) [4]int {
	var n [4]int
	for _, t := range tasks {
		n[view.BandOf(t)]++
	}
	return n
}

// Reader is the window's view of one directory and of what the state root
// holds against it, and the only thing in Orbit that remembers where it
// stopped reading.
//
// It is not a goroutine and it starts none: Refresh and Rescan are called
// by whoever owns a clock, and in the window that is two tea.Cmds.
type Reader struct {
	store *store.Store

	// root is the directory this window was opened over, and it is not the
	// state root: the state root says what has been written down, and this
	// says what is there to write against. See NewReader.
	root string

	// mu serialises Refresh against Rescan and against itself. Bubble Tea
	// runs every Cmd on a goroutine of its own and the window has two
	// clocks, so nothing above this type keeps the 500 ms poll from
	// overlapping the 2 s enumeration — a Reader that was not safe to call
	// from both would be a data race the day the window is wired up.
	mu sync.Mutex

	repos    []*repoState
	tasks    []*taskState           // in the order Board.Tasks draws them
	index    map[taskKey]*taskState // the same states, to carry offsets across a rescan
	scanErrs []error                // what the last enumeration could not do
	scanned  bool                   // an enumeration has completed
	// baseline says a Refresh has completed, and it is the whole of the
	// first-refresh-rings-no-bell rule.
	baseline bool
}

// taskKey is how a task is identified between enumerations. It is the
// repository's path and the id together, because an id is unique inside one
// repository and nowhere else.
type taskKey struct{ repoPath, id string }

// NewReader makes a reader of one directory, folded against one state root.
// It touches no disk until it is asked to.
//
// root is the directory `orbit top` was pointed at, and it is what decides
// which repositories are on the board; the store is then asked what has been
// written against each of them. They are separate arguments because they are
// separate places — the record lives under $ORBIT_HOME and the checkouts do
// not — and it is a parameter rather than a field set afterwards because a
// Reader with no root cannot answer the first question it is asked. A zero
// value meaning "nowhere" is a constructor that can be called wrong.
func NewReader(s *store.Store, root string) *Reader {
	return &Reader{
		store: s,
		root:  root,
		index: make(map[taskKey]*taskState),
	}
}

