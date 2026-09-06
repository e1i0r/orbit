package ui

import "github.com/e1i0r/orbit/internal/ui/theme"

// The designer's diagram tab: the flow as it will run, read rather than
// edited.
//
// It draws the same flowchart and the same cards the flow inspector draws,
// off the phases being edited — so what somebody checks here is what they
// have, not what was last saved. Every card selects its phase and takes the
// reader to the fields, which is the gesture the diagram invites: you look
// at the picture to decide what to change.

// diagramRows is the tab.
func (m Model) diagramRows(w int) []builderLine {
	st := &m.flows
	p := m.opts.Words

	out := []builderLine{
		plainLine(""),
		plainLine("  " + theme.Paint(theme.Accent).Bold(true).Render(p.T("flows.pipeline_diagram", "Pipeline Flowchart:"))),
	}

	for _, line := range renderFlowDiagram(st.phases, w-4) {
		out = append(out, plainLine(fit("  "+line, w)))
	}

	out = append(out,
		plainLine(""),
		plainLine("  "+theme.Paint(theme.Live).Bold(true).Render(p.T("flows.phase_breakdown", "Phases Breakdown:"))),
	)

	for i, ph := range st.phases {
		for _, line := range m.phaseCard(i, ph, w) {
			out = append(out, builderLine{text: fit(line, w), field: noField, phase: i, pick: noPick})
		}
	}

	return append(out,
		plainLine(""),
		plainLine(fit("  "+theme.Paint(theme.Dim).Render(p.T("flows.diagram_ways",
			"click a phase to edit it · [^←/^→] tab · [esc] back")), w)),
	)
}
