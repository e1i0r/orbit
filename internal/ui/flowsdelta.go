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
	case flowFieldIsLoop:
		return m.toggleLoop(), nil
	case flowFieldLoopTurns:
		return m.setLoopTurns(st.cur().Loop.Max + delta), nil
	case flowFieldEngine:
		// The three dials are the build's: see engine_table.go. This one
		// offered every engine a model called sonnet, which is claude's
		// alone, and an effort called default, which internal/task refuses
		// by name before a run starts.
		//
		// They are set on the phase that runs, which for a loop is the
		// phase inside it: internal/flow refuses a phase that is both an
		// engine and a loop.
		st.edited().Engine = nextOption(m.engineNames(), st.edited().Engine, delta)
	case flowFieldModel:
		mdls, _ := m.modelsFor(m.dialEngine(st.edited().Engine))
		st.edited().Model = nextOption(mdls, st.edited().Model, delta)
	case flowFieldEffort:
		effs, _ := m.effortsFor(m.dialEngine(st.edited().Engine))
		st.edited().Effort = nextOption(effs, st.edited().Effort, delta)
	case flowFieldThinking:
		thks := []string{"adaptive", "on", "off"}
		st.edited().Thinking = nextOption(thks, st.edited().Thinking, delta)
	case flowFieldFeedOutput:
		st.edited().FeedOutput = !st.edited().FeedOutput
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
	case flowFieldEngine, flowFieldModel, flowFieldEffort:
		// Enter opens the list rather than stepping one along, because one
		// engine has sixty models and the reader knows which one they want.
		// Left and right still walk them, for the dials that are short.
		return m.openPicker(st.field), nil
	case flowFieldTemplate, flowFieldPhaseSelect, flowFieldThinking,
		flowFieldFeedOutput, flowFieldWait, flowFieldIsLoop:
		return m.handleFlowFieldDelta(1)
	case flowFieldAddPhase:
		st.phases = append(st.phases, flow.Phase{
			// A new phase is born on the window's engine and names no
			// model and no effort: sonnet is claude's model alone, so
			// naming it here breaks the phase on any other engine.
			Name:        fmt.Sprintf("%d-phase", len(st.phases)+1),
			Engine:      st.engine,
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
