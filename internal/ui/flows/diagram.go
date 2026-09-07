package flows

import (
	"github.com/e1i0r/orbit/internal/ui/theme"

	"github.com/e1i0r/orbit/internal/ui/cells"
)

// The designer's diagram tab: the flow as it will run, read rather than
// edited.
//
// It draws the same flowchart and the same cards the flow inspector draws,
// off the phases being edited — so what somebody checks here is what they
// have, not what was last saved. Every card selects its phase and takes the
// reader to the fields, which is the gesture the diagram invites: you look
// at the picture to decide what to change.

// diagramRows is the tab.
func (s State) diagramRows(w int, e Env) []builderLine {
	p := e.Words

	out := []builderLine{
		plainLine(""),
		plainLine("  " + theme.Paint(theme.Accent).Bold(true).Render(p.T("flows.pipeline_diagram", "Pipeline Flowchart:"))),
	}

	for _, line := range renderFlowDiagram(s.phases, w-4) {
		out = append(out, plainLine(cells.Fit("  "+line, w)))
	}

	out = append(out,
		plainLine(""),
		plainLine("  "+theme.Paint(theme.Live).Bold(true).Render(p.T("flows.phase_breakdown", "Phases Breakdown:"))),
	)

	for i, ph := range s.phases {
		for _, line := range s.phaseCard(i, ph, w, e) {
			out = append(out, builderLine{text: cells.Fit(line, w), field: noField, phase: i, pick: noPick})
		}
	}

	return append(out,
		plainLine(""),
		plainLine(cells.Fit("  "+theme.Paint(theme.Dim).Render(p.T("flows.diagram_ways",
			"click a phase to edit it · [^←/^→] tab · [esc] back")), w)),
	)
}
