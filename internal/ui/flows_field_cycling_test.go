package ui

// flows_field_delta_coverage_test.go is the rest of handleFlowFieldDelta
// and handleFlowFieldAction's switches: the field kinds neither
// flows_more_coverage_test.go's Left/Right walk nor
// flowstpl_click_coverage_test.go's default-field click happened to name.

import "testing"

func TestHandleFlowFieldDeltaEveryCyclingField(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.startCreateFlow()
	m.flows.phases = append(m.flows.phases, m.flows.phases[0])

	m.flows.field = flowFieldPhaseSelect
	before := m.flows.activePhase

	m2, _ := m.handleFlowFieldDelta(1)
	if m2.flows.activePhase == before {
		t.Errorf("expected flowFieldPhaseSelect to move activePhase")
	}

	m.flows.field = flowFieldModel
	beforeModel := m.flows.cur().Model

	m3, _ := m.handleFlowFieldDelta(1)
	if m3.flows.cur().Model == beforeModel {
		t.Errorf("expected flowFieldModel to cycle the model")
	}

	m.flows.field = flowFieldEffort
	beforeEffort := m.flows.cur().Effort

	m4, _ := m.handleFlowFieldDelta(1)
	if m4.flows.cur().Effort == beforeEffort {
		t.Errorf("expected flowFieldEffort to cycle the effort")
	}

	m.flows.field = flowFieldThinking
	beforeThinking := m.flows.cur().Thinking

	m5, _ := m.handleFlowFieldDelta(1)
	if m5.flows.cur().Thinking == beforeThinking {
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
		m, _ := testModel(t, 100, 30)
		m = m.startCreateFlow()
		m.flows.phases = append(m.flows.phases, m.flows.phases[0])
		m.flows.field = f
		// A field action must not panic and must return some model; the
		// per-field effect is already checked directly above and in
		// flows_more_coverage_test.go. This walk exists to reach every
		// case of handleFlowFieldAction's switch at least once.
		if _, cmd := m.handleFlowFieldAction(); cmd != nil && f != flowFieldTemplate {
			t.Errorf("field %d: unexpected cmd %v", f, cmd)
		}
	}
}

// TestHandleFlowFieldActionUnhandledField is a text field, which the
// action switch names no case for: enter on it does nothing.
func TestHandleFlowFieldActionUnhandledField(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.startCreateFlow()
	m.flows.field = flowFieldPrompt
	before := m.flows.cur().Prompt

	m2, cmd := m.handleFlowFieldAction()
	if cmd != nil {
		t.Fatalf("expected no cmd from an unhandled field")
	}

	if m2.flows.cur().Prompt != before {
		t.Errorf("expected the prompt untouched by an action on its own field")
	}
}
