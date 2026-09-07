package ui

// Where the window meets the screens that are packages of their own.
//
// One file rather than one per screen, and nothing in it decides anything:
// it builds the little world a screen was written against, and it does what
// the screen asked for when it hands the answer back. A branch in here would
// be a decision living one package away from the one that made it, and the
// first bug it caused would be looked for in the wrong place.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/flows"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/settings"
)

// settingsEnv is the world the settings screen was written against.
func (m Model) settingsEnv() settings.Env {
	return settings.Env{
		Words: m.opts.Words,
		Keys:  m.keys,
		Store: m.opts.Settings,
		Dials: settings.Dials{
			Engine:   m.knobs.Engine,
			Model:    m.knobs.Model,
			Effort:   m.knobs.Effort,
			Thinking: m.knobs.Thinking,
		},
		Engines: m.engineNames,
		Models:  m.modelsFor,
		Efforts: m.effortsFor,
		Flows:   func() []string { return flow.Names(m.opts.Flows) },
	}
}

// tookSettings does what the settings screen asked the window for.
func (m Model) tookSettings(out settings.Out) (tea.Model, tea.Cmd) {
	if out.Dials != nil {
		m.knobs.Engine = out.Dials.Engine
		m.knobs.Model = out.Dials.Model
		m.knobs.Effort = out.Dials.Effort
		m.knobs.Thinking = out.Dials.Thinking
	}

	if out.Close {
		m.settings = settings.State{}
		m.screen = screenList
	}

	if out.Said != "" {
		m = m.say(out.Said)
	}

	cmd := out.Cmd
	if out.Lang != "" {
		lang := out.Lang
		cmd = tea.Batch(cmd, func() tea.Msg { return languageMsg{Lang: lang} })
	}

	return m, cmd
}

// openSettings brings the screen up with the flows it offers already read.
func (m Model) openSettings() Model {
	m.screen = screenSettings
	m.settings = settings.Open(m.settingsEnv())

	return m
}

// settingsKey hands one keystroke to the screen and does what it asked.
func (m Model) settingsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, out := m.settings.Key(msg, m.settingsEnv())
	m.settings = next

	return m.tookSettings(out)
}

// applySetting writes one setting down, for a click on a pill and for the
// engine screen, which sets the engine and its model from its own list.
func (m Model) applySetting(name, val string) (tea.Model, tea.Cmd) {
	return m.tookSettings(settings.Apply(name, val, m.settingsEnv()))
}

// cycleSetting turns the chosen row's dial by one.
func (m Model) cycleSetting(delta int) (tea.Model, tea.Cmd) {
	return m.tookSettings(m.settings.Cycle(delta, m.settingsEnv()))
}

// settingRowsList is the table as the mouse and the tip need it.
func (m Model) settingRowsList() []settings.Row {
	return m.settings.Rows(m.settingsEnv())
}

// settingsRows is the screen drawn.
func (m Model) settingsRows(h, w int) []string {
	return m.settings.View(h, w, m.settingsEnv())
}

// flowsEnv is the world the flow designer was written against.
func (m Model) flowsEnv() flows.Env {
	return flows.Env{
		Words:   m.opts.Words,
		Keys:    m.keys,
		Frame:   m.frame,
		Flows:   m.opts.Flows,
		Now:     m.now,
		Engine:  m.dialEngine(""),
		Spinner: m.spinner,
		Draft:   m.opts.Draft,
		Engines: m.engineNames,
		Models:  m.modelsFor,
		Efforts: m.effortsFor,
	}
}

// tookFlows does what the designer asked the window for.
func (m Model) tookFlows(next flows.State, out flows.Out) (tea.Model, tea.Cmd) {
	m.flows = next

	if out.Chose != "" {
		m.compose.Write(out.Chose)
	}

	if out.Leave {
		m.screen = backTo(out.Back)
		if out.Back == flows.FromCompose {
			m.compose.Refresh(m.opts.Flows)
		}
	}

	if out.Said != "" {
		m = m.say(out.Said)
	}

	cmd := out.Cmd
	if out.Waiting {
		next, frame := m.nextFrame()
		m = next
		cmd = tea.Batch(cmd, frame)
	}

	return m, cmd
}

// backTo is the screen a reader leaving the designer goes back to.
func backTo(from flows.From) screen {
	switch from {
	case flows.FromCompose:
		return screenCompose
	case flows.FromStart:
		return screenStart
	default:
		return screenList
	}
}

// cameFrom is the same answer the other way round, for opening the designer.
func cameFrom(s screen) flows.From {
	switch s {
	case screenCompose:
		return flows.FromCompose
	case screenStart:
		return flows.FromStart
	default:
		return flows.FromBoard
	}
}

// openFlows brings the designer up over whatever screen asked for it.
func (m Model) openFlows() Model {
	m.flows = flows.Open(cameFrom(m.screen), m.flowsEnv())
	m.screen = screenFlows

	return m
}

// openFlowPreview opens one flow read-only, which is what the compose form
// asks for when somebody wants to see what a flow does before choosing it.
func (m Model) openFlowPreview(name string) Model {
	m.flows = flows.Preview(name, cameFrom(m.screen), m.flowsEnv())
	m.screen = screenFlows

	return m
}

// flowsKey hands one keystroke to the designer.
func (m Model) flowsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	next, out := m.flows.Key(msg, m.flowsEnv())

	return m.tookFlows(next, out)
}

// handleFlowClick hands one click to the designer.
func (m Model) handleFlowClick(t point.Target) (tea.Model, tea.Cmd) {
	next, out := m.flows.Click(t, m.flowsEnv())

	return m.tookFlows(next, out)
}

// draftedFlow takes the engine's answer into the form.
func (m Model) draftedFlow(msg flows.DraftedMsg) (tea.Model, tea.Cmd) {
	next, out := m.flows.Took(msg, m.flowsEnv())

	return m.tookFlows(next, out)
}

// hitFlows is what the designer has at that cell.
func (m Model) hitFlows(x, y int) point.Target {
	return m.flows.Hit(x, y, m.flowsEnv())
}

// flowsRows is the designer drawn.
func (m Model) flowsRows(h, w int) []string {
	return m.flows.View(h, w, m.flowsEnv())
}
