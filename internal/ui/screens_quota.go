package ui

// Where the window and the quota screen meet.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/quota"
)

// quotaEnv is the world the quota screen was written against.
func (m Model) quotaEnv() quota.Env {
	return quota.Env{
		Words:   m.opts.Words,
		Keys:    m.keys,
		Engines: m.opts.Engines,
		Read:    m.opts.Quota,
	}
}

// openQuota brings the screen up, at the top of the reading.
func (m Model) openQuota() Model {
	m.quota = quota.Open()
	m.screen = screenQuota

	return m
}

// quotaKey hands one keystroke to the screen.
func (m Model) quotaKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, out := m.quota.Key(msg, m.quotaEnv())

	m.quota = next
	if out.Leave {
		m.screen = screenList
	}

	return m, nil
}

// wheelQuota takes the reading a notch, which is the gesture a reading this
// long is read with.
func (m Model) wheelQuota(d int) Model {
	m.quota = m.quota.Scroll(d, m.quotaEnv())

	return m
}

// quotaRows is the screen drawn.
func (m Model) quotaRows(h, w int) []string {
	return m.quota.View(h, w, m.quotaEnv())
}
