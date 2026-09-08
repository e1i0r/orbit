package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/view"
)

// sendKey puts a keystroke through the same door a pressed key goes through,
// which is how a clicked hint reaches the verb it names.
//
// key() and not the screen maps under it. Those are the second half of the
// dispatch: key() answers the menu, the palette and the filter above them and
// only then falls through, so a hint the bar drew for one of those was
// promised to the reader and dropped — the click reached a map with no case
// for it and nothing happened, which is the shape of "I clicked it and it did
// nothing". One door for both, and a clicked hint is its key.
func (m Model) sendKey(k keystroke) (tea.Model, tea.Cmd) {
	// Except while a filter is being typed, where the character would land
	// in the box. A hint clicked then is not a character the reader typed.
	if m.filtering {
		return m, nil
	}

	return m.key(tea.KeyPressMsg{Text: string(k)})
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
func (m Model) rightClick(t point.Target) (tea.Model, tea.Cmd) {
	if t.Kind == point.PaneBody {
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
	if t.Kind == point.Task {
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
