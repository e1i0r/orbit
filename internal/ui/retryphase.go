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

// aimed is the phase the reader has open on the flow tree, and whether
// there is exactly one.
//
// Exactly one, because two open nodes is two answers and picking either
// would be the window choosing for them. It is how the keyboard aims: the
// buttons on the tree are pressed with the mouse, and ^R presses the one
// whose node is open.
func (m Model) aimed() (string, bool) {
	if m.tab != tabFlow {
		return "", false
	}

	found, at := 0, -1

	for i := range m.opened[tabFlow] {
		if m.rowOpen(tabFlow, i) {
			found, at = found+1, i
		}
	}

	if found != 1 {
		return "", false
	}

	return m.phaseNamed(at)
}

// retryPhase runs the phase the reader is pointing at, and otherwise the
// one that went wrong.
func (m Model) retryPhase() (tea.Model, tea.Cmd) {
	// A node open on the tree is the reader saying which one they mean,
	// and it wins: they opened it to press its button.
	if phase, aiming := m.aimed(); aiming {
		return m.runFrom(phase)
	}

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

// runFromPhase starts the task again at one phase of its flow, named by
// where it sits in it.
//
// This is what the tree's buttons press. `^R` without one picks the phase
// that went wrong, which is the common case and needs no aiming; the
// buttons are for every other case, and there is no other way to say "the
// work is done, now do review" short of a command line.
func (m Model) runFromPhase(at int) (tea.Model, tea.Cmd) {
	name, ok := m.phaseNamed(at)
	if !ok {
		return m, nil
	}

	return m.runFrom(name)
}

// runFrom is the call both doors make.
func (m Model) runFrom(phase string) (tea.Model, tea.Cmd) {
	t := m.subject()
	if t.ID == "" || m.opts.Retry == nil {
		return m, nil
	}

	if t.Live == view.LiveHeld {
		return m.say(m.opts.Words.T("retry.running",
			"{id} is running; stop it before starting a phase again", about("id", t.ID))), nil
	}

	return m.awaiting(t.ID, gestureStart),
		retryFrom(m.opts.Retry, t, phase, board.Unread(m.board))
}
