package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/e1i0r/orbit/internal/view"
)

// sendKey puts a keystroke through the same map a pressed key goes through,
// which is how a clicked hint reaches the verb it names.
func (m Model) sendKey(k keystroke) (tea.Model, tea.Cmd) {
	switch {
	case m.filtering:
		return m, nil
	case m.screen == screenStart:
		return m.startKey(k)
	case m.screen == screenDetail:
		return m.detailKey(k)
	}
	return m.listKey(k)
}

// flip is one of the start dialog's switches, clicked.
func (m Model) flip(field string) (tea.Model, tea.Cmd) {
	on := m.autopilotOn()
	switch {
	case field == fieldFlow:
		return m.cycleFlow(), nil
	case field == fieldAutopilotOn && !on, field == fieldAutopilotOff && on:
		return m.autopilot()
	}
	return m, nil
}

// rightClick opens the menu for what was pointed at.
func (m Model) rightClick(t Target) (tea.Model, tea.Cmd) {
	if t.Kind == TargetPaneBody {
		if s := m.subject(); s.ID != "" {
			return m.openMenu(s.ID), nil
		}
		return m, nil
	}
	i, ok := m.rowOf(t)
	if !ok {
		return m, nil
	}
	next := m.moveTo(i)
	if t.Kind == TargetTask {
		return next.openMenu(t.ID), nil
	}
	return next, nil
}

func (m Model) jumpToBand(b view.Band) (tea.Model, tea.Cmd) {
	m = m.expand(b)
	all := m.rows()
	for i, r := range all {
		if r.band == b && !r.blank {
			return m.moveTo(i).clampCursor(), nil
		}
	}
	return m, nil
}
