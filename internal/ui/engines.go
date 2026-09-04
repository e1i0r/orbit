package ui

// The engine & model knobs menu: choose model, effort level, and extended
// thinking mode for the run, or view setup steps for an engine not yet configured.

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Knobs is the runtime overrides for an engine's dials.
type Knobs struct {
	Engine   string
	Model    string
	Effort   string
	Thinking string
}

type enginesState struct {
	sel int

	// offset is the first line of the list on show, moved only by
	// keepEngineRowSeen: opencode alone offers sixty-four models, so this
	// list has been taller than the screen since the day it stopped being
	// a shortlist.
	offset int

	// open is which engines are showing their models, where absent means
	// the one in force is open and the others are not. Nothing forces one
	// open or shut: two engines side by side is how a reader compares them.
	open map[string]bool

	showingSetup bool
	setupEngine  string
	fromScreen   screen
}

type engineRowKind int

const (
	rowHeader engineRowKind = iota
	rowEngine
	rowModel
	rowEffort
	rowThinking
)

type engineRow struct {
	kind     engineRowKind
	title    string
	engine   string
	id       string
	selected bool
	disabled bool
	open     bool
	models   int
	setup    []string
}

func (m Model) openEngines() Model {
	m.engines = enginesState{fromScreen: m.screen}
	m.screen = screenEngines

	return m
}

func (m Model) abandonEngines() Model {
	prev := m.engines.fromScreen

	m.engines = enginesState{}
	if prev == screenStart {
		m.screen = screenStart
		return m
	}

	m.screen = screenList

	return m
}

func (m Model) knobChip() string {
	if m.knobs.Engine == "" && m.knobs.Model == "" && m.knobs.Effort == "" && m.knobs.Thinking == "" {
		return ""
	}

	engineName := m.dialEngine(m.knobs.Engine)

	var parts []string

	parts = append(parts, engineName)
	if m.knobs.Model != "" {
		parts = append(parts, m.knobs.Model)
	}

	if m.knobs.Effort != "" {
		parts = append(parts, m.knobs.Effort)
	}

	if m.knobs.Thinking != "" && m.knobs.Thinking != "off" {
		parts = append(parts, "thinking")
	}

	return strings.Join(parts, " · ")
}

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

		open := m.engineOpen(eng.Name)

		rows = append(rows, engineRow{
			kind:     rowEngine,
			title:    eng.Name,
			engine:   eng.Name,
			selected: eng.Name == activeEngine,
			open:     open,
			models:   len(eng.Models),
		})

		if open {
			for _, mdl := range eng.Models {
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

func (m Model) selectableEngineIndices(rows []engineRow) []int {
	var idxs []int

	for i, r := range rows {
		if r.kind != rowHeader {
			idxs = append(idxs, i)
		}
	}

	return idxs
}

func (m Model) enginesKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.engines.showingSetup {
		if key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Quit) || key.Matches(msg, m.keys.Open) {
			m.engines.showingSetup = false
			return m, nil
		}

		return m, nil
	}

	rows := m.collectEngineRows()

	idxs := m.selectableEngineIndices(rows)
	if len(idxs) == 0 {
		if key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Quit) {
			return m.abandonEngines(), nil
		}

		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Quit):
		p := m.opts.Words

		return m.abandonEngines().say(p.T("engines.updated", "updated dials: {chip}",
			about("chip", m.knobChip()))), nil
	case key.Matches(msg, m.keys.Up):
		m.engines.sel--
		if m.engines.sel < 0 {
			m.engines.sel = len(idxs) - 1
		}

		return m.keepEngineRowSeen(), nil
	case key.Matches(msg, m.keys.Down):
		m.engines.sel++
		if m.engines.sel >= len(idxs) {
			m.engines.sel = 0
		}

		return m.keepEngineRowSeen(), nil
	case key.Matches(msg, m.keys.Sideways):
		return m.foldKnob(msg.String() != "left"), nil
	case key.Matches(msg, m.keys.Open), msg.Text == " ":
		selectedRow := rows[idxs[m.engines.sel]]
		return m.applyEngineChoice(selectedRow), nil
	}

	return m, nil
}

func (m Model) applyEngineChoice(selectedRow engineRow) Model {
	if selectedRow.disabled {
		m.engines.showingSetup = true
		m.engines.setupEngine = selectedRow.engine

		return m
	}

	switch selectedRow.kind {
	case rowEngine:
		// An engine already in force has nothing left to choose, so ⏎ on
		// its name folds it instead: the reader is done with its models,
		// or wants them back.
		if m.knobs.Engine == selectedRow.engine {
			return m.foldKnob(!selectedRow.open)
		}

		m.knobs.Engine, m.knobs.Model = selectedRow.engine, ""
		m = m.setOpt("engine", selectedRow.engine).foldEngine(selectedRow.engine, true)
	case rowModel:
		m.knobs.Engine, m.knobs.Model = selectedRow.engine, selectedRow.id
		m = m.setOpt("model", selectedRow.id)
	case rowEffort:
		m.knobs.Effort = selectedRow.id
	case rowThinking:
		m.knobs.Thinking = selectedRow.id
	}

	// Choosing an engine opens its models under it and closes the one that
	// was open, so the list this was picked from is not the list it leaves
	// behind: where the view sits has to be asked again.
	return m.keepEngineRowSeen()
}

// pickEngineRow moves the selection by rows, for the wheel. It stops at
// either end rather than wrapping as the arrows do: a notch that carried the
// reader from the last model back to the top would read as the list jumping
// under their hand.
func (m Model) pickEngineRow(d int) Model {
	n := len(m.selectableEngineIndices(m.collectEngineRows()))
	if n == 0 {
		return m
	}

	m.engines.sel = min(max(m.engines.sel+d, 0), n-1)

	return m.keepEngineRowSeen()
}

func (m Model) setOpt(k, v string) Model {
	m2, _ := m.applySetting(k, v)
	if mod, ok := m2.(Model); ok {
		return mod
	}

	return m
}
