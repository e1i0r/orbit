package ui

// settingswrite.go is the other half of the settings screen: putting one
// setting into the file, and saying what stopped it.
//
// It is a file of its own because the reading half — every row, its options
// and the keys that move between them — is a screenful on its own, and the
// two halves are read at different times: one when a dial looks wrong, the
// other when a change did not stick.

import (
	"errors"
	"slices"
	"strconv"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

// applySetting writes one setting down and says what it now is, or why it
// is not.
//
// The sentence is out of the catalogue rather than concatenated here. It was
// two English strings glued to a value — "autopilot is now on" — and it was
// the only line on a screen whose title, subtitle and every way out of it
// are already in the reader's language.
//
// What comes back from a failed write is not: those sentences name a file or
// a lock and they are English wherever they are raised, the same way the
// window puts internal/cli's "not an engine this build can run" in this band
// verbatim.
func (m Model) applySetting(keyName, val string) (tea.Model, tea.Cmd) {
	next, err := m.writeSetting(keyName, val)
	if err != nil {
		return m.say(err.Error()), nil
	}

	m = next
	p := m.opts.Words

	if keyName == "language" {
		return m.say(p.T("settings.language_changed", "language changed to {lang}", about("lang", val))),
			func() tea.Msg { return languageMsg{Lang: val} }
	}

	return m.say(p.T("settings.now", "{key} is now {val}", about("key", keyName), about("val", val))), nil
}

// writeSetting puts one setting down, and answers what stopped it.
//
// The new value is already drawn by the time this runs, so a write that fails
// silently is a window showing a setting that is not in the file, and the next
// time the reader opens it the switch has flipped itself back. The settings
// file has a lock, so these can also refuse, in words, after waiting two
// seconds for a second orbit to finish changing something.
//
// Each write happens once. Shelling out to `orbit set` afterwards would take
// the lock a second time to write the same file, and for effort and thinking
// it could only fail: neither is a key `orbit set` has.
func (m Model) writeSetting(keyName, val string) (Model, error) {
	s := m.opts.Settings
	if s == nil {
		return m, nil
	}

	switch keyName {
	case "language":
		return m, s.SetLanguage(val)
	case "autopilot":
		return m, s.SetAutopilot(val == "on")
	case "unread-cap":
		return m, m.writeUnreadCap(val)
	case "engine":
		return m.writeEngine(val)
	case "model":
		return m, s.SetModel(val)
	case "effort":
		m.knobs.Effort = val
	case "thinking":
		m.knobs.Thinking = val
	case "flow":
		// A name that is a path could never be a flow in any future, and
		// it is the one thing `orbit set` refused that this screen did
		// not — it typed the same value into the same file without the
		// check, so what the command line would not take, the window did.
		if err := flow.ValidName(val); err != nil {
			return m, err
		}

		return m, s.SetFlow(val)
	case "theme":
		if err := s.SetTheme(val); err != nil {
			return m, err
		}

		SetCurrentTheme(val)
	}

	return m, nil
}

// writeUnreadCap is the one setting a number is typed into, and the two ways
// that goes wrong.
//
// A value that was not a number was dropped on the floor and the band still
// said "unread-cap is now lots". A negative one was written, and both this
// window and internal/task read anything but a positive number as no cap at
// all — so typing -1 turned the brake off while looking like it set one.
func (m Model) writeUnreadCap(val string) error {
	p := m.opts.Words

	n, err := strconv.Atoi(val)
	if err != nil {
		return errors.New(p.T("settings.not_a_number", "{val} is not a whole number", about("val", val)))
	}

	if n < 0 {
		return errors.New(p.T("settings.negative_cap", "the unread cap cannot be negative; zero is no cap at all"))
	}

	return m.opts.Settings.SetUnreadCap(n)
}

// writeEngine also moves the model and the effort when the engine they were
// chosen for is no longer the one selected.
func (m Model) writeEngine(val string) (Model, error) {
	s := m.opts.Settings
	if err := s.SetEngine(val); err != nil {
		return m, err
	}

	models, _ := m.modelsFor(val)
	if !slices.Contains(models, s.Model()) && len(models) > 0 {
		if err := s.SetModel(models[0]); err != nil {
			return m, err
		}
	}

	efforts, _ := m.effortsFor(val)
	if !slices.Contains(efforts, m.knobs.Effort) && len(efforts) > 0 {
		m.knobs.Effort = efforts[0]
	}

	return m, nil
}
