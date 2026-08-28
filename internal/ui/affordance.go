package ui

// What can be done to the thing under the cursor. The key bar, the help
// overlay and the task menu are all shortlists of one answer, computed here.

import (
	"charm.land/bubbles/v2/key"

	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// Conditions is the standing state a verb's answer depends on, as this package
// needs it.
//
// It is declared here, where it is consumed, and it is deliberately not
// store.Settings: the window cannot import internal/store, and a struct that
// carried the reader's language and unread cap into a function about verbs
// would invite exactly the coupling arch.layers exists to prevent. The
// caller converts, from the persisted settings and from what the configured
// engine can do.
//
// It is called Conditions and not Settings because the window's port to the
// settings file took that name, and the two are different things: Settings
// is where the answers are read and written, Conditions is the snapshot the
// verbs are asked about.
type Conditions struct {
	Autopilot bool // the switch; a paused task is still the reader's to lift

	// CanResume is whether the engine *this task* ran under can carry on a
	// session it started before. It is per-task, not per-program: the
	// refusal it produces names an engine, and a standing answer made that
	// sentence name whichever engine the task in front of the reader
	// happened to use rather than the one that could not resume.
	CanResume bool

	// Taken is whether the caller handed the terminal to an engine for this
	// task and has not handed it back. It is here rather than on view.Task
	// because nothing in the record says it — the argument, and what it
	// costs, is at took in gesture.go.
	Taken bool
}

// Affordance is one verb and whether it can be done to the task under the
// cursor.
//
// WhyNot is the catalogue key for the sentence that says why not, in
// WhyNot.Name, and the one value that sentence names, in WhyNot.Value —
// which today is only ever an engine's name. It is empty for a verb that is
// offered. Read it through Why rather than looking the key up yourself; that
// is where the English lives.
//
// A refusal without a reason is the failure this type is shaped to prevent:
// the key bar hides a refused verb and the menu shows it greyed, and a
// greyed entry with nothing beside it tells a reader they have done
// something wrong without telling them what.
type Affordance struct {
	Key    key.Binding
	OK     bool
	WhyNot words.Arg
}

// Affordances answers what can be done to one task, as the same list of
// verbs in the same order every time.
//
// The order is fixed on purpose: the menu is something a reader reaches for
// rather than reads, and a menu whose entries move about between one task
// and the next has to be read every time. What changes from task to task is
// the answers, not the list.
//
// It is a method on Keys rather than a free function because an affordance
// carries the binding it is about — the bar prints the key, the menu prints
// the key and its help — and the bindings are translated, so the alternative
// was a package variable holding a key map, which is the state this package
// does not have.
//
// Two questions are answered here and they are different: "what should I
// press", which is the offered verbs, and "what can I do with this one",
// which is the whole list with its reasons. The key bar asks the first, the
// menu asks the second.
func (k Keys) Affordances(t view.Task, s Conditions) []Affordance {
	return []Affordance{
		// Open is the one verb that is never refused: every task can be
		// opened, including one that has never run — that is where its
		// description is.
		{Key: k.Open, OK: true},
		answer(k.Pause, whyNotPause(t, s)),
		answer(k.Resume, whyNotResume(t)),
		answer(k.Cancel, whyNotCancel(t)),
		answer(k.Take, whyNotTake(t, s)),
		answer(k.Hand, whyNotHand(t, s)),
		// Ask is listed and refused. Orbit has no way to put a question to
		// an engine yet, and the honest thing is to say so in the same
		// voice it uses about an engine that cannot resume a session —
		// rather than leaving a gap the reader discovers by pressing a key
		// that does nothing.
		answer(k.Ask, because(whyAskNotBuilt)),
		answer(k.MarkRead, whyNotMarkRead(t)),
		answer(k.Delete, whyNotDelete(t)),
	}
}

// answer pairs a binding with the reason it is refused, if it is. An empty
// reason is what "offered" means, so the two can never disagree.
func answer(b key.Binding, why words.Arg) Affordance {
	return Affordance{Key: b, OK: why.Name == "", WhyNot: why}
}

// because names a reason that needs no value. The parameter is not called
// key, which would shadow the package of that name for the length of the
// function — harmless here and a trap for whoever edits it next.
func because(name string) words.Arg {
	return words.Arg{Name: name}
}

// about names a reason and the one value its sentence uses.
func about(name, value string) words.Arg {
	return words.Arg{Name: name, Value: value}
}

// parked is a live run stopped at a phase boundary — held because the reader
// asked, or waiting because the flow asked. internal/view keeps the two
// apart because the row says different things about them; here they are one
// state, because r is what lets go of either.
func parked(t view.Task) bool {
	return t.Live && (t.Reason.Key == view.ReasonHeld || t.Reason.Key == view.ReasonGate)
}

// working is a run inside a phase right now, as opposed to one stopped at
// the boundary of one.
func working(t view.Task) bool {
	return t.Live && !parked(t)
}

// started is whether a run has ever begun. Attempt is the signal and not
// Started, which is a timestamp a record with a bad clock can leave at zero
// for a run that certainly happened.
func started(t view.Task) bool {
	return t.Attempt > 0
}

// whyNotPause refuses in four different ways, and the differences are the
// point: each one names something else to press.
func whyNotPause(t view.Task, s Conditions) words.Arg {
	switch {
	case !t.Live:
		return because(whyPauseNotRunning)
	case t.Reason.Key == view.ReasonHeld:
		return because(whyPauseAlreadyHeld)
	case t.Reason.Key == view.ReasonGate && s.Autopilot:
		// The switch is about to let this gate go, so pausing it is a race
		// the reader loses. What they want is A. Autopilot lifts the flow's
		// gates and never a pause a person put on, which is why this is the
		// only place the switch changes an answer.
		return because(whyPauseAutopilotIsLifting)
	case t.Reason.Key == view.ReasonGate:
		return because(whyPauseAlreadyWaiting)
	}

	return words.Arg{}
}

// whyNotResume refuses a running task differently from an idle one, because
// "this one is running" is an answer and "you cannot do that" is not.
func whyNotResume(t view.Task) words.Arg {
	switch {
	case parked(t):
		return words.Arg{}
	case t.Live:
		return because(whyResumeStillRunning)
	}

	return because(whyResumeNotRunning)
}

// whyNotCancel offers cancellation for anything a process still holds,
// paused and gated included: a run stopped at a gate is still holding a
// worktree and still holding a slot.
func whyNotCancel(t view.Task) words.Arg {
	if t.Live {
		return words.Arg{}
	}

	return because(whyCancelNotRunning)
}

// whyNotTake answers for the keyboard. Taking it means carrying on the
// engine's own session by hand, so it needs a session to carry on — a run
// that has started — and a phase that is not in the middle of writing
// something.
func whyNotTake(t view.Task, s Conditions) words.Arg {
	switch {
	case !started(t):
		return because(whyTakeNeverRun)
	case working(t):
		return because(whyTakeStillRunning)
	case !s.CanResume:
		return about(whyTakeEngineCannotResume, t.Engine)
	}

	return words.Arg{}
}

// whyNotHand answers for handing the keyboard back, and it asks the one
// question the previous plan could not: did this reader take it.
//
// Until Conditions carried Taken, h was offered on any parked run — including
// one that was merely paused, where "hand the keyboard back" names a keyboard
// nobody had.
//
// The engine is asked about last, as it is in whyNotTake and for the same
// reason: the sentence it produces names the engine, and a task that has
// never run has no engine to name. Ask it first and a task nothing has ever
// touched is refused with a sentence that begins with a blank. A run that is
// stopped at a phase is a run that happened, so by the time the engine's arm
// is reached there is a name to put in it.
func whyNotHand(t view.Task, s Conditions) words.Arg {
	switch {
	case !parked(t):
		return because(whyHandBackNotStopped)
	case !s.CanResume:
		return about(whyHandBackEngineCannotResume, t.Engine)
	case !s.Taken:
		return because(whyHandBackNotTaken)
	}

	return words.Arg{}
}

// whyNotMarkRead answers for the unread cap's one gesture. Marking read is
// what moves the brake, so it is offered only where it means something.
func whyNotMarkRead(t view.Task) words.Arg {
	switch {
	case view.BandOf(t) != view.Done:
		return because(whyReadNotFinished)
	case t.Read:
		return because(whyReadAlreadyRead)
	}

	return words.Arg{}
}

// whyNotDelete answers for deleting a task. A running task must be cancelled first.
func whyNotDelete(t view.Task) words.Arg {
	if t.Live {
		return because(whyDeleteRunning)
	}

	return words.Arg{}
}
