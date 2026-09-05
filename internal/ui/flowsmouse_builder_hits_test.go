package ui

// hitFlows's builder-view half: the pipeline overview, the field rows and
// the boxes, once the reader is creating or editing a flow rather than
// looking at the list.
//
// The rows are found rather than written down. This test held a table of
// line numbers walked by hand, which is the same table hitFlows itself held
// — so the two agreed about a layout neither of them was reading. Both now
// read builderView, and a row that moves moves in the test with it.

import "testing"

func builderModel(t *testing.T) Model {
	t.Helper()
	m, _ := testModel(t, 100, 36)

	return m.startCreateFlow().onFields()
}

// rowOfField is the screen row the given field was drawn on, and the last of
// them when the field is a box several rows tall.
func rowOfField(t *testing.T, m Model, field int) int {
	t.Helper()

	lines, start := m.builderView(m.frame.Body.H, m.frame.Body.W)

	for i, l := range lines {
		if l.field == field {
			return m.frame.Body.Y + i - start
		}
	}

	t.Fatalf("no row was drawn for field %d", field)

	return 0
}

// rowsOf is every screen row the given predicate drew.
func rowsOf(m Model, keep func(builderLine) bool) []int {
	lines, start := m.builderView(m.frame.Body.H, m.frame.Body.W)

	var out []int

	for i, l := range lines {
		if keep(l) {
			out = append(out, m.frame.Body.Y+i-start)
		}
	}

	return out
}

func TestHitFlowsBuilderOverview(t *testing.T) {
	m := builderModel(t)

	rows := rowsOf(m, func(l builderLine) bool { return l.phase == 0 })
	if len(rows) != 1 {
		t.Fatalf("the lone phase drew %d overview rows", len(rows))
	}

	got := m.hitFlows(10, rows[0])
	if got.Kind != TargetFlowItem || got.Field != "select_phase" || got.Phase != 0 {
		t.Errorf("hitFlows on the overview row = %+v, want select_phase 0", got)
	}

	// The screen's own title selects nothing.
	if got := m.hitFlows(10, m.frame.Body.Y); got.Kind != TargetNone {
		t.Errorf("hitFlows on the title = %+v, want the zero Target", got)
	}
}

func TestHitFlowsBuilderOverviewWithPrompt(t *testing.T) {
	m := builderModel(t)
	m.flows.phases[0].Prompt = "already written"

	// A phase with a prompt draws an extra line, so its overview row is
	// two lines tall rather than one, and both select it.
	rows := rowsOf(m, func(l builderLine) bool { return l.phase == 0 })
	if len(rows) != 2 {
		t.Fatalf("a phase with a prompt drew %d overview rows, want 2", len(rows))
	}

	for _, y := range rows {
		got := m.hitFlows(10, y)
		if got.Kind != TargetFlowItem || got.Field != "select_phase" || got.Phase != 0 {
			t.Errorf("hitFlows at overview row %d = %+v, want select_phase 0", y, got)
		}
	}
}

func TestHitFlowsBuilderSecondPhase(t *testing.T) {
	m := builderModel(t)
	m.flows.phases = append(m.flows.phases, m.flows.phases[0])

	for phase := range 2 {
		rows := rowsOf(m, func(l builderLine) bool { return l.phase == phase })
		if len(rows) == 0 {
			t.Fatalf("phase %d drew no overview row", phase)
		}

		got := m.hitFlows(10, rows[0])
		if got.Kind != TargetFlowItem || got.Field != "select_phase" || got.Phase != phase {
			t.Errorf("overview row of phase %d = %+v", phase, got)
		}
	}
}

// TestHitFlowsBuilderFields is every field of the form, clicked where it was
// drawn.
func TestHitFlowsBuilderFields(t *testing.T) {
	m := builderModel(t)

	for _, field := range m.flows.fieldsShown() {
		if field == flowFieldAddPhase || field == flowFieldDelPhase || field == flowFieldSave {
			continue // three fields on one row, told apart by x below
		}

		got := m.hitFlows(10, rowOfField(t, m, field))
		if got.Kind != TargetFlowItem || got.Field != "" || got.Phase != field {
			t.Errorf("hitFlows on field %d = %+v", field, got)
		}
	}
}

// TestHitFlowsBuilderLoopFields is the two fields that exist only while the
// phase repeats, which is the whole reason the layout is read rather than
// written down.
func TestHitFlowsBuilderLoopFields(t *testing.T) {
	m := builderModel(t)
	m = m.toggleLoop()

	for _, field := range []int{flowFieldLoopTurns, flowFieldLoopUntil} {
		got := m.hitFlows(10, rowOfField(t, m, field))
		if got.Kind != TargetFlowItem || got.Phase != field {
			t.Errorf("hitFlows on loop field %d = %+v", field, got)
		}
	}
}

// TestHitFlowsBuilderPromptRowButtons is the prompt row's own x-ranges: the
// paste, autogenerate and clear pills to its right, and the field itself to
// their left.
func TestHitFlowsBuilderPromptRowButtons(t *testing.T) {
	m := builderModel(t)

	lines, start := m.builderView(m.frame.Body.H, m.frame.Body.W)

	y := 0

	for i, l := range lines {
		if l.head && l.field == flowFieldPrompt {
			y = m.frame.Body.Y + i - start
		}
	}

	got := m.hitFlows(10, y)
	if got.Kind != TargetFlowItem || got.Field != "" || got.Phase != flowFieldPrompt {
		t.Errorf("hitFlows left of the prompt row's pills = %+v, want the prompt field", got)
	}

	if got := m.hitFlows(29, y); got.Field != "paste_prompt" {
		t.Errorf("hitFlows on the paste pill = %+v, want paste_prompt", got)
	}

	if got := m.hitFlows(43, y); got.Field != "autogen_prompt" {
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
	m.flows.field = flowFieldPrompt
	m = m.followField()

	rows := rowsOf(m, func(l builderLine) bool { return l.field == flowFieldPrompt && !l.head })
	if len(rows) < 3 {
		t.Fatalf("the prompt box drew %d rows, want its frame and something inside it", len(rows))
	}

	for _, y := range rows {
		got := m.hitFlows(10, y)
		if got.Kind != TargetFlowItem || got.Field != "" || got.Phase != flowFieldPrompt {
			t.Errorf("hitFlows in the prompt box at %d = %+v, want the prompt field", y, got)
		}
	}
}

// TestHitFlowsBuilderButtonsRow is add/delete/save, in that left-to-right
// order along the row under the form.
func TestHitFlowsBuilderButtonsRow(t *testing.T) {
	m := builderModel(t)
	// The form is taller than a short terminal and the window follows the
	// cursor, so the buttons are on screen when the reader is on them.
	m.flows.field = flowFieldSave
	m = m.followField()

	y := rowOfField(t, m, flowFieldAddPhase)

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
