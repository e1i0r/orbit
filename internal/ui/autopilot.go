package ui

// The standing switch: with it on, a task in to do starts by itself and every
// phase runs without asking.
//
// It is here rather than in keypress.go because it is not a keystroke — the
// key is one of three doors onto it, beside the header chip and the bar's
// own — and keypress.go is the table of what a key does, not where the thing
// it does lives.

import (
	tea "charm.land/bubbletea/v2"
)

// autopilot flips the standing switch and says which way it went.
//
// It says what it just did rather than what it undid. The program this
// replaces printed "autopilot was off" after turning it on, and the sentence
// is ambiguous in exactly the moment a reader needs it not to be.
func (m Model) autopilot() (tea.Model, tea.Cmd) {
	if m.opts.Settings == nil {
		return m, nil
	}

	on := !m.opts.Settings.Autopilot()
	if err := m.opts.Settings.SetAutopilot(on); err != nil {
		return m.say(err.Error()), nil
	}

	if on {
		m = m.say(m.opts.Words.T("msg.autopilot_on", "autopilot is on: every phase runs without asking"))
		nextM, cmd := m.autoStartNext()

		return nextM, cmd
	}

	return m.say(m.opts.Words.T("msg.autopilot_off", "autopilot is off: every phase stops for you")), nil
}
