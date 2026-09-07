package ui

import (
	"github.com/e1i0r/orbit/internal/ui/theme"

	"github.com/e1i0r/orbit/internal/ui/cells"
)

// cycleEffort moves the effort knob to the next one the engine offers.
//
// The list is not written out here — low, medium, high, xhigh — because that
// makes this knob and the engine it is turned for disagree: codex has no
// xhigh, and a phase carrying one is a phase internal/task refuses before it
// runs.
func (m Model) cycleEffort() Model {
	efforts, _ := m.effortsFor(m.dialEngine(m.knobs.Engine))
	m.knobs.Effort = cells.NextOption(efforts, m.knobs.Effort, 1)

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

	// What is drawn is what a run would actually use: the knob when one is
	// turned, and what stands behind it when none is. The words claude,
	// sonnet and high are true of none of the three on a build without
	// claude.
	eng := m.dialEngine(m.knobs.Engine)

	models, _ := m.modelsFor(eng)
	mod := cells.OrDef(m.knobs.Model, cells.First(models))

	if s := m.opts.Settings; s != nil && m.knobs.Model == "" {
		mod = cells.OrDef(s.Model(), mod)
	}

	efforts, _ := m.effortsFor(eng)
	eff := cells.OrDef(m.knobs.Effort, cells.First(efforts))

	eng, mod, eff = cells.OrDef(eng, unsetDial), cells.OrDef(mod, unsetDial), cells.OrDef(eff, unsetDial)

	thk := m.knobs.Thinking

	thkLabel := p.T("start.thinking_on", "thinking: on")
	if thk == "off" {
		thkLabel = p.T("start.thinking_off", "thinking: off")
	}

	left := startIndent + theme.Paint(theme.Dim).Render(p.T("start.engine_config", "engine")) + "     " +
		theme.Paint(theme.Live).Bold(true).Render(eng) + " · " +
		theme.Paint(theme.Accent).Render(mod) + " · " +
		theme.Paint(theme.Dim).Render(p.T("start.effort_label", "effort:")+eff) + " · " +
		theme.Paint(theme.OK).Render(thkLabel)

	hints := theme.Paint(theme.Dim).Render(p.T("start.dials_hint", "[m] model  [o] effort  [t] thinking"))

	return cells.Spread(left, hints, w)
}

// unset is what a dial with nothing on it is drawn as. A window whose
// engines port answers nothing has no engine, no model and no effort to
// name, and a dash says that without naming one it does not have.
const unsetDial = "—"
