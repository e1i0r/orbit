package ui

import (
	"github.com/e1i0r/orbit/internal/flow"
)

func (m Model) startCreateFlow() Model {
	m.flows.creating = true
	m.flows.engine = m.dialEngine("")
	m.flows.isEditing = false
	m.flows.confirmDiscard = false
	m.flows.confirmDelete = false
	m.flows.field = 0
	m.flows.template = "ninguna"
	m.flows.flowName = ""
	m.flows.phases = []flow.Phase{
		{Name: "1-implement", Engine: m.flows.engine, Thinking: "adaptive", Permissions: []string{"repo"}},
	}
	m.flows.activePhase = 0
	m.flows.checksTyped = false
	m.flows.scroll = 0

	return m
}

// flowsBuilderRows is the form as the window draws it: the rows of
// builderLines, from wherever the window starts.
func (m Model) flowsBuilderRows(h, w int) []string {
	lines, start := m.builderView(h, w)

	out := make([]string, 0, h)
	for _, l := range lines[min(start, len(lines)):] {
		out = append(out, l.text)
	}

	return fill(out, h)
}
