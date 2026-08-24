package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

const (
	flowFieldName = iota
	flowFieldPhaseName
	flowFieldEngine
	flowFieldModel
	flowFieldEffort
	flowFieldThinking
	flowFieldFeedOutput
	flowFieldWait
	flowFieldPrompt
	flowFieldAddPhase
	flowFieldSave
	flowFieldCount
)

type flowsState struct {
	sel        int
	creating   bool
	field      int
	flowName   string
	phases     []flow.Phase
	phaseName  string
	engine     string
	model      string
	effort     string
	thinking   string
	feedOutput bool
	wait       bool
	prompt     string
}

func (m Model) openFlows() Model {
	m.screen = screenFlows
	m.flows = flowsState{
		engine:   "claude",
		model:    "sonnet",
		effort:   "default",
		thinking: "adaptive",
	}
	return m
}

func (m Model) abandonFlows() Model {
	m.flows = flowsState{}
	m.screen = screenList
	return m
}

func (st *flowsState) currentPhase() flow.Phase {
	pName := strings.TrimSpace(st.phaseName)
	if pName == "" {
		pName = fmt.Sprintf("phase-%d", len(st.phases)+1)
	}
	mdl := st.model
	if mdl == "default" {
		mdl = ""
	}
	eff := st.effort
	if eff == "default" {
		eff = ""
	}
	thk := st.thinking
	if thk == "adaptive" {
		thk = ""
	}
	return flow.Phase{
		Name:        pName,
		Engine:      st.engine,
		Model:       mdl,
		Effort:      eff,
		Thinking:    thk,
		Prompt:      strings.TrimSpace(st.prompt),
		FeedOutput:  st.feedOutput,
		Wait:        st.wait,
		Permissions: []string{"repo"},
	}
}

func (m Model) flowsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.flows.creating {
		return m.flowsFormKey(msg)
	}
	list := flow.List(m.opts.Flows)
	switch {
	case key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Quit):
		return m.abandonFlows(), nil
	case msg.Text == "n" || msg.Text == "N" || key.Matches(msg, m.keys.Start):
		m.flows.creating = true
		m.flows.field = 0
		m.flows.flowName = ""
		m.flows.phases = nil
		m.flows.phaseName = "implement"
		m.flows.engine = "claude"
		m.flows.model = "sonnet"
		m.flows.effort = "default"
		m.flows.thinking = "adaptive"
		m.flows.feedOutput = false
		m.flows.wait = false
		m.flows.prompt = ""
		return m, nil
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
	switch {
	case key.Matches(msg, m.keys.Back):
		st.creating = false
		return m, nil
	case key.Matches(msg, m.keys.NextTab) || key.Matches(msg, m.keys.Down):
		st.field = (st.field + 1) % flowFieldCount
		return m, nil
	case key.Matches(msg, m.keys.PrevTab) || key.Matches(msg, m.keys.Up):
		st.field = (st.field - 1 + flowFieldCount) % flowFieldCount
		return m, nil
	case key.Matches(msg, m.keys.Open) || msg.Text == " ":
		return m.handleFlowFieldAction()
	case msg.Code == tea.KeyBackspace:
		switch st.field {
		case flowFieldName:
			st.flowName = trimLastRune(st.flowName)
		case flowFieldPhaseName:
			st.phaseName = trimLastRune(st.phaseName)
		case flowFieldPrompt:
			st.prompt = trimLastRune(st.prompt)
		}
		return m, nil
	}
	if msg.Text != "" {
		switch st.field {
		case flowFieldName:
			st.flowName += msg.Text
		case flowFieldPhaseName:
			st.phaseName += msg.Text
		case flowFieldPrompt:
			st.prompt += msg.Text
		}
	}
	return m, nil
}

func (m Model) handleFlowFieldAction() (Model, tea.Cmd) {
	st := &m.flows
	switch st.field {
	case flowFieldEngine:
		engs := []string{"claude", "codex", "opencode"}
		st.engine = nextOption(engs, st.engine, 1)
	case flowFieldModel:
		mdls := []string{"sonnet", "opus", "haiku", "default"}
		st.model = nextOption(mdls, st.model, 1)
	case flowFieldEffort:
		effs := []string{"default", "low", "medium", "high", "xhigh", "max"}
		st.effort = nextOption(effs, st.effort, 1)
	case flowFieldThinking:
		thks := []string{"adaptive", "on", "off"}
		st.thinking = nextOption(thks, st.thinking, 1)
	case flowFieldFeedOutput:
		st.feedOutput = !st.feedOutput
	case flowFieldWait:
		st.wait = !st.wait
	case flowFieldAddPhase:
		st.phases = append(st.phases, st.currentPhase())
		st.phaseName = fmt.Sprintf("phase-%d", len(st.phases)+1)
		st.prompt = ""
		st.field = flowFieldPhaseName
		return m.say(fmt.Sprintf("fase %d añadida", len(st.phases))), nil
	case flowFieldSave:
		return m.saveCustomFlow()
	}
	return m, nil
}

func (m Model) saveCustomFlow() (Model, tea.Cmd) {
	st := &m.flows
	name := strings.TrimSpace(st.flowName)
	if name == "" {
		return m.say("indica un nombre para el flujo"), nil
	}
	phases := st.phases
	if len(phases) == 0 && st.phaseName != "" {
		phases = append(phases, st.currentPhase())
	}
	if len(phases) == 0 {
		return m.say("el flujo debe tener al menos una fase"), nil
	}
	fl := flow.Flow{
		Name:   name,
		Phases: phases,
	}
	if err := fl.Validate(); err != nil {
		return m.say(err.Error()), nil
	}

	dir := ""
	if m.opts.Flows != nil {
		dir = m.opts.Flows.FlowDir()
	}
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".orbit", "flows")
	}
	_ = os.MkdirAll(dir, 0755)
	data, err := json.MarshalIndent(fl, "", "  ")
	if err != nil {
		return m.say(err.Error()), nil
	}
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return m.say(err.Error()), nil
	}
	m.flows.creating = false
	m.flows.phases = nil
	m.flows.flowName = ""
	p := m.opts.Words
	return m.say(p.T("flows.saved", "flow {name} saved", about("name", name))), nil
}

func (m Model) startCreateFlow() Model {
	m.flows.creating = true
	m.flows.field = 0
	m.flows.flowName = ""
	m.flows.phases = nil
	m.flows.phaseName = "implement"
	m.flows.engine = "claude"
	m.flows.model = "sonnet"
	m.flows.effort = "default"
	m.flows.thinking = "adaptive"
	m.flows.feedOutput = false
	m.flows.wait = false
	m.flows.prompt = ""
	return m
}

func (m Model) handleFlowClick(t Target) (tea.Model, tea.Cmd) {
	if t.Field == "create" {
		return m.startCreateFlow(), nil
	}
	if t.Field == "add_phase" {
		m.flows.field = flowFieldAddPhase
		return m.handleFlowFieldAction()
	}
	if t.Field == "save" {
		m.flows.field = flowFieldSave
		return m.handleFlowFieldAction()
	}
	m.flows.field = t.Phase
	return m.handleFlowFieldAction()
}
