package ui

import (
	"fmt"
	"strconv"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

func (m Model) handleFlowFieldDelta(delta int) (Model, tea.Cmd) {
	st := &m.flows
	st.ensurePhase()

	switch st.field {
	case flowFieldTemplate:
		tpls := []string{"ninguna", "TDD Fuzz & PR", "TDD Cycle", "Security Audit", "Turbo Fix"}
		st.template = nextOption(tpls, st.template, delta)

		return m.applyFlowTemplate(st.template)
	case flowFieldPhaseSelect:
		n := len(st.phases)
		if n > 0 {
			st.activePhase = (st.activePhase + delta + n) % n
		}
	case flowFieldEngine:
		engs := []string{"claude", "codex", "opencode"}
		st.cur().Engine = nextOption(engs, st.cur().Engine, delta)
	case flowFieldModel:
		mdls := []string{"sonnet", "opus", "haiku", "default"}
		st.cur().Model = nextOption(mdls, st.cur().Model, delta)
	case flowFieldEffort:
		effs := []string{"default", "low", "medium", "high", "xhigh", "max"}
		st.cur().Effort = nextOption(effs, st.cur().Effort, delta)
	case flowFieldThinking:
		thks := []string{"adaptive", "on", "off"}
		st.cur().Thinking = nextOption(thks, st.cur().Thinking, delta)
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

	p := m.opts.Words

	switch st.field {
	case flowFieldTemplate, flowFieldPhaseSelect, flowFieldEngine,
		flowFieldModel, flowFieldEffort, flowFieldThinking,
		flowFieldFeedOutput, flowFieldWait:
		return m.handleFlowFieldDelta(1)
	case flowFieldAddPhase:
		st.phases = append(st.phases, flow.Phase{
			Name:        fmt.Sprintf("%d-phase", len(st.phases)+1),
			Engine:      "claude",
			Model:       "sonnet",
			Effort:      "default",
			Thinking:    "adaptive",
			FeedOutput:  true,
			Permissions: []string{"repo"},
		})
		st.activePhase = len(st.phases) - 1
		st.field = flowFieldPhaseName

		return m.say(p.T("flows.phase_added", "phase {n} added",
			about("n", strconv.Itoa(len(st.phases))))), nil
	case flowFieldDelPhase:
		if len(st.phases) <= 1 {
			return m.say(p.T("flows.min_phases_required",
				"the flow must have at least one phase")), nil
		}

		idx := st.activePhase

		st.phases = append(st.phases[:idx], st.phases[idx+1:]...)
		if st.activePhase >= len(st.phases) {
			st.activePhase = len(st.phases) - 1
		}

		return m.say(p.T("flows.phase_deleted", "phase deleted")), nil
	case flowFieldSave:
		return m.saveCustomFlow()
	}

	return m, nil
}
