package ui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

const (
	flowFieldTemplate = iota
	flowFieldName
	flowFieldPhaseSelect
	flowFieldPhaseName
	flowFieldEngine
	flowFieldModel
	flowFieldEffort
	flowFieldThinking
	flowFieldFeedOutput
	flowFieldWait
	flowFieldPrompt
	flowFieldAddPhase
	flowFieldDelPhase
	flowFieldSave
	flowFieldCount
)

type flowsState struct {
	sel            int
	creating       bool
	confirmDiscard bool
	confirmDelete  bool
	field          int
	template       string
	flowName       string
	activePhase    int
	phases         []flow.Phase
}

func (st *flowsState) ensurePhase() {
	if len(st.phases) == 0 {
		st.phases = []flow.Phase{
			{Name: "1-implement", Engine: "claude", Model: "sonnet", Effort: "default", Thinking: "adaptive", Permissions: []string{"repo"}},
		}
		st.activePhase = 0
	}
	if st.activePhase < 0 {
		st.activePhase = 0
	}
	if st.activePhase >= len(st.phases) {
		st.activePhase = len(st.phases) - 1
	}
}

func (st *flowsState) cur() *flow.Phase {
	st.ensurePhase()
	return &st.phases[st.activePhase]
}

func (m Model) openFlows() Model {
	m.screen = screenFlows
	m.flows = flowsState{
		template: "ninguna",
	}
	m.flows.ensurePhase()
	return m
}

func (m Model) abandonFlows() Model {
	m.flows = flowsState{}
	m.screen = screenList
	return m
}

func (m Model) flowsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.flows.creating {
		return m.flowsFormKey(msg)
	}
	if m.flows.confirmDelete {
		switch {
		case msg.Text == "y" || msg.Text == "Y" || msg.Text == "s" || msg.Text == "S" || key.Matches(msg, m.keys.Open):
			return m.confirmDeleteFlow()
		default:
			m.flows.confirmDelete = false
			return m.say("borrado cancelado"), nil
		}
	}
	list := flow.List(m.opts.Flows)
	switch {
	case key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Quit):
		return m.abandonFlows(), nil
	case msg.Text == "n" || msg.Text == "N" || key.Matches(msg, m.keys.Start):
		return m.startCreateFlow(), nil
	case msg.Text == "e" || msg.Text == "E" || key.Matches(msg, m.keys.Open):
		return m.editSelectedFlow()
	case msg.Text == "d" || msg.Text == "D" || msg.Code == tea.KeyDelete:
		return m.deleteSelectedFlow()
	case key.Matches(msg, m.keys.Up):
		if len(list) > 0 {
			m.flows.sel--
			if m.flows.sel < 0 {
				m.flows.sel = len(list) - 1
			}
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if len(list) > 0 {
			m.flows.sel++
			if m.flows.sel >= len(list) {
				m.flows.sel = 0
			}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) flowsFormKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	st := &m.flows
	st.ensurePhase()
	if st.confirmDiscard {
		switch {
		case msg.Text == "y" || msg.Text == "Y" || msg.Text == "s" || msg.Text == "S" || key.Matches(msg, m.keys.Open) || key.Matches(msg, m.keys.Back):
			st.creating = false
			st.confirmDiscard = false
			return m.say("cambios descartados"), nil
		default:
			st.confirmDiscard = false
			return m.say("edición reanudada"), nil
		}
	}

	isText := st.field == flowFieldName || st.field == flowFieldPhaseName || st.field == flowFieldPrompt
	switch {
	case key.Matches(msg, m.keys.Back):
		if st.flowName != "" || len(st.phases) > 1 || st.cur().Prompt != "" {
			st.confirmDiscard = true
			return m.say("¿Descartar cambios del flujo? [y] sí / [n] no (o presiona Esc otra vez)"), nil
		}
		st.creating = false
		return m, nil
	case key.Matches(msg, m.keys.NextTab) || key.Matches(msg, m.keys.Down):
		st.field = (st.field + 1) % flowFieldCount
		return m, nil
	case key.Matches(msg, m.keys.PrevTab) || key.Matches(msg, m.keys.Up):
		st.field = (st.field - 1 + flowFieldCount) % flowFieldCount
		return m, nil
	case !isText && (msg.Code == tea.KeyLeft || msg.Code == tea.KeyRight):
		delta := 1
		if msg.Code == tea.KeyLeft {
			delta = -1
		}
		return m.handleFlowFieldDelta(delta)
	case key.Matches(msg, m.keys.Open) || (!isText && msg.Text == " "):
		return m.handleFlowFieldAction()
	case msg.Code == tea.KeyBackspace:
		switch st.field {
		case flowFieldName:
			st.flowName = trimLastRune(st.flowName)
		case flowFieldPhaseName:
			st.cur().Name = trimLastRune(st.cur().Name)
		case flowFieldPrompt:
			st.cur().Prompt = trimLastRune(st.cur().Prompt)
		}
		return m, nil
	}

	if msg.Text != "" {
		switch st.field {
		case flowFieldName:
			st.flowName += msg.Text
		case flowFieldPhaseName:
			st.cur().Name += msg.Text
		case flowFieldPrompt:
			st.cur().Prompt += msg.Text
		}
	}
	return m, nil
}

