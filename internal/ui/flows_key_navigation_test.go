package ui

// flows_more_coverage_test.go is flows.go's own remaining gaps: ensurePhase
// as a method on flowsState directly, the list and form keymaps' less
// common branches, and the two field-delta/action switches driven field by
// field rather than through one lifecycle walk.

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

// twoPhases is a phase list long enough to clamp an activePhase against,
// with nothing about its content that any test below reads.
func twoPhases() []flow.Phase {
	return []flow.Phase{
		{Name: "a", Engine: "claude"},
		{Name: "b", Engine: "claude"},
	}
}

func TestEnsurePhase(t *testing.T) {
	// 1. An empty state gets a default phase.
	st := flowsState{}
	st.ensurePhase()

	if len(st.phases) != 1 || st.activePhase != 0 {
		t.Fatalf("ensurePhase on empty state = %+v", st)
	}

	// 2. A negative active phase clamps to zero.
	st2 := flowsState{phases: twoPhases(), activePhase: -3}
	st2.ensurePhase()

	if st2.activePhase != 0 {
		t.Errorf("activePhase after a negative index = %d, want 0", st2.activePhase)
	}

	// 3. An active phase past the end clamps to the last one.
	st3 := flowsState{phases: twoPhases(), activePhase: 99}
	st3.ensurePhase()

	if st3.activePhase != 1 {
		t.Errorf("activePhase after an out-of-range index = %d, want 1", st3.activePhase)
	}
}

func TestAbandonFlowsFromStart(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.screen = screenStart
	m = m.openFlows()

	m2 := m.abandonFlows()
	if m2.screen != screenStart {
		t.Errorf("abandonFlows from the start dialog = %v, want screenStart", m2.screen)
	}
}

func TestFlowsListKeyConfirmDelete(t *testing.T) {
	dir := t.TempDir()
	writeFlowFile(t, dir, "zzz-mine", `{"name":"zzz-mine","phases":[{"name":"implement","engine":"claude"}]}`)

	m, _ := testModel(t, 100, 30)
	m.opts.Flows = flowsTestDir(dir)
	m = m.openFlows()
	// The last row: the flows are sorted by name and zzz-mine is after every
	// built-in, so it is the one past the end of them.
	m.flows.sel, m.flows.confirmDelete = len(flow.BuiltinNames()), true

	// A key other than yes cancels the deletion.
	m2raw, _ := m.flowsListKey(press("x"))

	m2 := asModel(t, m2raw)
	if m2.flows.confirmDelete {
		t.Fatalf("expected the confirmation to be cancelled")
	}

	wantBand(t, m2, "cancelled")

	// "y" confirms it.
	m.flows.confirmDelete = true
	m3raw, _ := m.flowsListKey(press("y"))
	m3 := asModel(t, m3raw)
	wantBand(t, m3, "zzz-mine")
}

func TestFlowsListKeyNavigationBoundaries(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openFlows()

	// sel starts at -1 and Up must not go lower.
	m2raw, _ := m.flowsListKey(press("up"))
	if asModel(t, m2raw).flows.sel != -1 {
		t.Errorf("Up at sel=-1 moved: %d", asModel(t, m2raw).flows.sel)
	}

	// Down walks off the end of the descriptor list and no further. Counted
	// against the builtins themselves: a number written here goes stale the
	// day a flow ships, and the test that fails then is not about what
	// changed.
	builtinCount := len(flow.BuiltinNames())

	m2 := asModel(t, m2raw)
	for range builtinCount + 2 {
		next, _ := m2.flowsListKey(press("down"))
		m2 = asModel(t, next)
	}

	if m2.flows.sel != builtinCount-1 {
		t.Errorf("sel after walking past the end = %d, want %d", m2.flows.sel, builtinCount-1)
	}

	// Back leaves the screen.
	m3raw, _ := m2.flowsListKey(press("esc"))
	if asModel(t, m3raw).screen == screenFlows {
		t.Errorf("expected Back to leave the flows screen")
	}

	// Start ('n'), Open on nothing selected, and the plain letters all open
	// the builder or an editor.
	m4raw, _ := m.flowsListKey(press("n"))
	if !asModel(t, m4raw).flows.creating {
		t.Errorf("expected 'n' to open the builder")
	}

	m5 := m
	m5.flows.sel = -1

	m5raw, _ := m5.flowsListKey(press("enter"))
	if !asModel(t, m5raw).flows.creating {
		t.Errorf("expected Open with nothing selected to open the builder")
	}

	m6 := m
	m6.flows.sel = 0

	m6raw, _ := m6.flowsListKey(press("e"))
	if !asModel(t, m6raw).flows.creating {
		t.Errorf("expected 'e' to edit the selected flow")
	}

	m7 := m
	m7.flows.sel = 0
	m7raw, _ := m7.flowsListKey(press("d"))
	wantBand(t, asModel(t, m7raw), "cannot be deleted")
}

