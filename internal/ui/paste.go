package ui

import (
	"strings"
)

// paste inserts clipboard or bracketed paste content into the active focused
// field.
//
// Both gestures end here. ^V shells out to pbpaste, which is the one that
// works over ssh and in a terminal with bracketed paste turned off; cmd+V
// arrives as tea.PasteMsg, wrapped by the terminal and handed over whole. A
// screen missing from the switch below takes neither, and says nothing about
// not having taken them.
func (m Model) paste(content string) Model {
	trimmed := strings.TrimRight(content, "\r\n")
	if trimmed == "" {
		return m
	}

	switch {
	case m.screen == screenSupervisor:
		// While a line is being picked there is nothing being typed into,
		// and text arriving would land in a field nobody can see.
		if !m.supervisor.picking {
			m.supervisor.input += trimmed
		}

		return m
	case m.note.open:
		m.note.text += trimmed
		return m
	case m.screen == screenCompose:
		return m.composeEdit(func(in *input) { in.insert(trimmed) })
	case m.filtering:
		m.filter += trimmed
		return m.clampCursor()
	case m.palette.open:
		m.palette.typed += trimmed
		return m.ensureVisible()
	case m.screen == screenFlows:
		if m.flows.creating {
			m.flows.write(trimmed)
		}

		return m
	}

	return m
}
