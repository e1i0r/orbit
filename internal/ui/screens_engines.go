package ui

// Where the window and the engine knobs meet.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/engines"
	"github.com/e1i0r/orbit/internal/ui/point"
)

// enginesEnv is the world the knobs screen was written against.
func (m Model) enginesEnv() engines.Env {
	return engines.Env{
		Words:   m.opts.Words,
		Keys:    m.keys,
		Frame:   m.frame,
		Engines: m.opts.Engines,
		Quota:   m.opts.Quota,
		Settled: m.dialEngine(""),
	}
}

// tookEngines does what the screen asked the window for.
func (m Model) tookEngines(next engines.State, out engines.Out) (tea.Model, tea.Cmd) {
	m.engines = next
	// The dials are the window's — every screen that starts work reads them
	// — and this screen is where they are turned.
	m.knobs = next.Dials()

	for _, set := range out.Set {
		if applied, ok := m.settingWritten(set); ok {
			m = applied
		}
	}

	if out.Leave {
		m.screen = screen(out.Back)
		if m.screen == screenEngines {
			m.screen = screenList
		}
	}

	if out.Said != "" {
		m = m.say(out.Said)
	}

	return m, nil
}

// settingWritten puts one turned dial in the settings file, and says whether
// the window that came back is one to keep.
func (m Model) settingWritten(set engines.Setting) (Model, bool) {
	next, _ := m.applySetting(set.Name, set.Value)

	written, ok := next.(Model)

	return written, ok
}

// openEngines puts the screen up, on the dials as they stand.
func (m Model) openEngines() Model {
	m.engines = engines.Open(int(m.screen), m.knobs)
	m.screen = screenEngines

	return m
}

// enginesKey hands one keystroke to the screen.
func (m Model) enginesKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, out := m.engines.Key(msg, m.enginesEnv())

	return m.tookEngines(next, out)
}

// chooseEngineRow takes the row a click landed on.
func (m Model) chooseEngineRow(at int) (tea.Model, tea.Cmd) {
	next, out := m.engines.Choose(at, m.enginesEnv())

	return m.tookEngines(next, out)
}

// pickEngineRow moves the selection by rows, for the wheel.
func (m Model) pickEngineRow(d int) Model {
	m.engines = m.engines.Wheel(d, m.enginesEnv())

	return m
}

// knobChip is the dials in one line, for the header.
func (m Model) knobChip() string {
	return engines.Chip(m.knobs, m.enginesEnv())
}

// enginesRows is the screen drawn.
func (m Model) enginesRows(h, w int) []string {
	return m.engines.View(h, w, m.enginesEnv())
}

// hitEngines is what the screen has at that cell.
func (m Model) hitEngines(x, y int) point.Target {
	return m.engines.Hit(x, y, m.enginesEnv())
}
