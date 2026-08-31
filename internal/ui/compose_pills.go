package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

const composeLabelWidth = 14

func composeLabel(label string, active bool) string {
	mark := strings.Repeat(" ", gutter)
	if active {
		mark = markGlyph + strings.Repeat(" ", gutter-1)
	}

	padded := pad(label+":", composeLabelWidth, false)

	return mark + Paint(Dim).Render(padded) + " "
}

func (m Model) composeRepoLine(active bool, w int) string {
	p := m.opts.Words
	prefix := composeLabel(p.T("compose.repo", "repository"), active)

	if len(m.compose.repos) == 0 {
		val := m.compose.repo
		role := Accent

		if val == "" {
			val = p.T("compose.repo_placeholder", "which repository?")
			role = Dim
		}

		line := prefix + Paint(role).Render(val)
		if active {
			line += Paint(Sel).Render(" ")
		}

		return fit(line, w)
	}

	var pills []string

	for i, r := range m.compose.repos {
		selected := i == m.compose.repoIdx
		if selected {
			pills = append(pills, Pill(" ● "+r.name+" ", "#000000", "#38BDF8"))
		} else {
			pills = append(pills, Pill(" "+r.name+" ", "#94A3B8", "#1E293B"))
		}
	}

	line := prefix + strings.Join(pills, " ")
	if active {
		line += " " + Paint(Dim).Render(p.T("compose.repo_hint", "(←/→ to cycle)"))
	}

	return fit(line, w)
}

func (m Model) composeFlowLine(active bool, w int) string {
	p := m.opts.Words
	prefix := composeLabel(p.T("compose.flow", "flow"), active)

	var pills []string

	for i, f := range m.compose.flows {
		selected := i == m.compose.flowIdx
		glyph := "⚡ "

		switch f {
		case "quick":
			glyph = "🚀 "
		case "careful":
			glyph = "🛡️ "
		}

		if selected {
			pills = append(pills, Pill(" ● "+glyph+f+" ", "#000000", "#A855F7"))
		} else {
			pills = append(pills, Pill(" "+glyph+f+" ", "#94A3B8", "#1E293B"))
		}
	}

	newBtn := Pill(" ➕ "+p.T("compose.new_flow_btn", "New")+" ", "#FFFFFF", "#6366F1")
	pills = append(pills, newBtn)

	line := prefix + strings.Join(pills, " ")
	if active {
		line += " " + Paint(Dim).Render(p.T("compose.flow_hint", "(←/→ to cycle, click again/i for details, + new)"))
	}

	return fit(line, w)
}

func (m Model) flowSummary(name string) string {
	fl, err := flow.Resolve(m.opts.Flows, name)
	if err != nil {
		return ""
	}

	var phaseDescs []string

	for _, ph := range fl.Phases {
		desc := ph.Name
		if ph.Engine != "" {
			engMod := ph.Engine
			if ph.Model != "" && ph.Model != "default" {
				engMod += "/" + ph.Model
			}

			desc += " (" + engMod + ")"
		}

		if ph.Wait {
			desc += " ⏸"
		}

		phaseDescs = append(phaseDescs, desc)
	}

	pipeline := strings.Join(phaseDescs, " ➔ ")
	if fl.Description != "" {
		return pipeline + " · " + fl.Description
	}

	return pipeline
}

const composeLabelStart = gutter + composeLabelWidth + 1

func composePillWidth(name string, selected bool) int {
	if selected {
		return lipgloss.Width(Pill(" ● "+name+" ", "#000000", "#FFFFFF"))
	}

	return lipgloss.Width(Pill(" "+name+" ", "#94A3B8", "#1E293B"))
}
