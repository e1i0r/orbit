package flows

// hitFlows's builder-view half: the pipeline overview, the field rows and
// the boxes, once the reader is creating or editing a flow rather than
// looking at the list.
//
// The rows are found rather than written down. This test held a table of
// line numbers walked by hand, which is the same table hitFlows itself held
// — so the two agreed about a layout neither of them was reading. Both now
// read builderView, and a row that moves moves in the test with it.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/ui/point"
)

// rowOfField is the screen row the given field was drawn on, and the last of
// them when the field is a box several rows tall.
func rowOfField(t *testing.T, s State, field int, e Env) int {
	t.Helper()

	lines, start := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)

	for i, l := range lines {
		if l.field == field {
			return e.Frame.Body.Y + i - start
		}
	}

	t.Fatalf("no row was drawn for field %d", field)

	return 0
}

// rowsOf is every screen row the given predicate drew.
func rowsOf(s State, keep func(builderLine) bool, e Env) []int {
	lines, start := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)

	var out []int

	for i, l := range lines {
		if keep(l) {
			out = append(out, e.Frame.Body.Y+i-start)
		}
	}

	return out
}

func TestHitFlowsBuilderOverview(t *testing.T) {
	s, e := editing(t)

	rows := rowsOf(s, func(l builderLine) bool { return l.phase == 0 }, e)
	if len(rows) != 1 {
		t.Fatalf("the lone phase drew %d overview rows", len(rows))
	}

	got := s.Hit(10, rows[0], e)
	if got.Kind != point.FlowItem || got.Field != "select_phase" || got.Phase != 0 {
		t.Errorf("hitFlows on the overview row = %+v, want select_phase 0", got)
	}

	// The screen's own title selects nothing.
	if got := s.Hit(10, e.Frame.Body.Y, e); got.Kind != point.None {
		t.Errorf("hitFlows on the title = %+v, want the zero point.Target", got)
	}
}

func TestHitFlowsBuilderOverviewWithPrompt(t *testing.T) {
	s, e := editing(t)
	s.phases[0].Prompt = "already written"

	// A phase with a prompt draws an extra line, so its overview row is
	// two lines tall rather than one, and both select it.
	rows := rowsOf(s, func(l builderLine) bool { return l.phase == 0 }, e)
	if len(rows) != 2 {
		t.Fatalf("a phase with a prompt drew %d overview rows, want 2", len(rows))
	}

	for _, y := range rows {
		got := s.Hit(10, y, e)
		if got.Kind != point.FlowItem || got.Field != "select_phase" || got.Phase != 0 {
			t.Errorf("hitFlows at overview row %d = %+v, want select_phase 0", y, got)
		}
	}
}

func TestHitFlowsBuilderSecondPhase(t *testing.T) {
	s, e := editing(t)
	s.phases = append(s.phases, s.phases[0])

	for phase := range 2 {
		rows := rowsOf(s, func(l builderLine) bool { return l.phase == phase }, e)
		if len(rows) == 0 {
			t.Fatalf("phase %d drew no overview row", phase)
		}

		got := s.Hit(10, rows[0], e)
		if got.Kind != point.FlowItem || got.Field != "select_phase" || got.Phase != phase {
			t.Errorf("overview row of phase %d = %+v", phase, got)
		}
	}
}

// TestHitFlowsBuilderFields is every field of the form, clicked where it was
// drawn.
func TestHitFlowsBuilderFields(t *testing.T) {
	s, e := editing(t)

	for _, field := range s.fieldsShown() {
		if field == flowFieldAddPhase || field == flowFieldDelPhase || field == flowFieldSave {
			continue // three fields on one row, told apart by x below
		}

		got := s.Hit(10, rowOfField(t, s, field, e), e)
		if got.Kind != point.FlowItem || got.Field != "" || got.Phase != field {
			t.Errorf("hitFlows on field %d = %+v", field, got)
		}
	}
}

// TestHitFlowsBuilderLoopFields is the two fields that exist only while the
// phase repeats, which is the whole reason the layout is read rather than
// written down.
func TestHitFlowsBuilderLoopFields(t *testing.T) {
	s, e := editing(t)
	s = s.toggleLoop()

	for _, field := range []int{flowFieldLoopTurns, flowFieldLoopUntil} {
		got := s.Hit(10, rowOfField(t, s, field, e), e)
		if got.Kind != point.FlowItem || got.Phase != field {
			t.Errorf("hitFlows on loop field %d = %+v", field, got)
		}
	}
}

// TestHitFlowsBuilderPromptRowButtons is the prompt row's own x-ranges: the
// paste, autogenerate and clear pills to its right, and the field itself to
// their left.
func TestHitFlowsBuilderPromptRowButtons(t *testing.T) {
	s, e := editing(t)

	lines, start := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)

	y := 0

	for i, l := range lines {
		if l.head && l.field == flowFieldPrompt {
			y = e.Frame.Body.Y + i - start
		}
	}

	got := s.Hit(10, y, e)
	if got.Kind != point.FlowItem || got.Field != "" || got.Phase != flowFieldPrompt {
		t.Errorf("hitFlows left of the prompt row's pills = %+v, want the prompt field", got)
	}

	if got := s.Hit(29, y, e); got.Field != "paste_prompt" {
		t.Errorf("hitFlows on the paste pill = %+v, want paste_prompt", got)
	}

	if got := s.Hit(43, y, e); got.Field != "autogen_prompt" {
		t.Errorf("hitFlows on the autogenerate pill = %+v, want autogen_prompt", got)
	}

	if got := s.Hit(60, y, e); got.Field != "clear_prompt" {
		t.Errorf("hitFlows on the clear pill = %+v, want clear_prompt", got)
	}
}

// TestHitFlowsBuilderPromptBox is the multi-line box under the prompt row,
// which is still the prompt field wherever inside it the reader clicks.
func TestHitFlowsBuilderPromptBox(t *testing.T) {
	s, e := editing(t)
	s.field = flowFieldPrompt
	s = s.followField(e)

	rows := rowsOf(s, func(l builderLine) bool { return l.field == flowFieldPrompt && !l.head }, e)
	if len(rows) < 3 {
		t.Fatalf("the prompt box drew %d rows, want its frame and something inside it", len(rows))
	}

	for _, y := range rows {
		got := s.Hit(10, y, e)
		if got.Kind != point.FlowItem || got.Field != "" || got.Phase != flowFieldPrompt {
			t.Errorf("hitFlows in the prompt box at %d = %+v, want the prompt field", y, got)
		}
	}
}

// TestHitFlowsBuilderButtonsRow is add/delete/save, in that left-to-right
// order along the row under the form.
func TestHitFlowsBuilderButtonsRow(t *testing.T) {
	s, e := editing(t)
	// The form is taller than a short terminal and the window follows the
	// cursor, so the buttons are on screen when the reader is on them.
	s.field = flowFieldSave
	s = s.followField(e)

	y := rowOfField(t, s, flowFieldAddPhase, e)

	if got := s.Hit(10, y, e); got.Field != "add_phase" {
		t.Errorf("hitFlows on add_phase = %+v", got)
	}

	if got := s.Hit(30, y, e); got.Field != "del_phase" {
		t.Errorf("hitFlows on del_phase = %+v", got)
	}

	if got := s.Hit(50, y, e); got.Field != "save" {
		t.Errorf("hitFlows on save = %+v", got)
	}
}
