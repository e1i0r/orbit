package ui

// flowsmouse_builder_coverage_test.go is hitFlows's builder-view half: the
// pipeline overview, the field rows and the prompt box, once the reader is
// creating or editing a flow rather than looking at the list.
//
// The line numbers below are hitFlows's own arithmetic, walked by hand for
// a single default phase with an empty prompt — the shape startCreateFlow
// leaves the form in — so a change to that geometry breaks the test that
// depends on it rather than silently drawing under the reader's click.

import "testing"

func builderModel(t *testing.T) Model {
	t.Helper()
	m, _ := testModel(t, 100, 30)
	return m.startCreateFlow()
}

func TestHitFlowsBuilderOverview(t *testing.T) {
	m := builderModel(t)
	base := m.frame.Body.Y

	// The lone phase's own overview row selects it.
	if got := m.hitFlows(10, base+5); got.Kind != TargetFlowItem || got.Field != "select_phase" || got.Phase != 0 {
		t.Errorf("hitFlows on the overview row = %+v, want select_phase 0", got)
	}

	// The gap between the overview and the first field selects nothing.
	if got := m.hitFlows(10, base+6); got.Kind != TargetNone {
		t.Errorf("hitFlows on the overview's blank gap = %+v, want the zero Target", got)
	}
}

func TestHitFlowsBuilderOverviewWithPrompt(t *testing.T) {
	m := builderModel(t)
	m.flows.phases[0].Prompt = "already written"
	base := m.frame.Body.Y

	// A phase with a prompt draws an extra line, so its overview row is
	// two lines tall rather than one.
	for _, dy := range []int{5, 6} {
		if got := m.hitFlows(10, base+dy); got.Kind != TargetFlowItem || got.Field != "select_phase" || got.Phase != 0 {
			t.Errorf("hitFlows at overview row %d = %+v, want select_phase 0", dy, got)
		}
	}
}

func TestHitFlowsBuilderSecondPhase(t *testing.T) {
	m := builderModel(t)
	m.flows.phases = append(m.flows.phases, m.flows.phases[0])
	base := m.frame.Body.Y

	if got := m.hitFlows(10, base+5); got.Phase != 0 {
		t.Errorf("first phase overview row = %+v, want phase 0", got)
	}
	if got := m.hitFlows(10, base+6); got.Kind != TargetFlowItem || got.Field != "select_phase" || got.Phase != 1 {
		t.Errorf("second phase overview row = %+v, want select_phase 1", got)
	}
}

// TestHitFlowsBuilderFields walks every field row hitFlows can name, for
// the single-default-phase shape: overview line 5, blank line 6, fields
// starting at line 7.
func TestHitFlowsBuilderFields(t *testing.T) {
	m := builderModel(t)
	base := m.frame.Body.Y

	cases := []struct {
		dy    int
		field int
	}{
		{7, flowFieldTemplate},
		{8, flowFieldName},
		{9, flowFieldDescription},
		{10, flowFieldPhaseSelect},
		{11, flowFieldPhaseName},
		{12, flowFieldEngine},
		{13, flowFieldModel},
		{14, flowFieldEffort},
		{15, flowFieldThinking},
		{16, flowFieldFeedOutput},
		{17, flowFieldWait},
	}
	for _, c := range cases {
		got := m.hitFlows(10, base+c.dy)
		if got.Kind != TargetFlowItem || got.Field != "" || got.Phase != c.field {
			t.Errorf("hitFlows at field row %d = %+v, want field %d", c.dy, got, c.field)
		}
	}
}

// TestHitFlowsBuilderPromptRowButtons is the prompt row's own x-ranges: the
// paste, autogenerate and clear pills to its right, and the field itself to
// their left.
func TestHitFlowsBuilderPromptRowButtons(t *testing.T) {
	m := builderModel(t)
	y := m.frame.Body.Y + 18

	if got := m.hitFlows(10, y); got.Kind != TargetFlowItem || got.Field != "" || got.Phase != flowFieldPrompt {
		t.Errorf("hitFlows left of the prompt row's pills = %+v, want the prompt field", got)
	}
	if got := m.hitFlows(30, y); got.Field != "paste_prompt" {
		t.Errorf("hitFlows on the paste pill = %+v, want paste_prompt", got)
	}
	if got := m.hitFlows(45, y); got.Field != "autogen_prompt" {
		t.Errorf("hitFlows on the autogenerate pill = %+v, want autogen_prompt", got)
	}
	if got := m.hitFlows(60, y); got.Field != "clear_prompt" {
		t.Errorf("hitFlows on the clear pill = %+v, want clear_prompt", got)
	}
}

// TestHitFlowsBuilderPromptBox is the multi-line box under the prompt row,
// which is still the prompt field wherever inside it the reader clicks.
func TestHitFlowsBuilderPromptBox(t *testing.T) {
	m := builderModel(t)
	base := m.frame.Body.Y
	for _, dy := range []int{19, 20, 21} {
		if got := m.hitFlows(10, base+dy); got.Kind != TargetFlowItem || got.Field != "" || got.Phase != flowFieldPrompt {
			t.Errorf("hitFlows in the prompt box at line %d = %+v, want the prompt field", dy, got)
		}
	}
}

// TestHitFlowsBuilderButtonsRow is add/delete/save, in that left-to-right
// order along the row under the prompt box.
func TestHitFlowsBuilderButtonsRow(t *testing.T) {
	m := builderModel(t)
	y := m.frame.Body.Y + 22

	if got := m.hitFlows(10, y); got.Field != "add_phase" {
		t.Errorf("hitFlows on add_phase = %+v", got)
	}
	if got := m.hitFlows(30, y); got.Field != "del_phase" {
		t.Errorf("hitFlows on del_phase = %+v", got)
	}
	if got := m.hitFlows(50, y); got.Field != "save" {
		t.Errorf("hitFlows on save = %+v", got)
	}
}
