package ui

// The quota screen: what is left of every engine's windows, and when each
// one comes back.
//
// The header carries a chip for the engine a run would go to, because a chip
// is what fits beside the engine's name and the share used is the part a
// reader glances at. It was drawn and answered nothing — the fact behind a
// percentage is several windows of several engines, and one line of the
// header has room for none of that. This screen is where the pointer lands
// now, and it is the only place the whole reading is written down: every
// engine Orbit can run, the windows each has, and the hour each returns.

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// openQuota brings the screen up.
func (m Model) openQuota() Model {
	m.screen = screenQuota
	return m
}

// quotaKey answers the keyboard while it is up.
//
// Nothing on the screen is chosen, so the only key is the one that leaves.
// Every other key does nothing rather than reaching the board underneath,
// which is the rule the cheat sheet and the supervisor's thread follow for
// the same reason: a reader looking at one screen should not be able to move
// a cursor on another.
func (m Model) quotaKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.Code == tea.KeyEscape || key.Matches(msg, m.keys.Back) {
		m.screen = screenList
		return m, nil
	}

	return m, nil
}

// quotaReadings is every engine the window offers, each with what is known
// about its quota right now.
//
// The engines come from the same list the engine picker is built from, so an
// engine that appears there appears here: a reader who chose codex two
// screens ago and came looking for what it has left would read its absence
// as an answer, and it would be the wrong one. An engine with no source is
// still a row — quotaSilence is what it says.
func (m Model) quotaReadings() []QuotaReading {
	if m.opts.Quota == nil || m.opts.Engines == nil {
		return nil
	}

	engines := m.opts.Engines()

	out := make([]QuotaReading, 0, len(engines))
	for _, e := range engines {
		out = append(out, m.opts.Quota(e.Name))
	}

	return out
}

// quotaSilence is what a reading with no window of its own says instead.
//
// Three ways to have no percentage, and a screen that drew the same blank
// for all of them would be the silence a quota was read to end. An engine
// paid per token has no window to be at the end of; one with a source that
// has answered nothing is a source to go and look at — a base URL a proxy
// does not serve /quota on reads exactly like an engine with no proxy, and
// only one of those is worth fixing; and an engine with nowhere to look at
// all says so, as it does on the status line.
func (m Model) quotaSilence(reading QuotaReading) string {
	p := m.opts.Words

	switch {
	case reading.Money:
		return p.T("quota.per_token", "billed per token")
	case reading.Sourced:
		return p.T("quota.silent", "source answered nothing")
	default:
		return p.T("status.quota_unread", "no quota source for {engine}",
			about("engine", reading.Engine))
	}
}
