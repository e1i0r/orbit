package ui

import (
	"strings"
)

// paste inserts clipboard or bracketed paste content into the active focused field.
func (m Model) paste(content string) Model {
	trimmed := strings.TrimRight(content, "\r\n")
	if trimmed == "" {
		return m
	}

	switch {
	case m.screen == screenCompose:
		m.compose.set(m.compose.get() + trimmed)
		m.onComposeChanged()
		return m
	case m.filtering:
		m.filter += trimmed
		return m.clampCursor()
	case m.palette.open:
		m.palette.typed += trimmed
		return m.ensureVisible()
	case m.screen == screenFlows:
		if m.flows.creating {
			m.flows.cur().Prompt += trimmed
		}
		return m
	}
	return m
}
