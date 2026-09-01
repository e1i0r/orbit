package ui

// The sentences a refused verb says, and the keys they are catalogued under.
//
// They live beside the rules rather than inside them for one reason: every
// one of these keys has to exist in lang/es.json with the English at this
// call site as its source, and internal/words fails the build when it does
// not. Keeping them in one switch is what makes that checkable by reading a
// single screen.

import "github.com/e1i0r/orbit/internal/words"

// The reason keys, named why.<verb>_<condition>.
//
// A separate key per condition, rather than one sentence per verb with the
// condition substituted in, because the difference between these sentences
// is what the reader should press next — and because a state word
// substituted into a translated sentence is one English word in the middle
// of a Spanish one.
const (
	whyPauseNotRunning         = "why.pause_not_running"
	whyPauseAlreadyHeld        = "why.pause_already_held"
	whyPauseAlreadyWaiting     = "why.pause_already_waiting"
	whyPauseAutopilotIsLifting = "why.pause_autopilot_is_lifting"

	whyResumeStillRunning = "why.resume_still_running"
	whyResumeNotRunning   = "why.resume_not_running"

	whyCancelNotRunning = "why.cancel_not_running"

	whyRequeueAlreadyToDo = "why.requeue_already_todo"

	whyTakeNeverRun               = "why.take_never_run"
	whyTakeStillRunning           = "why.take_still_running"
	whyTakeEngineCannotResume     = "why.take_engine_cannot_resume"
	whyHandBackNotTaken           = "why.hand_back_not_taken"
	whyHandBackNotStopped         = "why.hand_back_not_stopped"
	whyHandBackEngineCannotResume = "why.hand_back_engine_cannot_resume"

	whyAskNotBuilt = "why.ask_not_built"

	whyReadNotFinished = "why.read_not_finished"
	whyReadAlreadyRead = "why.read_already_read"

	whyDeleteRunning = "why.delete_running"

	// One key for every verb that meets an unreadable run marker. The rule
	// above is a key per condition and not per verb, and here the condition
	// is one: nobody knows what holds this task. Five sentences would be
	// five ways of saying the same thing to a reader whose next move is the
	// same in all five.
	whyMarkerUnreadable = "why.marker_unreadable"
)

// engineArg names the one placeholder any of these sentences uses.
const engineArg = "engine"

// Why is the sentence for a refused affordance, in the reader's language,
// and the empty string for one that is offered.
//
// The keys are written out as literals in the T calls rather than passed as
// the constants above, because internal/words verifies a key against
// es.json statically and can only do that for a literal. The duplication is
// held together by that same check: a typo makes the key unknown to es.json
// and fails the build, and a key swapped with another fails on the source
// text not matching the English beside it.
func (a Affordance) Why(p *words.Printer) string {
	switch a.WhyNot.Name {
	case whyPauseNotRunning:
		return p.T("why.pause_not_running", "pausing needs a running task; nothing is running here")
	case whyPauseAlreadyHeld:
		return p.T("why.pause_already_held", "this task is already paused; press r to let it go")
	case whyPauseAlreadyWaiting:
		return p.T("why.pause_already_waiting", "this phase is already waiting for you; press r to let it go")
	case whyPauseAutopilotIsLifting:
		return p.T("why.pause_autopilot_is_lifting", "autopilot is lifting this gate; press A to turn it off")
	case whyResumeStillRunning:
		return p.T("why.resume_still_running", "resuming needs a paused task; this one is running")
	case whyResumeNotRunning:
		return p.T("why.resume_not_running", "resuming needs a paused task; nothing is running here")
	case whyCancelNotRunning:
		return p.T("why.cancel_not_running", "cancelling needs a running task; nothing is running here")
	case whyRequeueAlreadyToDo:
		return p.T("why.requeue_already_todo", "this task is already waiting in to do")
	case whyTakeNeverRun:
		return p.T("why.take_never_run", "taking the keyboard needs a session; this task has never run")
	case whyTakeStillRunning:
		return p.T("why.take_still_running", "a phase is writing in this worktree; press p to stop it, then take the keyboard")
	case whyTakeEngineCannotResume:
		return p.T("why.take_engine_cannot_resume", "{engine} cannot resume a session, so there is nothing to take", a.engine())
	case whyHandBackNotTaken:
		return p.T("why.hand_back_not_taken", "nobody took the keyboard here; press t to take it, or r to let this task go")
	case whyHandBackNotStopped:
		return p.T("why.hand_back_not_stopped", "handing the keyboard back needs a run stopped at a phase; this one is not stopped")
	case whyHandBackEngineCannotResume:
		return p.T("why.hand_back_engine_cannot_resume", "{engine} cannot resume a session, so nothing was taken", a.engine())
	case whyAskNotBuilt:
		return p.T("why.ask_not_built", "orbit cannot ask an engine a question yet; take the keyboard with t and ask it there")
	case whyReadNotFinished:
		return p.T("why.read_not_finished", "marking read needs a finished task; this one is not finished")
	case whyReadAlreadyRead:
		return p.T("why.read_already_read", "this task is already marked read")
	case whyDeleteRunning:
		return p.T("why.delete_running", "cannot delete a running task; cancel it first")
	case whyMarkerUnreadable:
		return p.T("why.marker_unreadable", "orbit cannot read this task's run marker, so it cannot tell whether a phase is running; look at the run file in the task's directory")
	}

	return ""
}

// engine is the value the two engine sentences name. It is built here rather
// than stored on the Affordance so that WhyNot stays what it is everywhere
// else: a key and the one value that key's sentence needs, with the
// placeholder's name kept beside the sentence that uses it.
func (a Affordance) engine() words.Arg {
	return words.Arg{Name: engineArg, Value: a.WhyNot.Value}
}
