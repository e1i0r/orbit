package task

// Closing the record of a run that is not coming back.

import (
	"errors"
	"os"
	"sort"
	"strconv"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// Reconcile ends the record of a task whose run is gone, and reports whether
// it had to.
//
// A log whose last event is phase.started, with a marker naming a process
// that is not there, is a run that was SIGKILLed or a machine that went
// down. Nothing in the log says so and nothing ever will: the dying process
// wrote no line, which is why the invariant any reader may rely on is the
// weaker, true one — a task's log ends in a terminal event, or a reader
// appends one. This is the reader that appends it.
//
// It is the one place a viewer causes a write to the record, and the defence
// is not that the write is small. The alternative is worse: a window that
// instead *displayed* "abandoned" as a private computation would know
// something no other reader could know — `orbit show` would still say
// phase.started, `cat` would still say phase.started, and the window would
// be the only correct reader of a record everybody can read. That drift is
// what got the two boards of the previous design deleted. Reconciling is a
// supervisor's act, not a view: it lives here beside Run, it is also `orbit
// reconcile` on the command line, and a window calling it is calling the
// subcommand's function like every other gesture. The rule the invariant is
// really protecting — every reader of the record agrees — is kept by
// writing, and broken by not writing.
//
// It is called once per task when a window opens, and never in the render
// path. Rendering must not be able to change what it renders.
//
// Three answers, and only one of them writes anything:
//
//   - No marker, or a marker whose process is answering: false, and the
//     record is untouched. Nothing claims the task, or something still does.
//   - A stale marker over a log that already ends properly: false, and the
//     record is untouched. The claim is swept up, because a false claim left
//     on disk is not tidier than one taken away.
//   - A stale marker over a run still open: task.abandoned, the claim swept
//     up, and true.
func Reconcile(s *store.Store, t Task) (bool, error) {
	pid, alive, err := Alive(s, t)
	if err != nil {
		return false, err
	}

	if pid == 0 || alive {
		return false, nil
	}

	events, err := Events(s, t)
	if err != nil {
		return false, err
	}

	// The run is gone, so a word left for it is for nobody. See forget.
	if err := forget(s, t); err != nil {
		return false, err
	}

	if !inFlight(events) {
		return false, removeMarker(s, t)
	}

	if err := emit(s, t, record.Event{Kind: record.TaskAbandoned}); err != nil {
		return false, err
	}
	// True even if the sweep below fails: the record is what callers act on,
	// and task.abandoned is in it now. The error still travels, because the
	// marker left behind will bring the next reader back here.
	return true, removeMarker(s, t)
}

// inFlight reports whether the log has a run still open — an attempt that
// began and never ended.
//
// It walks the whole log rather than reading the last event, because the
// last event is not always the one that decides. task.read can land after a
// run has finished, and phase.failed can be the last line there is when the
// task.failed after it never got written. What opens a run is task.started;
// what closes it is any of the five ways a run can end.
func inFlight(events []record.Event) bool {
	open := false

	for _, e := range events {
		switch e.Kind {
		case record.TaskStarted:
			open = true
		case record.TaskFinished, record.TaskFailed, record.TaskCancelled,
			record.TaskTimedOut, record.TaskAbandoned, record.TaskStuck,
			record.TaskOverBudget, record.TaskOverDiff, record.TaskNewDependency,
			record.TaskContradicts,
			// A run that ran out and had nowhere to go is over. Left off
			// this list it reads as a run still in flight, and reconcile
			// writes task.abandoned over a task that is only waiting for
			// an allowance to come back.
			record.TaskNeedsEngine, record.TaskNoEngine,
			// sendBack writes this one and the run returns behind it, so a
			// requeued task is not a run still in flight. internal/db's
			// closesRun reads the same list.
			record.TaskRequeued:
			open = false
		}
	}

	return open
}

// errCarrierGone is why a delivery closed here broke.
var errCarrierGone = errors.New("the window carrying it closed before it came back")

// ReconcileDeliveries answers, as broken, every delivery verb still open on
// a task whose carrier is gone, and reports how many it closed.
//
// A delivery is carried by the window it was asked in: the supervisor runs
// inside it, and so does the command a key starts. When that window closes,
// what it was carrying dies with it and nothing writes the answer, so the
// tree said "in progress" for as long as anybody looked. Reconcile does the
// same for a run; this is the other half, called beside it when a window
// opens, for the reason it gives for writing rather than displaying.
//
// An ask with no pid on it was written before asks carried one, by a window
// that is not this one, and is taken as gone.
func ReconcileDeliveries(s *store.Store, t Task) (int, error) {
	events, err := Events(s, t)
	if err != nil {
		return 0, err
	}

	open := map[string]record.Event{}

	for _, e := range events {
		switch e.Kind {
		case record.DeliverAsked:
			open[e.Data["verb"]] = e
		case record.DeliverAnswered:
			delete(open, e.Data["verb"])
		}
	}

	verbs := make([]string, 0, len(open))
	for verb, ask := range open {
		if !carried(ask) {
			verbs = append(verbs, verb)
		}
	}

	sort.Strings(verbs)

	for i, verb := range verbs {
		if err := Delivered(s, t, verb, "", errCarrierGone); err != nil {
			return i, err
		}
	}

	return len(verbs), nil
}

// carried is whether the window an ask was made in is still there.
func carried(ask record.Event) bool {
	pid, err := strconv.Atoi(ask.Data["pid"])
	if err != nil || pid <= 0 {
		return false
	}

	return pid == os.Getpid() || running(pid)
}
