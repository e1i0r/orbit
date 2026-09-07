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

// openQuota brings the screen up.
func (m Model) openQuota() Model {
	m.screen = screenQuota

	return m
}

// quotaKey hands one keystroke to the screen.
func (m Model) quotaKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if out := quota.Key(msg, m.quotaEnv()); out.Leave {
		m.screen = screenList
	}

	return m, nil
}

// quotaRows is the screen drawn.
func (m Model) quotaRows(h, w int) []string {
	return quota.View(h, w, m.quotaEnv())
}
