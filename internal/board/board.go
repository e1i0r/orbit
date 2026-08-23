// Package board walks every repository in the state root, folds every
// task's record, and answers what is on screen right now and what changed
// since last time.
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

	"github.com/e1i0r/orbit/internal/record"
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
	// Repos is how many repositories the state root has a readable record
	// of — the number in the header, not the number of rows.
	Repos int
	// Counts is how many tasks are in each band, indexed by the view.Band
	// value. It is filled by calling view.BandOf on the very tasks in
	// Tasks, so the number above a band and the rows inside it are one
	// answer rather than two rules that agree by inspection.
	Counts [4]int
	// ReadAt is when this board was read, so the window can age its elapsed
	// column against one time rather than against time.Now per row.
	ReadAt time.Time
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

// Reader is the window's view of the state root, and the only thing in
// Orbit that remembers where it stopped reading.
//
// It is not a goroutine and it starts none: Refresh and Rescan are called
// by whoever owns a clock, and in the window that is two tea.Cmds.
type Reader struct {
	store *store.Store

	// mu serialises Refresh against Rescan and against itself. Bubble Tea
	// runs every Cmd on a goroutine of its own and the window has two
	// clocks, so nothing above this type keeps the 500 ms poll from
	// overlapping the 2 s enumeration — a Reader that was not safe to call
	// from both would be a data race the day the window is wired up.
	mu sync.Mutex

	repos []*repoState
	tasks []*taskState           // in the order Board.Tasks draws them
	index map[taskKey]*taskState // the same states, to carry offsets across a rescan
	// opened is the repositories repo.Open has already answered for. Only
	// successes are kept; see open.
	opened   map[string]*repoState
	scanErrs []error // what the last enumeration could not do
	scanned  bool    // an enumeration has completed
	// baseline says a Refresh has completed, and it is the whole of the
	// first-refresh-rings-no-bell rule.
	baseline bool
}

// taskKey is how a task is identified between enumerations. It is the
// repository's path and the id together, because an id is unique inside one
// repository and nowhere else.
type taskKey struct{ repoPath, id string }

// NewReader makes a reader of one state root. It touches no disk until it
// is asked to.
func NewReader(s *store.Store) *Reader {
	return &Reader{
		store:  s,
		index:  make(map[taskKey]*taskState),
		opened: make(map[string]*repoState),
	}
}

// repoState is one repository the board knows about.
//
// path is the marker's path and not git's, deliberately. The store files a
// repository's record under a hash of the path it was created with, so that
// path is the key everything under repos/ is reached by; substituting the
// top level git resolves to — which differs whenever the path passes
// through a symlink — would look up a directory that does not exist and
// lose every task in it. The only thing taken from repo.Open is the name.
type repoState struct {
	path string
	name string
}

// taskState is what the Reader remembers about one task between refreshes.
// It is the whole of the polling design: an offset, the size the last stat
// saw, and the events read so far.
type taskState struct {
	repo *repoState
	id   string
	path string // the task's events.jsonl
	// offset is the next byte to read, and it is only ever what
	// record.ReadFrom answered — never arithmetic of this package's own.
	// That matters because ReadFrom advances the offset past a complete,
	// newline-terminated line and no further: a torn final line is a write
	// in flight rather than damage, and holding the offset before it is
	// what makes the next read pick the line up once it lands.
	offset int64
	// size is the file size the last stat saw. An unchanged size is the
	// cheap answer this design is built on: nothing was appended, so there
	// is nothing to open, read or parse.
	size int64
	// events is everything read so far, oldest first. view.Fold is a
	// function of the whole log and holds nothing between calls, so the
	// delta is appended here and the fold re-run over the total — which
	// costs a walk of a slice already in memory, and never a re-read of the
	// file. That is the trade the offsets buy: the I/O and the JSON are
	// what polling has to avoid, not the switch statement.
	events []record.Event
	task   view.Task
	// band is the band at the previous refresh, and it is what Entered is a
	// crossing of. A task the reader has not folded yet has the zero Band,
	// view.ToDo, so a task that arrives already needing you does cross.
	band view.Band
	// err is the last read's verdict, kept so that a log nobody can read
	// keeps saying so on the refreshes that skip it for being unchanged.
	err error
	// seen says a refresh has folded this task at least once. It is what
	// makes a task with an empty log — written down and never run — fold
	// once rather than never.
	seen bool
}
