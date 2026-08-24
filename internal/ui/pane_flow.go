package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/flow"
)

// flowLines renders Pane 2: Flow & Phase Breakdown.
func (m Model) flowLines() []string {
	p := m.opts.Words
	t, ok := m.task(m.detail)
	if !ok {
		return []string{"  " + Paint(Dim).Render(p.T("detail.gone", "this task is no longer on the board"))}
	}

	flowName := t.Flow
	if flowName == "" {
		flowName = "quick"
	}
	f, err := flow.Resolve(m.opts.Flows, flowName)
	if err != nil {
		return []string{"  " + Paint(Bad).Render(fmt.Sprintf("flow %q: %v", flowName, err))}
	}

	out := []string{
		"",
		"  " + Paint(Accent).Render(p.T("flow.title", "Flow: {name}", about("name", f.Name))),
		"",
	}

	for i, phase := range f.Phases {
		tag := fmt.Sprintf("[%d/%d] %s", i+1, len(f.Phases), phase.Name)
		var details []string
		if phase.Engine != "" {
			details = append(details, phase.Engine)
		}
		if phase.Model != "" {
			details = append(details, phase.Model)
		}
		if phase.Effort != "" {
			details = append(details, "effort:"+phase.Effort)
		}
		if phase.Thinking != "" {
			details = append(details, "thinking:"+phase.Thinking)
		}
		if len(phase.Permissions) > 0 {
			details = append(details, "perms:"+strings.Join(phase.Permissions, ","))
		}
		if phase.Wait {
			details = append(details, "stops at gate")
		}
		if len(phase.Gates) > 0 {
			details = append(details, fmt.Sprintf("%d verification gate(s)", len(phase.Gates)))
		}

		out = append(out,
			"  "+Paint(Accent).Render(tag),
			"      "+Paint(Dim).Render(strings.Join(details, " · ")),
		)
		for _, g := range phase.Gates {
			out = append(out, "      "+Paint(Dim).Render("gate "+g.Name+": ")+g.Command)
		}
		out = append(out, "")
	}

	return out
}
