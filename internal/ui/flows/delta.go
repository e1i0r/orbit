package flows

import (
	"fmt"
	"strconv"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
)

func (s State) handleFlowFieldDelta(delta int, e Env) (State, Out) {
	s.ensurePhase()

	switch s.field {
	case flowFieldTemplate:
		tpls := []string{"ninguna", "TDD Fuzz & PR", "TDD Cycle", "Security Audit", "Turbo Fix"}
		s.template = cells.NextOption(tpls, s.template, delta)

		return s.applyFlowTemplate(s.template, e)
	case flowFieldPhaseSelect:
		n := len(s.phases)
		if n > 0 {
			s.activePhase = (s.activePhase + delta + n) % n
		}
	case flowFieldIsLoop:
		return s.toggleLoop(), Out{}
	case flowFieldLoopTurns:
		return s.setLoopTurns(s.cur().Loop.Max + delta), Out{}
	case flowFieldEngine:
		// The three dials are the build's: see engine_table.go. This one
		// offered every engine a model called sonnet, which is claude's
		// alone, and an effort called default, which internal/task refuses
		// by name before a run starts.
		//
		// They are set on the phase that runs, which for a loop is the
		// phase inside it: internal/flow refuses a phase that is both an
		// engine and a loop.
		s.edited().Engine = cells.NextOption(e.Engines(), s.edited().Engine, delta)
	case flowFieldModel:
		mdls, _ := e.Models(cells.OrDef(s.edited().Engine, e.Engine))
		s.edited().Model = cells.NextOption(mdls, s.edited().Model, delta)
	case flowFieldEffort:
		effs, _ := e.Efforts(cells.OrDef(s.edited().Engine, e.Engine))
		s.edited().Effort = cells.NextOption(effs, s.edited().Effort, delta)
	case flowFieldThinking:
		thks := []string{"adaptive", "on", "off"}
		s.edited().Thinking = cells.NextOption(thks, s.edited().Thinking, delta)
	case flowFieldFeedOutput:
		s.edited().FeedOutput = !s.edited().FeedOutput
	case flowFieldWait:
		s.cur().Wait = !s.cur().Wait
	}

	return s, Out{}
}

func (s State) handleFlowFieldAction(e Env) (State, Out) {
	s.ensurePhase()

	p := e.Words

	switch s.field {
	case flowFieldEngine, flowFieldModel, flowFieldEffort:
		// Enter opens the list rather than stepping one along, because one
		// engine has sixty models and the reader knows which one they want.
		// Left and right still walk them, for the dials that are short.
		return s.openPicker(s.field, e), Out{}
	case flowFieldTemplate, flowFieldPhaseSelect, flowFieldThinking,
		flowFieldFeedOutput, flowFieldWait, flowFieldIsLoop:
		return s.handleFlowFieldDelta(1, e)
	case flowFieldAddPhase:
		s.phases = append(s.phases, flow.Phase{
			// A new phase is born on the window's engine and names no
			// model and no effort: sonnet is claude's model alone, so
			// naming it here breaks the phase on any other engine.
			Name:        fmt.Sprintf("%d-phase", len(s.phases)+1),
			Engine:      s.engine,
			Thinking:    "adaptive",
			FeedOutput:  true,
			Permissions: []string{"repo"},
		})
		s.activePhase = len(s.phases) - 1
		s.field = flowFieldPhaseName

		return s, said(p.T("flows.phase_added", "phase {n} added",
			about("n", strconv.Itoa(len(s.phases)))))
	case flowFieldDelPhase:
		if len(s.phases) <= 1 {
			return s, said(p.T("flows.min_phases_required",
				"the flow must have at least one phase"))
		}

		idx := s.activePhase

		s.phases = append(s.phases[:idx], s.phases[idx+1:]...)
		if s.activePhase >= len(s.phases) {
			s.activePhase = len(s.phases) - 1
		}

		return s, Out{Said: p.T("flows.phase_deleted", "phase deleted")}
	case flowFieldSave:
		return s.saveCustomFlow(e)
	}

	return s, Out{}
}
