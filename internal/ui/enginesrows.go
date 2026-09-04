package ui

// The rows of the knobs screen: every engine, the models under the ones that
// are open, and the two dials that belong to the engine in force.
//
// The list is built from scratch on every frame rather than kept, because
// each row carries an answer that can change under it — whether the engine
// is installed, which model is chosen, what the filter has cut away.

func (m Model) collectEngineRows() []engineRow {
	p := m.opts.Words

	var rows []engineRow

	engineList := m.engineTable()

	activeEngine := m.dialEngine(m.knobs.Engine)

	// 1. Model & Engine Section
	rows = append(rows, engineRow{kind: rowHeader, title: p.T("engines.section_model", "Engine & Model")})

	var currentEng EngineInfo

	for _, eng := range engineList {
		if eng.Name == activeEngine {
			currentEng = eng
		}

		if !eng.Available {
			var steps []string
			if eng.Setup != nil {
				steps = eng.Setup(p)
			}

			rows = append(rows, engineRow{
				kind:     rowEngine,
				title:    eng.Name + " " + p.T("engines.setup_tag", "[setup required]"),
				engine:   eng.Name,
				disabled: true,
				setup:    steps,
			})

			continue
		}

		// A filter opens what it matched: a name with the models it was
		// typed to find still folded under it says nothing.
		models := m.matchingModels(eng)
		if m.knobFilter() != "" && len(models) == 0 {
			continue
		}

		open := m.engineOpen(eng.Name) || m.knobFilter() != ""

		rows = append(rows, engineRow{
			kind:     rowEngine,
			title:    eng.Name,
			engine:   eng.Name,
			selected: eng.Name == activeEngine,
			open:     open,
			models:   len(models),
			chosen:   m.chosenModel(eng),
		})

		if open {
			for _, mdl := range models {
				lbl := mdl.Label
				if mdl.ID == "" {
					lbl = p.T("engines.default", "default")
				}

				rows = append(rows, engineRow{
					kind:     rowModel,
					title:    "  " + lbl,
					engine:   eng.Name,
					id:       mdl.ID,
					selected: m.knobs.Model == mdl.ID,
				})
			}
		}
	}

	// 2. Effort Section
	rows = append(rows, engineRow{kind: rowHeader, title: p.T("engines.section_effort", "Effort")})
	if len(currentEng.Efforts) == 0 {
		rows = append(rows, engineRow{
			kind:  rowHeader,
			title: "  " + p.T("engines.no_effort", "{engine} has no effort dial", about("engine", activeEngine)),
		})
	} else {
		for _, eff := range currentEng.Efforts {
			lbl := eff.Label
			if eff.ID == "" {
				lbl = p.T("engines.default", "default")
			}

			rows = append(rows, engineRow{
				kind:     rowEffort,
				title:    "  " + lbl,
				engine:   activeEngine,
				id:       eff.ID,
				selected: m.knobs.Effort == eff.ID,
			})
		}
	}

	// 3. Thinking Section
	rows = append(rows, engineRow{kind: rowHeader, title: p.T("engines.section_thinking", "Thinking Mode")})
	if !currentEng.CanThink {
		rows = append(rows, engineRow{
			kind:  rowHeader,
			title: "  " + p.T("engines.no_thinking", "{engine} has no thinking mode", about("engine", activeEngine)),
		})
	} else {
		rows = append(rows,
			engineRow{kind: rowThinking, title: "  " + p.T("engines.thinking_adaptive", "adaptive (default)"), id: "", selected: m.knobs.Thinking == "" || m.knobs.Thinking == "adaptive"},
			engineRow{kind: rowThinking, title: "  " + p.T("engines.thinking_on", "on"), id: "on", selected: m.knobs.Thinking == "on"},
			engineRow{kind: rowThinking, title: "  " + p.T("engines.thinking_off", "off"), id: "off", selected: m.knobs.Thinking == "off"},
		)
	}

	return rows
}
