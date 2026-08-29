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

// The four bands.
//
// ToDo is first so that it is the zero value: a Task nobody has folded is a
// task written down and nothing more, which is exactly what ToDo means. Any
// other order here would make an empty Task claim something it has not done.
// The order the bands are *drawn* in is Bands's and not this block's, so this
// one is free to be chosen for the zero value.
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
	ToDo     Band = iota // written down and never started
	NeedsYou             // failed, timed out, abandoned, or waiting at a gate
	Running              // a live process holds it, paused included
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

// state is where the record left a task: this package's working vocabulary
// while it walks a log, and the thing Fold converts into a Band once at the
// end.
//
// It is unexported because it is not something the window has any business
// drawing — the window draws the band, the phase and the reason. It decides
// nothing on its own: bandOfState is read exactly once, by Fold, and what
// leaves this package is the Band it produced. A caller who cannot see this
// field is a caller who cannot be surprised by it.
type state int

const (
	stateNew         state = iota // written down; no run has started
	stateRunning                  // a run is between task.started and a terminal event
	stateHeld                     // stopped at a gate because the reader asked
	stateWaiting                  // stopped at a gate because the flow asked
	statePhaseFailed              // a phase failed and the task-level event has not arrived
	stateFailed                   // the run stopped and task.failed says so
	stateTimedOut                 // task.timedout
	stateAbandoned                // task.abandoned
	stateCancelled                // task.cancelled, or a phase.cancelled with nothing after it
	stateFinished                 // task.finished

	// stateCount is not a state. It is how many there are, so a test can
	// walk every one and fail when a new state arrives without a band.
	stateCount
)

// ActionKind says whether the action a row is showing is the model thinking
// or the model reaching for a tool.
//
// It is here rather than drawn into CurrentAction because this package holds
// no presentation: what a reader sees in front of the action is the business
// of whoever draws it, and a caller that is not drawing at all — the MCP
// server sends this field over the wire — gets the action and nothing else.
type ActionKind string

// The two kinds of action, and the absence of one.
const (
	ActionNone     ActionKind = ""
	ActionThinking ActionKind = "thinking"
	ActionTool     ActionKind = "tool"
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
	// Band is the band this task is drawn in. Fold writes it from the
	// record and BandOf reads it, so the header counts and the list draws
	// one value rather than two rules that agree by inspection. A Task
	// built by hand — a fixture, a golden, a row a test types out — is
	// answered as the band it was given, which is the whole reason this is
	// an exported field and not a private conclusion.
	Band   Band
	Flow   string // the flow it was written against, or the one a run overrode it with
	Phase  string // the phase now, or the phase it stopped in
	PhaseN int    // 1-based, from Data["n"]; 0 when the record does not say
	Engine string // from Data["engine"]
	Model  string // from Data["model"]; empty is a fact, not a gap
	// Since is when the state on screen began, as the record tells it, and
	// zero if the record never carried an honest timestamp. It is the
	// record's word and not a checked one: a log whose clock went backwards
	// can put Since before Started, so a window subtracting the two clamps
	// at zero rather than drawing a negative age. The fold does not clamp,
	// because inventing a time is how a record stops being evidence.
	Since time.Time
	// Started is when the current run began, and zero if the record never
	// carried an honest timestamp for it — which is not the same as never
	// having run. Attempt is the has-it-run signal: Attempt > 0 means a
	// task.started is in the log, whatever its clock said.
	Started       time.Time
	Reason        Reason  // the word the row needs beyond its phase; zero when there is none
	Attempt       int     // how many task.started blocks are in the log
	Live          bool    // a process is believed to hold it; always false out of Fold, and board owns it
	Read          bool    // task.read is in the log
	Cost          float64 // summed from every phase.finished that reported one
	Damaged       int     // count of record.unreadable markers
	CurrentAction string  // formatted live action or tool call currently running
	// ActionKind is which of the two CurrentAction is, so that a caller can
	// mark it as one or the other. The mark used to be part of the string —
	// a brain or a hammer glued to the front of it in Fold — which put a
	// glyph inside a field the MCP server hands to a model as
	// current_action, and inside the detail pane's action line, which has
	// the word "thinking" written under it already.
	ActionKind     ActionKind
	CurrentThought string // latest live thinking block from the model
	ToolCallCount  int    // total tool calls invoked in the current attempt
	// state is what Fold folds into and what bandOfState reads. It is last
	// because it is the only field of this struct that is not something the
	// window draws, and it leaves this package only as the Band above.
	state state
}

// BandOf answers which band a task is in, and it is the only thing that does.
// The header counts with it and the list draws with it, so the two cannot
// disagree about a task the way v1's did.
//
// It reads Task.Band. That is the point: the value the row draws and the
// value the header counts are one value. An earlier shape of this function
// derived the answer from an unexported field instead, and it was wrong in
// precisely the way this package exists to prevent — a Task built by hand as
// `Task{Band: Running}` was counted as ToDo, because nothing outside this
// package can set the field the predicate was reading. A golden fixture
// written that way would have enshrined v1's bug in a test file.
//
// It does not read Live: a run whose process is gone is not abandoned until
// somebody writes task.abandoned into the record, and a window that banded on
// liveness alone would know something no other reader of the record could
// know — `orbit show` and `cat` would still say the run was in a phase, and
// only the screen would say otherwise. Reconciling is a supervisor's act and
// it belongs in a write, not in a fold.
//
// It is total. A Band no constant names — a fixture with an arithmetic slip
// in it, a value read from a file somebody edited — comes back as NeedsYou
// rather than as itself, so every task lands in one of the four lists the
// window actually draws. NeedsYou is where a defect should surface: a task in
// front of the reader is one somebody notices, and one filed under Done is
// one nobody opens again.
func BandOf(t Task) Band {
	switch t.Band {
	case ToDo, NeedsYou, Running, Done:
		return t.Band
	default:
		return NeedsYou
	}
}

// bandOfState maps where the record left a task to the band it is drawn in.
//
// Fold is its only caller, and that is the whole design: the state is this
// package's private vocabulary, the band is what leaves, and the conversion
// happens once. Adding a state without adding it here is caught by
// TestEveryStateHasABand rather than by a reader wondering why a task is in
// the wrong list.
func bandOfState(s state) Band {
	switch s {
	case stateNew:
		return ToDo
	case stateRunning, stateHeld:
		return Running
	case stateWaiting, statePhaseFailed, stateFailed, stateTimedOut, stateAbandoned:
		return NeedsYou
	case stateCancelled, stateFinished:
		return Done
	default:
		return NeedsYou
	}
}

// inAttempt reports whether the fold is inside a run that had started: one
// working, one stopped at a gate, or one whose phase has just failed and
// whose task-level event has not arrived yet.
//
// It exists because internal/task writes one task.failed for two different
// things and puts no phase on either — task-level events name no phase,
// phase-level events do. A run that died in a phase and a run that never
// reached one produce the same event, and the only thing that tells them
// apart is where the fold already was when it arrived. Inside an attempt,
// the phase the fold is holding is this run's. Outside one — before the
// first run, or after a previous one ended — the phase it is holding
// belongs to something that is over.
func inAttempt(s state) bool {
	switch s {
	case stateRunning, stateHeld, stateWaiting, statePhaseFailed:
		return true
	default:
		return false
	}
}
