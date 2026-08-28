package ui

import (
	"maps"

	"charm.land/bubbles/v2/key"

	"github.com/e1i0r/orbit/internal/view"
)

// conditions is the standing state the verbs are asked about, for one task.
//
// It takes the task because two of the three answers are about a particular
// one: whether this window handed the terminal to an engine for it, and
// whether the engine that ran it can carry a session on at all. Only the
// autopilot switch is about the whole program.
func (m Model) conditions(t view.Task) Conditions {
	return Conditions{
		Autopilot: m.autopilotOn(),
		CanResume: m.canResume(t.Engine),
		Taken:     m.taken[t.ID],
	}
}

// canResume asks the port about one engine by name, and answers no for a
// window that was handed no way to ask — a rendering test, or a window built
// before the composition root knows what it can run.
func (m Model) canResume(engine string) bool {
	return m.opts.CanResume != nil && m.opts.CanResume(engine)
}

// autopilotOn reads the switch, and answers for a window opened without a
// settings file at all — which is what a rendering test hands it.
func (m Model) autopilotOn() bool {
	return m.opts.Settings != nil && m.opts.Settings.Autopilot()
}

// unreadCap is how many unread finished tasks may stand before nothing new
// starts, and zero when there is no settings file to ask. Whether the brake
// is actually on is atUnreadCap, beside the refusal it produces.
func (m Model) unreadCap() int {
	if m.opts.Settings == nil {
		return 0
	}

	return m.opts.Settings.UnreadCap()
}

// affordance finds one verb's answer for one task, by the glyph its binding
// prints. The glyph is the same in every language, which is what lets this
// match a binding the key map may have rebuilt since.
func (m Model) affordance(t view.Task, b key.Binding) (Affordance, bool) {
	for _, a := range m.keys.Affordances(t, m.conditions(t)) {
		if a.Key.Help().Key == b.Help().Key {
			return a, true
		}
	}

	return Affordance{}, false
}

// task finds one task on the board by id.
func (m Model) task(id string) (view.Task, bool) {
	for _, t := range m.board.Tasks {
		if t.ID == id {
			return t, true
		}
	}

	return view.Task{}, false
}

// expand toggles one band open or shut, on a copy of the map.
func (m Model) expand(b view.Band) Model {
	open := maps.Clone(m.expanded)
	open[b] = !open[b]
	m.expanded = open

	return m
}
