package ui

// The filter: one of the window's two text-entry modes, the palette being
// the other.
//
// It is a file of its own for the reason the ceiling asks for and the seam
// agrees with: these are the keys of one mode — what typing into a line
// means, how it is undone, and what closes it — and they sat beside the
// routing that hands keys to every mode until the palette arrived needing
// the same room. Nothing here changed when it moved.

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
)

// filterKey feeds the text input, which owns every key it is not given a
// reason to give up.
func (m Model) filterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.filtering, m.filter = false, ""
		return m.clampCursor(), nil
	case key.Matches(msg, m.keys.Open):
		// The filter stays applied and the keyboard goes back to the list.
		// Closing it and clearing it are two different gestures because
		// filtering to one repository and then working in it is the whole
		// point of having filtered.
		m.filtering = false
		return m.clampCursor(), nil
	case msg.Code == tea.KeyBackspace:
		m.filter = cells.TrimLastRune(m.filter)
		return m.clampCursor(), nil
	}

	if msg.Text != "" {
		m.filter += msg.Text
	}

	return m.clampCursor(), nil
}
