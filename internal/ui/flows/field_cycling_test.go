package flows

// flows_field_delta_coverage_test.go is the rest of handleFlowFieldDelta
// and handleFlowFieldAction's switches: the field kinds neither
// flows_more_coverage_test.go's Left/Right walk nor
// flowstpl_click_coverage_test.go's default-field click happened to name.

import "testing"

func TestHandleFlowFieldDeltaEveryCyclingField(t *testing.T) {
	s, e := designer(t)
	s = s.startCreateFlow(e).OnFields()
	s.phases = append(s.phases, s.phases[0])

	s.field = flowFieldPhaseSelect
	before := s.activePhase

	m2, _ := s.handleFlowFieldDelta(1, e)
	if m2.activePhase == before {
		t.Errorf("expected flowFieldPhaseSelect to move activePhase")
	}

	s.field = flowFieldModel
	beforeModel := s.cur().Model

	m3, _ := s.handleFlowFieldDelta(1, e)
	if m3.cur().Model == beforeModel {
		t.Errorf("expected flowFieldModel to cycle the model")
	}

	s.field = flowFieldEffort
	beforeEffort := s.cur().Effort

	m4, _ := s.handleFlowFieldDelta(1, e)
	if m4.cur().Effort == beforeEffort {
		t.Errorf("expected flowFieldEffort to cycle the effort")
	}

	s.field = flowFieldThinking
	beforeThinking := s.cur().Thinking

	m5, _ := s.handleFlowFieldDelta(1, e)
	if m5.cur().Thinking == beforeThinking {
		t.Errorf("expected flowFieldThinking to cycle the thinking mode")
	}
}

// TestHandleFlowFieldActionDelegatesEveryCyclingField is every field whose
// action key just cycles it forward by one, the way pressing enter on a
// combo pill does instead of pressing right.
func TestHandleFlowFieldActionDelegatesEveryCyclingField(t *testing.T) {
	fields := []int{
		flowFieldTemplate,
		flowFieldPhaseSelect,
		flowFieldModel,
		flowFieldEffort,
		flowFieldThinking,
		flowFieldWait,
	}
	for _, f := range fields {
		s, e := designer(t)
		s = s.startCreateFlow(e).OnFields()
		s.phases = append(s.phases, s.phases[0])
		s.field = f
		// A field action must not panic and must return some model; the
		// per-field effect is already checked directly above and in
		// flows_more_coverage_test.go. This walk exists to reach every
		// case of handleFlowFieldAction's switch at least once.
		if _, out := s.handleFlowFieldAction(e); out.Cmd != nil && f != flowFieldTemplate {
			t.Errorf("field %d: unexpected cmd %v", f, out.Cmd)
		}
	}
}

// TestHandleFlowFieldActionUnhandledField is a text field, which the
// action switch names no case for: enter on it does nothing.
func TestHandleFlowFieldActionUnhandledField(t *testing.T) {
	s, e := designer(t)
	s = s.startCreateFlow(e).OnFields()
	s.field = flowFieldPrompt
	before := s.cur().Prompt

	m2, out := s.handleFlowFieldAction(e)
	if out.Cmd != nil {
		t.Fatalf("expected no cmd from an unhandled field")
	}

	if m2.cur().Prompt != before {
		t.Errorf("expected the prompt untouched by an action on its own field")
	}
}