func (m Model) handleFlowFieldDelta(delta int) (Model, tea.Cmd) {
	st := &m.flows
	st.ensurePhase()
	switch st.field {
	case flowFieldTemplate:
		tpls := []string{"ninguna", "TDD Cycle", "Security Audit", "Turbo Fix"}
		st.template = nextOption(tpls, st.template, delta)
		return m.applyFlowTemplate(st.template)
	case flowFieldPhaseSelect:
		if len(st.phases) > 0 {
			st.activePhase = (st.activePhase + delta + len(st.phases)) % len(st.phases)
		}
	case flowFieldEngine:
		engs := []string{"claude", "codex", "opencode"}
		st.cur().Engine = nextOption(engs, orDef(st.cur().Engine, "claude"), delta)
	case flowFieldModel:
		mdls := []string{"sonnet", "opus", "haiku", "default"}
		st.cur().Model = nextOption(mdls, orDef(st.cur().Model, "sonnet"), delta)
	case flowFieldEffort:
		effs := []string{"default", "low", "medium", "high", "xhigh", "max"}
		st.cur().Effort = nextOption(effs, orDef(st.cur().Effort, "default"), delta)
	case flowFieldThinking:
		thks := []string{"adaptive", "on", "off"}
		st.cur().Thinking = nextOption(thks, orDef(st.cur().Thinking, "adaptive"), delta)
	case flowFieldFeedOutput:
		st.cur().FeedOutput = !st.cur().FeedOutput
	case flowFieldWait:
		st.cur().Wait = !st.cur().Wait
	}
	return m, nil
}

func (m Model) handleFlowFieldAction() (Model, tea.Cmd) {
	st := &m.flows
	st.ensurePhase()
	switch st.field {
	case flowFieldTemplate:
		return m.handleFlowFieldDelta(1)
	case flowFieldPhaseSelect:
		return m.handleFlowFieldDelta(1)
	case flowFieldEngine:
		return m.handleFlowFieldDelta(1)
	case flowFieldModel:
		return m.handleFlowFieldDelta(1)
	case flowFieldEffort:
		return m.handleFlowFieldDelta(1)
	case flowFieldThinking:
		return m.handleFlowFieldDelta(1)
	case flowFieldFeedOutput:
		return m.handleFlowFieldDelta(1)
	case flowFieldWait:
		return m.handleFlowFieldDelta(1)
	case flowFieldAddPhase:
		st.phases = append(st.phases, flow.Phase{
			Name:        fmt.Sprintf("%d-fase", len(st.phases)+1),
			Engine:      "claude",
			Model:       "sonnet",
			Effort:      "default",
			Thinking:    "adaptive",
			FeedOutput:  true,
			Permissions: []string{"repo"},
		})
		st.activePhase = len(st.phases) - 1
		st.field = flowFieldPhaseName
		return m.say(fmt.Sprintf("fase %d añadida", len(st.phases))), nil
	case flowFieldDelPhase:
		if len(st.phases) <= 1 {
			return m.say("el flujo debe tener al menos una fase"), nil
		}
		idx := st.activePhase
		st.phases = append(st.phases[:idx], st.phases[idx+1:]...)
		if st.activePhase >= len(st.phases) {
			st.activePhase = len(st.phases) - 1
		}
		return m.say("fase eliminada"), nil
	case flowFieldSave:
		return m.saveCustomFlow()
	}
	return m, nil
}

func (m Model) startCreateFlow() Model {
	m.flows.creating = true
	m.flows.confirmDiscard = false
	m.flows.confirmDelete = false
	m.flows.field = 0
	m.flows.template = "ninguna"
	m.flows.flowName = ""
	m.flows.phases = []flow.Phase{
		{Name: "1-implement", Engine: "claude", Model: "sonnet", Effort: "default", Thinking: "adaptive", Permissions: []string{"repo"}},
	}
	m.flows.activePhase = 0
	return m
}
