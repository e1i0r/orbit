// Package view folds one task's append-only record into the single state the
// window draws, and answers which of four bands that task belongs to.
//
// Nothing here is remembered between folds. Fold is a function of the events
// it is handed and of nothing else: no cache, no package variable, no init.
// A view that remembered would be a second copy of the record, and two
// copies of anything eventually disagree — which is the entire class of bug
// this package exists to make impossible.
package view

import (
	"strconv"
	"time"
)

// Band is what the reader is being asked for. There are four and every task
// is in exactly one — Bands returns them in the order they are drawn and
// BandOf is the only thing that decides membership.
//
// v1 shipped `Pending 4` above a different four, on a web board and then
// twice more in its console, because the header and the list were two rules
// that agreed by inspection. One predicate, called by both, is the fix.
//
// The predicate is called BandOf rather than Band because Go will not let a
// type, a struct field and a function share one name, and the type and the
// field on Task are both Band.
type Band int

// The four bands, in the order they are drawn.
//
// Two memberships are worth stating here because they are decisions and not
// definitions. A failed run is NeedsYou and not Done: to whoever is reading
// this screen a failure and a question are the same sentence — nothing else
// happens here until you do something. A cancelled run is Done and not
// NeedsYou for the mirror reason: the reader is the one who stopped it, and
// telling somebody that the thing they just cancelled now needs them is how
// a channel stops being read. A timeout is not a cancellation; nobody chose
// it, so it needs you.
const (
	NeedsYou Band = iota // failed, timed out, abandoned, or waiting at a gate
	Running              // a live process holds it, paused included
	ToDo                 // written down and never started
	Done                 // finished or cancelled, and read or not
)

// Bands returns the four bands in the order they are drawn.
//
// It builds a new slice on every call rather than handing out a package
// variable, because a package variable a caller can reorder is package
// state, and this package holds none.
func Bands() []Band {
	return []Band{NeedsYou, Running, ToDo, Done}
}

// String names a band for a test failure or a debug line. The window draws
// translated words through internal/words and never these.
func (b Band) String() string {
	switch b {
	case NeedsYou:
		return "needs you"
	case Running:
		return "running"
	case ToDo:
		return "to do"
	case Done:
		return "done"
	default:
		return "band(" + strconv.Itoa(int(b)) + ")"
	}
}

// state is where the record left a task: the one fact Fold decides and the
// one fact BandOf reads.
//
// It is unexported because it is not something the window has any business
// drawing — the window draws the band, the phase and the reason — and
// because a value only this package can produce is a value only this
// package can get wrong. The zero value is the state of a task that has
// been written down and nothing more, so a Task nobody folded is ToDo.
type state int

const (
	stateNew       state = iota // written down; no run has started
	stateRunning                // a run is between task.started and a terminal event
	stateHeld                   // stopped at a gate because the reader asked
	stateWaiting                // stopped at a gate because the flow asked
	stateFailed                 // task.failed, or a phase.failed with nothing after it
	stateTimedOut               // task.timedout
	stateAbandoned              // task.abandoned
	stateCancelled              // task.cancelled, or a phase.cancelled with nothing after it
	stateFinished               // task.finished

	// stateCount is not a state. It is how many there are, so a test can
	// walk every one and fail when a new state arrives without a band.
	stateCount
)

// Task is everything the window knows about one task. Every field is derived
// from the record; nothing here is remembered between folds.
//
// ID, Repo and RepoPath are the exception and are left empty by Fold: they
// are facts about where the log lives rather than facts inside it, and the
// caller that opened the log is the only thing that knows them.
type Task struct {
	ID       string
	Repo     string // the repository's name, for the column
	RepoPath string // where it is, for the diff tab and $EDITOR
	Title    string // the first line of task.md
	// Band is BandOf(t) as of the fold, stored so a row and a header count
	// read the same value instead of each deciding again. BandOf reads only
	// state, which nothing outside this package can set, so no caller can
	// leave this field disagreeing with the predicate.
	Band    Band
	Flow    string    // the flow it was written against, or the one a run overrode it with
	Phase   string    // the phase now, or the phase it stopped in
	PhaseN  int       // 1-based, from Data["n"]; 0 when the record does not say
	Engine  string    // from Data["engine"]
	Model   string    // from Data["model"]; empty is a fact, not a gap
	Since   time.Time // when the state on screen began; zero if the record never carried an honest timestamp
	Started time.Time // when the current run began; zero if never run
	Reason  Reason    // the word the row needs beyond its phase; zero when there is none
	Attempt int       // how many task.started blocks are in the log
	Live    bool      // a process is believed to hold it; always false out of Fold, and board owns it
	Read    bool      // task.read is in the log
	Cost    float64   // summed from every phase.finished that reported one
	Damaged int       // count of record.unreadable markers
	// state is what BandOf switches on. It is last because it is the only
	// field of this struct that is not something the window draws.
	state state
}

// BandOf answers which band a task is in, and it is the only thing that does.
// The header counts with it and the list draws with it, so the two cannot
// disagree about a task the way v1's did.
//
// It reads state and nothing else. In particular it does not read Live: a run
// whose process is gone is not abandoned until somebody writes task.abandoned
// into the record, and a window that banded on liveness alone would know
// something no other reader of the record could know — `orbit show` and `cat`
// would still say the run was in a phase, and only the screen would say
// otherwise. Reconciling is a supervisor's act and it belongs in a write, not
// in a fold.
//
// It is total: every value, including one no constant here names, comes back
// as one of the four bands, which is what lets the partition hold for any
// Task at all rather than only for a Task this package built.
func BandOf(t Task) Band {
	switch t.state {
	case stateNew:
		return ToDo
	case stateRunning, stateHeld:
		return Running
	case stateWaiting, stateFailed, stateTimedOut, stateAbandoned:
		return NeedsYou
	case stateCancelled, stateFinished:
		return Done
	default:
		// A state this function does not know is a defect in this package,
		// and NeedsYou is where a defect should surface: a task that lands
		// somewhere the reader looks is a task somebody notices, and one
		// filed under Done is one nobody opens again.
		return NeedsYou
	}
}
