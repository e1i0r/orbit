package ui

// cycleEffort moves the effort knob to the next one the engine offers.
//
// The list used to be written out here — low, medium, high, xhigh — so this
// knob and the engine it was turned for disagreed: codex has no xhigh, and
// a phase carrying one is a phase internal/task refuses before it runs.
func (m Model) cycleEffort() Model {
	efforts, _ := m.effortsFor(m.dialEngine(m.knobs.Engine))
	m.knobs.Effort = nextOption(efforts, m.knobs.Effort, 1)

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
	// turned, and what stands behind it when none is. It used to be the
	// words claude, sonnet and high, which on a build without claude were
	// true of none of the three.
	eng := m.dialEngine(m.knobs.Engine)

	models, _ := m.modelsFor(eng)
	mod := orDef(m.knobs.Model, first(models))

	if s := m.opts.Settings; s != nil && m.knobs.Model == "" {
		mod = orDef(s.Model(), mod)
	}

	efforts, _ := m.effortsFor(eng)
	eff := orDef(m.knobs.Effort, first(efforts))

	eng, mod, eff = orDef(eng, unsetDial), orDef(mod, unsetDial), orDef(eff, unsetDial)

	thk := m.knobs.Thinking

	thkLabel := p.T("start.thinking_on", "thinking: on")
	if thk == "off" {
		thkLabel = p.T("start.thinking_off", "thinking: off")
	}

	left := startIndent + Paint(Dim).Render(p.T("start.engine_config", "engine")) + "     " +
		Paint(Live).Bold(true).Render(eng) + " · " +
		Paint(Accent).Render(mod) + " · " +
		Paint(Dim).Render(p.T("start.effort_label", "effort:")+eff) + " · " +
		Paint(OK).Render(thkLabel)

	hints := Paint(Dim).Render(p.T("start.dials_hint", "[m] model  [o] effort  [t] thinking"))

	return spread(left, hints, w)
}

// unset is what a dial with nothing on it is drawn as. A window whose
// engines port answers nothing has no engine, no model and no effort to
// name, and a dash says that without naming one it does not have.
const unsetDial = "—"
