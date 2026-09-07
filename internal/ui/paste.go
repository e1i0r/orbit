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
		m.supervisor = m.supervisor.Type(trimmed)

		return m
	case m.note.open:
		m.note.text += trimmed
		return m
	case m.screen == screenCompose:
		m.compose = m.compose.Type(trimmed)

		return m
	case m.filtering:
		m.filter += trimmed
		return m.clampCursor()
	case m.palette.Up():
		m.palette = m.palette.Type(trimmed)
		return m
	case m.screen == screenFlows:
		if m.flows.Creating() {
			m.flows.Write(trimmed)
		}

		return m
	}

	return m
}
