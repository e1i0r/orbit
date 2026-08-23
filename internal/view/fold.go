package view

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// The kinds this fold understands. They are string constants here and string
// literals in internal/task, which writes them, because the layering forbids
// either package from importing the other — internal/view may import
// internal/record and nothing else, and that restriction is what keeps the
// window from being able to append an event. The seam is real: a kind
// renamed on the writing side and not here would fold to nothing. It is
// narrow enough to live with, and wide enough to be worth saying out loud.
const (
	kindCreated   = "task.created"
	kindStarted   = "task.started"
	kindFinished  = "task.finished"
	kindFailed    = "task.failed"
	kindCancelled = "task.cancelled"
	kindTimedOut  = "task.timedout"
	kindAbandoned = "task.abandoned"
	kindRead      = "task.read"

	kindPhaseStarted   = "phase.started"
	kindPhaseFinished  = "phase.finished"
	kindPhaseFailed    = "phase.failed"
	kindPhaseCancelled = "phase.cancelled"
	kindPhaseWaiting   = "phase.waiting"
	kindPhaseResumed   = "phase.resumed"
)

// whyPaused is the value phase.waiting carries when the reader asked for the
// stop rather than the flow. It is the difference between a task that needs
// you and a task you are holding, and it is the only thing in the record
// that can tell those two apart.
const whyPaused = "paused"

// Fold walks a task's events, oldest first, and returns the one state the
// window draws.
//
// It cannot fail. Every event is either understood, counted as damage, or
// ignored, and there is no input for which the honest answer is an error —
// a log that could not be read at all never reaches here, because there are
// no events to hand over.
//
// Three fields it leaves empty, and one it leaves false:
//
//   - ID, Repo and RepoPath say where the log was found, which is the
//     caller's knowledge and not the log's. internal/board fills them in.
//   - Live says whether a process is believed to hold the task, and the
//     record cannot answer it: a run killed with SIGKILL writes nothing on
//     its way out, so a log whose last event is phase.started is either a
//     run in flight or a run that died. internal/board decides from the pid
//     file, and where the answer is "it died", the fix is to append
//     task.abandoned rather than to display a private conclusion. So the
//     fold reports what the record says — Running — and leaves the promotion
//     to somebody who can look at a process.
func Fold(events []record.Event) Task {
	var t Task
	for _, e := range events {
		fold(&t, e)
	}
	t.Band = BandOf(t)
	return t
}

// fold applies one event. It is separate from the loop so that the loop is
// one line and this is a flat table of kinds, which is the shape a reader
// checks against the writer.
func fold(t *Task, e record.Event) {
	switch e.Kind {
	case record.Unreadable:
		// A line nobody could parse never becomes state. It is counted, and
		// the count is rendered as something the reader can go and look at
		// with their own eyes. Folding a damaged line into a verdict is
		// exactly the lie the whole record exists to prevent: the one thing
		// worse than not knowing what happened is being told confidently.
		t.Damaged++
	case kindCreated:
		// task.created carries the whole of task.md, and the first line of
		// it is the title. Everything below the first line is the task, and
		// the window has a pane for that.
		t.Title = firstLine(e.Text)
		flow(t, e)
		t.state = stateNew
		stamp(&t.Since, e.At)
	case kindStarted:
		// There is no run identifier anywhere in the record; the boundary
		// between one run and the next is this event. Everything the last
		// attempt left on the row goes with it — a phase from an attempt
		// that is over is a stale phase, and a reason that has been retried
		// is not a reason any more. Cost and Read do not: cost is what the
		// task has spent in total, and task.read is a fact about the log.
		t.Attempt++
		t.state = stateRunning
		t.Reason = Reason{}
		t.Phase, t.PhaseN, t.Engine, t.Model = "", 0, "", ""
		flow(t, e)
		stamp(&t.Started, e.At)
		stamp(&t.Since, e.At)
	case kindPhaseStarted:
		t.Phase = e.Phase
		t.PhaseN = count(e.Data["n"])
		// Assigned rather than merged: each phase declares its own engine
		// and model, so an event that does not name one is a record saying
		// there is none, and carrying the last phase's model forward would
		// put a model on a row that never used it.
		t.Engine = e.Data["engine"]
		t.Model = e.Data["model"]
		t.state = stateRunning
		t.Reason = Reason{}
		stamp(&t.Since, e.At)
	case kindPhaseFinished:
		// Since is deliberately not moved. What the row says does not change
		// when a phase ends — the task is still in the same run, and the
		// elapsed number beside it is how long that run has been in this
		// phase. The next phase.started moves it.
		t.Phase = e.Phase
		t.Cost += money(e.Data["cost"])
	case kindPhaseFailed:
		// phase.failed is where the phase's name is recorded, and the
		// task.failed that follows never carries one. Setting the state here
		// as well as there is what makes a log that ends at phase.failed —
		// which happens when writing the second event is what failed — fold
		// to the same verdict as a log that got both.
		t.Phase = e.Phase
		t.state = stateFailed
		t.Reason = failure(e.Phase)
		stamp(&t.Since, e.At)
	case kindFailed:
		t.state = stateFailed
		t.Reason = failure(t.Phase)
		stamp(&t.Since, e.At)
	case kindPhaseCancelled:
		t.Phase = e.Phase
		t.state = stateCancelled
		t.Reason = Reason{Key: ReasonCancelled}
		stamp(&t.Since, e.At)
	case kindCancelled:
		t.state = stateCancelled
		t.Reason = Reason{Key: ReasonCancelled}
		stamp(&t.Since, e.At)
	case kindTimedOut:
		t.state = stateTimedOut
		t.Reason = Reason{Key: ReasonTimedOut}
		stamp(&t.Since, e.At)
	case kindAbandoned:
		t.state = stateAbandoned
		t.Reason = Reason{Key: ReasonAbandoned}
		stamp(&t.Since, e.At)
	case kindPhaseWaiting:
		t.Phase = e.Phase
		t.state, t.Reason = waiting(e)
		stamp(&t.Since, e.At)
	case kindPhaseResumed:
		t.Phase = e.Phase
		t.state = stateRunning
		t.Reason = Reason{}
		stamp(&t.Since, e.At)
	case kindFinished:
		t.state = stateFinished
		t.Reason = Reason{}
		stamp(&t.Since, e.At)
	case kindRead:
		// Read says task.read is in the log and nothing more. It does not
		// move Since, because being read does not change what the row says
		// about when this task got where it is.
		t.Read = true
	default:
		// A kind this version has never heard of — including the empty kind
		// a bare `{}` line unmarshals to. It is not damage: the record is
		// append-only and older readers are expected to meet newer writers,
		// and a reader that turned an unfamiliar kind into a verdict would
		// be guessing. Damaged counts lines that are not JSON at all, which
		// is a different fact and one somebody can act on.
	}
}

