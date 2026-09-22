package ui

// Running one phase again, when that phase is the one that went wrong.
//
// The phases before it are left exactly as the record has them. Elio's
// rule, and the reason this is not simply `b`: a phase that ended well
// does not need doing again, and re-running it would spend the money a
// second time to reproduce work already written down.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
)

// troubled is the phase this task's record says went wrong, and whether
// there is one.
//
// The last one, not the first: a task run three times has three attempts
// in one log, and the phase worth offering is the one the newest attempt
// stopped in. Two things count as trouble — a phase that broke and a phase
// somebody stopped — and neither is the same as a phase that merely
// finished with a gate unhappy, which the run already retried by itself
// as many times as the flow allowed.
func troubled(entries []view.Entry) (string, bool) {
	name := ""

	for _, e := range entries {
		switch e.What() {
		case view.EntryStarted:
			// A phase that started again after failing is not the phase
			// to offer: whatever happened to it, the record moved on.
			if e.Phase != "" && e.Phase == name {
				name = ""
			}
		case view.EntryFailed, view.EntryCancelled:
			if e.Phase != "" {
				name = e.Phase
			}
		}
	}

	return name, name != ""
}

// canRetryPhase is whether this window can offer the gesture at all: a
// port to run it through, a phase that went wrong, and no run in flight to
// collide with.
func (m Model) canRetryPhase() (string, bool) {
	if m.opts.Retry == nil {
		return "", false
	}

	// Not while something holds it. A second run in the same worktree is
	// the collision task.Start refuses anyway, and refusing it here is
	// the difference between a key that is not offered and a key that is
	// offered and then says no.
	t := m.subject()
	if t.ID == "" || t.Live == view.LiveHeld {
		return "", false
	}

	return troubled(m.entries)
}

// retryPhase runs the phase that went wrong.
func (m Model) retryPhase() (tea.Model, tea.Cmd) {
	phase, ok := m.canRetryPhase()
	if !ok {
		return m.say(m.opts.Words.T("retry.nothing_wrong",
			"no phase of this task went wrong, so there is nothing to run again")), nil
	}

	t := m.subject()

	return m.say(m.opts.Words.T("retry.asked", "{phase} is being run again for {id}",
			about("phase", phase), about("id", t.ID))),
		retryFrom(m.opts.Retry, t, phase, board.Unread(m.board))
}

// retryFrom is the port call, off the draw loop like every other one.
func retryFrom(
	port func(view.Task, string, int) (int, error), t view.Task, phase string, unread int,
) tea.Cmd {
	return func() tea.Msg {
		if port == nil {
			return startedMsg{ID: t.ID, Err: errNoStartPort}
		}

		pid, err := port(t, phase, unread)

		return startedMsg{ID: t.ID, Pid: pid, Err: err}
	}
}

// runFromPhase answers one of the tree's buttons, by the number the node it
// hangs off carries.
//
// The phases are numbered first and the verbs asked for by hand after them,
// so a number past the last phase is one of those and goes to its own door.
// One map of buttons and one target kind, because to a reader they are the
// same gesture: press the thing on the node and the node does its thing.
func (m Model) runFromPhase(at int) (tea.Model, tea.Cmd) {
	name, ok := m.phaseNamed(at)
	if !ok {
		return m.pressedHand(at)
	}

	return m.runFrom(name)
}

// runFrom is the call both doors make.
func (m Model) runFrom(phase string) (tea.Model, tea.Cmd) {
	t := m.subject()
	if t.ID == "" {
		return m, nil
	}

	// A phase in flight is stopped rather than started, which is what its
	// button says. Signalled and not asked for, because the word in the
	// control file is read at the next phase boundary and the phase this
	// is about is the one that has not reached it.
	if t.Live == view.LiveHeld {
		if m.opts.Stop == nil {
			return m.say(m.opts.Words.T("retry.running",
				"{id} is running; stop it before starting a phase again", about("id", t.ID))), nil
		}

		return m.awaiting(t.ID, gestureCancel), stop(m.opts.Stop, t)
	}

	if m.opts.Retry == nil {
		return m, nil
	}

	return m.awaiting(t.ID, gestureStart),
		retryFrom(m.opts.Retry, t, phase, board.Unread(m.board))
}
