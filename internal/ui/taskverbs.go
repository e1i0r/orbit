package ui

// The verbs about one task, by the key each is offered under.

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// taskVerb does what one key stands for to the task pointedAt names, and
// says whether the key was one of them.
//
// The board answers these keys in its own map. A task's screen took p, t
// and D for itself, so it reaches the verbs here instead, for the letters
// it has left and for the menu, which names a verb by its key.
func (m Model) taskVerb(k fmt.Stringer) (tea.Model, tea.Cmd, bool) {
	var (
		next tea.Model
		cmd  tea.Cmd
	)

	switch {
	case key.Matches(k, m.keys.Pause):
		next, cmd = m.verb(m.keys.Pause, "pause")
	case key.Matches(k, m.keys.Resume):
		next, cmd = m.verb(m.keys.Resume, "resume")
	case key.Matches(k, m.keys.Skip):
		next, cmd = m.askSkip()
	case key.Matches(k, m.keys.Cancel):
		next, cmd = m.ask()
	case key.Matches(k, m.keys.Requeue):
		next, cmd = m.askRequeue()
	case key.Matches(k, m.keys.Take):
		next, cmd = m.takeKey()
	case key.Matches(k, m.keys.Hand):
		next, cmd = m.handBack()
	case key.Matches(k, m.keys.MarkRead):
		next, cmd = m.markReadKey()
	case key.Matches(k, m.keys.Delete):
		next, cmd = m.askDeleteTask()
	// The start dialog, because the menu's "start a run" arrives as its
	// letter, and on the diff tab that letter is the next hunk.
	case key.Matches(k, m.keys.Start):
		next, cmd = m.openStart()
	default:
		return m, nil, false
	}

	return next, cmd, true
}

// leftOver is a key the task screen did not take for itself: a verb about
// the task, or one the whole window answers. A is drawn in the bar on
// every screen and : is the command line from anywhere, and both did
// nothing here and said nothing.
func (m Model) leftOver(k fmt.Stringer) (tea.Model, tea.Cmd, bool) {
	if next, cmd, ok := m.taskVerb(k); ok {
		return next, cmd, true
	}

	switch {
	case key.Matches(k, m.keys.Autopilot):
		next, cmd := m.autopilot()

		return next, cmd, true
	case key.Matches(k, m.keys.Commands):
		return m.openPalette(), nil, true
	// The filter is the board's, so / goes back to it with the filter
	// open: a task is looked for among the others, not inside one.
	case key.Matches(k, m.keys.Filter):
		m.screen, m.filtering = screenList, true

		return m, nil, true
	}

	return m, nil, false
}