// waiting decides which of the two stops this is. The flow asking a phase to
// wait needs you: nothing moves until you answer. The reader asking for it
// does not — a paused run still holds a worktree and still holds a slot, so
// it stays in Running with a word that says it is held, and reporting the
// operator's own pause as a warning is how a warning channel stops being
// read.
//
// A phase.waiting that does not say why is treated as the flow's, because
// the two mistakes are not equal: a held task shown as needing you is noise,
// and a task that needs you shown as merely held is a task nobody comes back
// to.
func waiting(e record.Event) (state, Reason) {
	args := []Arg{{Name: "phase", Value: e.Phase}}
	if e.Data["why"] == whyPaused {
		return stateHeld, Reason{Key: ReasonHeld, Args: args}
	}
	return stateWaiting, Reason{Key: ReasonGate, Args: args}
}

// failure names the phase a run stopped in, or says the run never got that
// far. internal/task writes task.failed with no Phase on it in every case,
// so the phase here is whatever the fold already knew — and when it knew
// none, the run failed before any phase started.
func failure(phase string) Reason {
	if phase == "" {
		return Reason{Key: ReasonFailedToStart}
	}
	return Reason{Key: ReasonFailed, Args: []Arg{{Name: "phase", Value: phase}}}
}

// flow takes the flow's name from an event that carries one. A missing key
// leaves the last name standing: task.created says what the task was written
// against and task.started says what a run overrode it with, and an event
// that names neither is not an event saying the flow was withdrawn.
func flow(t *Task, e record.Event) {
	if name, ok := e.Data["flow"]; ok {
		t.Flow = name
	}
}

// stamp moves a time only when the event carried an honest one. A zero At is
// a damaged timestamp, and it would draw as an elapsed of half a century;
// the last real time this task had is a better answer than that, and a Task
// that never had one keeps the zero so the window can say so.
func stamp(when *time.Time, at time.Time) {
	if !at.IsZero() {
		*when = at
	}
}

// count reads a 1-based phase number. Anything that is not one — absent,
// misspelled, zero, negative — is 0, which is this package's way of saying
// the record does not know. Guessing 1 would put `1/3` on a row whose record
// never said which phase it was in.
func count(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0
	}
	return n
}

// money reads one phase's cost. A value that will not parse contributes
// nothing rather than poisoning the sum — and NaN and the infinities are
// refused by name, because ParseFloat accepts all three and a single NaN
// would make every total after it NaN for the rest of the log. A negative
// cost is not a discount, it is a damaged field.
func money(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

// firstLine is the title: everything up to the first newline, trimmed. The
// rest of task.md is the task itself and the window has a pane for it.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
