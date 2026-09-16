package view

// The two ways a run ends badly, and why they are two.
//
// A machine that was switched off is an accident. Running out of quota is
// expected, frequent and recoverable — and a reader told only that something
// broke has to open the log to find out which of the two it was, when the
// two send them to do opposite things: one is a bug to look at, the other is
// a wait or another engine.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/record"
)

// spaced is a comma-separated list of names as a person reads one.
//
// The record writes them tight because a field of a log is a field of a log;
// a row on a board is a sentence, and "claude,codex,opencode" is the seam
// where a reader can see the machine.
func spaced(list string) string {
	return strings.Join(strings.Split(list, ","), ", ")
}

// ranOut folds the ending a phase gets when the allowance is gone.
func ranOut(t *Task, e record.Event) {
	// What a phase spent it spent, whether it got anywhere or not —
	// the same reason phase.failed adds its cost above.
	//
	// The state is its own for the reason statePhaseFailed is: the
	// task.failed that follows carries no phase, and the fold has to be
	// able to tell "the phase I am holding is this run's" from "the
	// phase I am holding is over". And the reason is its own because
	// this is the whole point of the ending: "it ran out" and "it
	// broke" are read by somebody deciding what to do next.
	t.Phase = e.Phase
	t.Cost += money(e.Data["cost"])
	t.state = stateRanOut
	t.Reason = Reason{Key: ReasonRanOut, Args: []Arg{
		{Name: "phase", Value: e.Phase},
		{Name: "engine", Value: t.Engine},
	}}
	stamp(&t.Since, e.At)
}

// denied folds the ending a phase gets when it was refused what it needed
// and left nothing behind.
//
// Read by its exit code alone this was a success, which is how a task that
// did nothing came to sit in done. The state is its own for the reason
// stateRanOut is: the task.failed that follows carries no phase, and what a
// reader does about this is different from both of the others.
func denied(t *Task, e record.Event) {
	t.Phase = e.Phase
	t.Cost += money(e.Data["cost"])
	t.state = stateDenied
	t.Reason = Reason{Key: ReasonDenied, Args: []Arg{
		{Name: "phase", Value: e.Phase},
		{Name: "tool", Value: e.Data["tool"]},
	}}
	stamp(&t.Since, e.At)
}

// failedRun folds the event internal/task writes for every way a run ends
// badly, which is why where the fold already is decides what it means.
func failedRun(t *Task, e record.Event) {
	// One event, two situations, and internal/task writes the same
	// thing for both from one function. Where the fold already is
	// decides which this is.
	if t.state == stateRanOut || t.state == stateDenied {
		// The phase already said why, and this event does not know:
		// internal/task writes one task.failed for every way a run
		// ends badly, so reading it as a failure here would throw away
		// the one thing a reader needs — that nothing is broken and
		// the work is waiting on an allowance.
		stamp(&t.Since, e.At)

		return
	}

	if inAttempt(t.state) {
		// A run was under way, so the phase it is holding is the phase
		// it died in — phase.failed put it there a moment ago.
		t.state = stateFailed
		t.Reason = failure(t.Phase)
	} else {
		// Nothing was under way, so this event landed on the end of a
		// log whose last phase belongs to an attempt that is over.
		// Naming that phase would tell the reader a run died in review
		// when it never started, so the phase goes with the attempt it
		// belonged to.
		//
		// Since internal/task began writing task.started first, a
		// refused re-run reaches here with the phase already cleared and
		// this branch changes nothing. It still matters for two logs:
		// one written before that ordering changed, and one where the
		// run that opened the attempt ended without a task-level event
		// of its own — a phase.failed whose task.failed never got
		// written, and then a re-run.
		t.Phase, t.PhaseN, t.Engine, t.Model = "", 0, "", ""
		t.state = stateFailed
		t.Reason = Reason{Key: ReasonFailedToStart}
	}

	stamp(&t.Since, e.At)
}
