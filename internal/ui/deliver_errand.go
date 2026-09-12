package ui

// What one deliver key asks for, and who it is asked of.
//
// It is beside deliver_actions.go rather than in it because the six verbs
// there are a list and this is the shape they all take: the ask itself, the
// task it is about, and the sentence the band says once it is out.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/supervisor"
)

// askSupervisorTo hands one of these verbs to the supervisor: the window
// says what is wanted and where, and the supervisor finds out what that
// takes before doing it.
//
// What it is asked goes into the supervisor's own thread, exactly as a line
// typed on its screen does, so a keystroke and a sentence are one
// conversation in one order — and the answer comes back where the operator
// already reads its answers.
func (m Model) askSupervisorTo(e errand) (tea.Model, tea.Cmd) {
	p := m.opts.Words

	path := m.taskCheckoutPath(e.TaskID)
	if path == "" {
		return m.say(p.T("deliver.no_checkout", "{id} has no checkout, so there is nothing to work in",
			about("id", e.TaskID))), nil
	}

	next, cmd := m.sendSupervisorMessage(supervisor.Deliver("the cockpit", e.Caption, e.TaskID, path, e.Body))
	if cmd == nil {
		// The thread refused the line. What it said about that is the
		// only true sentence there is here.
		return next, nil
	}

	return next.asked(ask{TaskID: e.TaskID, Verb: e.Caption, By: deliverBySupervisor}).say(e.Said), cmd
}

// errand is one thing the supervisor is asked to do about a task.
//
// It is a struct and not five string parameters: the five were the same type
// in a row, so two of them swapped by hand would compile, run, and put the
// wrong verb in the record with the right sentence in the band. The checkout
// is not among them because every caller passed the same expression for it
// — where a task is worked is a fact about the task, not a choice the caller
// makes.
type errand struct {
	// Caption is the verb as the record, the band and the prompt all name
	// it: CREATE PR, FIX CHECKS.
	Caption string
	TaskID  string
	// Body is the instruction itself, out of internal/supervisor.
	Body string
	// Said is what the band says once the ask is out.
	Said string
}
