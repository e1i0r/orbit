package ui

// The circuit breaker: when every run the queue picks up comes back stuck,
// the queue stops picking up.

import (
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

// breakerStuck is how many runs in a row must end stuck before autopilot
// goes off.
//
// Three, because two is a task. A task that got stuck twice is a hard task
// and switching the queue off for it would be Orbit deciding what is worth
// trying; three runs in a row is a board where nothing is working, and what
// is usually wrong then is the repository or the plan rather than any of the
// three tasks.
const breakerStuck = 3

// breaker takes autopilot off when the last three runs to stop all ended
// stuck, and says so where a reader and the supervisor will both find it.
//
// It flips the switch the reader's own key flips rather than refusing to
// start tasks behind its back. A second mechanism holding the queue shut
// while the header still says autopilot is on is a window that lies about
// its own state, and the reader's answer to it — pressing the key that is
// already on — does nothing.
//
// Which is also why it does not trip twice for the same board. The poll is
// twice a second, and a reader who turns autopilot back on is making a
// judgement about what they have just read; a breaker that flipped it
// straight back off would be arguing with them. It trips again when a run
// stops stuck after the one it tripped on, which is new evidence rather than
// the same evidence read again.
func (m Model) breaker() Model {
	if !m.autopilotOn() {
		return m
	}

	streak := view.StuckStreak(m.board.Tasks)
	if len(streak) < breakerStuck {
		return m
	}

	newest := streak[0].Since
	if !newest.After(m.brokeAt) {
		return m
	}

	m.brokeAt = newest
	if err := m.opts.Settings.SetAutopilot(false); err != nil {
		return m.say(err.Error())
	}

	p := m.opts.Words
	m = m.say(p.T("msg.breaker", "autopilot off: {n} runs in a row ended stuck",
		about("n", strconv.Itoa(len(streak)))))

	if m.opts.RecordSupervisor == nil {
		return m
	}

	// "orbit" and not "operator": nobody typed this. The thread is read
	// back into the supervisor's own prompt, so a line claiming to be the
	// operator's would be an instruction the operator never gave.
	if err := m.opts.RecordSupervisor("orbit", "breaker", breakerLine(streak)); err != nil {
		return m.say(err.Error())
	}

	return m
}

// breakerLine is what the supervisor's thread is told, in the words a person
// reading that thread tomorrow needs: how many, which ones, and what three
// in a row usually means.
func breakerLine(streak []view.Task) string {
	ids := make([]string, 0, len(streak))
	for _, t := range streak {
		ids = append(ids, t.ID)
	}

	return "Autopilot off: the last " + strconv.Itoa(len(streak)) +
		" runs to stop all ended stuck (" + strings.Join(ids, ", ") + "). " +
		"Three in a row is usually the repository or the plan rather than the tasks, " +
		"so nothing new starts until somebody has looked."
}