func TestFlowsFormKeyConfirmDiscard(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.startCreateFlow()
	m.flows.flowName = "typed-something"

	// Back with something typed asks first, rather than discarding at once.
	m2raw, _ := m.flowsFormKey(press("esc"))

	m2 := asModel(t, m2raw)
	if !m2.flows.confirmDiscard {
		t.Fatalf("expected Back to ask for confirmation once something is typed")
	}

	// A non-yes key resumes editing.
	m3raw, _ := m2.flowsFormKey(press("x"))

	m3 := asModel(t, m3raw)
	if m3.flows.confirmDiscard || !m3.flows.creating {
		t.Fatalf("expected editing to resume on any other key")
	}

	wantBand(t, m3, "resumed")

	// "y" discards and closes the builder.
	m4raw, _ := m2.flowsFormKey(press("y"))

	m4 := asModel(t, m4raw)
	if m4.flows.creating {
		t.Fatalf("expected 'y' to discard and close the builder")
	}

	// Back with nothing typed closes without asking.
	m5, _ := testModel(t, 100, 30)
	m5 = m5.startCreateFlow()
	m6raw, _ := m5.flowsFormKey(press("esc"))

	m6 := asModel(t, m6raw)
	if m6.flows.creating || m6.flows.confirmDiscard {
		t.Fatalf("expected a blank form to close without confirmation")
	}
}

func TestFlowsFormKeyTextFields(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.startCreateFlow()

	m.flows.field = flowFieldName
	m2raw, _ := m.flowsFormKey(press("x"))

	m2 := asModel(t, m2raw)
	if m2.flows.flowName != "x" {
		t.Fatalf("flowName after typing = %q", m2.flows.flowName)
	}

	m3raw, _ := m2.flowsFormKey(tea.KeyPressMsg{Code: tea.KeyBackspace})

	m3 := asModel(t, m3raw)
	if m3.flows.flowName != "" {
		t.Errorf("flowName after backspace = %q, want empty", m3.flows.flowName)
	}

	m3.flows.field = flowFieldPhaseName
	m4raw, _ := m3.flowsFormKey(press("y"))

	m4 := asModel(t, m4raw)
	if m4.flows.cur().Name != "1-implementy" {
		t.Errorf("phase name after typing = %q", m4.flows.cur().Name)
	}

	m5raw, _ := m4.flowsFormKey(tea.KeyPressMsg{Code: tea.KeyBackspace})

	m5 := asModel(t, m5raw)
	if m5.flows.cur().Name != "1-implement" {
		t.Errorf("phase name after backspace = %q", m5.flows.cur().Name)
	}

	m5.flows.field = flowFieldPrompt
	m6raw, _ := m5.flowsFormKey(press("z"))

	m6 := asModel(t, m6raw)
	if m6.flows.cur().Prompt != "z" {
		t.Errorf("prompt after typing = %q", m6.flows.cur().Prompt)
	}

	m7raw, _ := m6.flowsFormKey(tea.KeyPressMsg{Code: tea.KeyBackspace})

	m7 := asModel(t, m7raw)
	if m7.flows.cur().Prompt != "" {
		t.Errorf("prompt after backspace = %q, want empty", m7.flows.cur().Prompt)
	}
}

func TestFlowsFormKeyTabAndArrows(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.startCreateFlow()

	m2raw, _ := m.flowsFormKey(press("tab"))
	if asModel(t, m2raw).flows.field != 1 {
		t.Errorf("field after Tab = %d, want 1", asModel(t, m2raw).flows.field)
	}

	m3raw, _ := asModel(t, m2raw).flowsFormKey(press("shift+tab"))
	if asModel(t, m3raw).flows.field != 0 {
		t.Errorf("field after Shift-Tab = %d, want 0", asModel(t, m3raw).flows.field)
	}

	m4 := m
	m4.flows.field = flowFieldEngine
	before := m4.flows.cur().Engine
	m5raw, _ := m4.flowsFormKey(press("right"))

	m5 := asModel(t, m5raw)
	if m5.flows.cur().Engine == before {
		t.Errorf("expected Right to cycle the engine field")
	}

	m6raw, _ := m5.flowsFormKey(press("left"))

	m6 := asModel(t, m6raw)
	if m6.flows.cur().Engine != before {
		t.Errorf("expected Left to cycle back")
	}

	m7 := m
	m7.flows.field = flowFieldFeedOutput
	m8raw, _ := m7.flowsFormKey(press(" "))

	m8 := asModel(t, m8raw)
	if !m8.flows.cur().FeedOutput {
		t.Errorf("expected space on FeedOutput to toggle it on")
	}
}
