package ui

import (
	"strings"
)

// cycleEffort cycles runtime effort setting across low, medium, high, xhigh.
func (m Model) cycleEffort() Model {
	efforts := []string{"low", "medium", "high", "xhigh"}

	cur := m.knobs.Effort
	if cur == "" {
		cur = "high"
	}

	nextIdx := 0

	for i, e := range efforts {
		if strings.EqualFold(e, cur) {
			nextIdx = (i + 1) % len(efforts)
			break
		}
	}

	m.knobs.Effort = efforts[nextIdx]

	return m
}

// cycleThinking toggles extended thinking reasoning on/off.
func (m Model) cycleThinking() Model {
	cur := m.knobs.Thinking
	if cur == "" || cur == "thinking" || cur == "on" {
		m.knobs.Thinking = "off"
	} else {
		m.knobs.Thinking = "thinking"
	}

	return m
}

// configLine draws the active engine, model, effort, and thinking options with interactive shortcuts.
func (m Model) configLine(w int) string {
	p := m.opts.Words

	eng := m.knobs.Engine
	if eng == "" {
		eng = "claude"
	}

	mod := m.knobs.Model
	if mod == "" {
		mod = "sonnet"
	}

	eff := m.knobs.Effort
	if eff == "" {
		eff = "high"
	}

	thk := m.knobs.Thinking

	thkLabel := "thinking: on"
	if thk == "off" {
		thkLabel = "thinking: off"
	}

	left := startIndent + Paint(Dim).Render(p.T("start.engine_config", "motor")) + "     " +
		Paint(Live).Bold(true).Render(eng) + " · " +
		Paint(Accent).Render(mod) + " · " +
		Paint(Dim).Render("effort:"+eff) + " · " +
		Paint(OK).Render(thkLabel)

	hints := Paint(Dim).Render(p.T("start.dials_hint", "[m] modelo  [o] esfuerzo  [t] thinking"))

	return spread(left, hints, w)
}
