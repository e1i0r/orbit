package flows

// flows_more_coverage_test.go is go's own remaining gaps: ensurePhase
// as a method on State directly, the list and form keymaps' less
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
	st := State{}
	st.ensurePhase()

	if len(st.phases) != 1 || st.activePhase != 0 {
		t.Fatalf("ensurePhase on empty state = %+v", st)
	}

	// 2. A negative active phase clamps to zero.
	st2 := State{phases: twoPhases(), activePhase: -3}
	st2.ensurePhase()

	if st2.activePhase != 0 {
		t.Errorf("activePhase after a negative index = %d, want 0", st2.activePhase)
	}

	// 3. An active phase past the end clamps to the last one.
	st3 := State{phases: twoPhases(), activePhase: 99}
	st3.ensurePhase()

	if st3.activePhase != 1 {
		t.Errorf("activePhase after an out-of-range index = %d, want 1", st3.activePhase)
	}
}

func TestAbandonFlowsFromStart(t *testing.T) {
	e := world(t)
	s := Open(FromStart, e)

	_, out := s.leave()
	if !out.Leave || out.Back != FromStart {
		t.Errorf("leaving the designer answered %+v, want it to go back to the start dialog", out)
	}
}

func TestFlowsListKeyConfirmDelete(t *testing.T) {
	dir := t.TempDir()
	writeFlowFile(t, dir, "zzz-mine", `{"name":"zzz-mine","phases":[{"name":"implement","engine":"claude"}]}`)

	_, e := designer(t)
	e.Flows = flowsTestDir(dir)
	s := Open(FromBoard, e)
	// The last row: the flows are sorted by name and zzz-mine is after every
	// built-in, so it is the one past the end of them.
	s.sel, s.confirmDelete = len(flow.BuiltinNames()), true

	// A key other than yes cancels the deletion.
	m2, m2Out := s.flowsListKey(press("x"), e)

	if m2.confirmDelete {
		t.Fatalf("expected the confirmation to be cancelled")
	}

	wantBand(t, m2Out, "cancelled")

	// "y" confirms it.
	s.confirmDelete = true
	_, m3Out := s.flowsListKey(press("y"), e)

	wantBand(t, m3Out, "zzz-mine")
}

func TestFlowsListKeyNavigationBoundaries(t *testing.T) {
	s, e := designer(t)

	// sel starts at -1 and Up must not go lower.
	m2, _ := s.flowsListKey(press("up"), e)
	if m2.sel != -1 {
		t.Errorf("Up at sel=-1 moved: %d", m2.sel)
	}

	// Down walks off the end of the descriptor list and no further. Counted
	// against the builtins themselves: a number written here goes stale the
	// day a flow ships, and the test that fails then is not about what
	// changed.
	builtinCount := len(flow.BuiltinNames())

	for range builtinCount + 2 {
		next, _ := m2.flowsListKey(press("down"), e)
		m2 = next
	}

	if m2.sel != builtinCount-1 {
		t.Errorf("sel after walking past the end = %d, want %d", m2.sel, builtinCount-1)
	}

	// Back leaves the screen.
	_, m3Out := m2.flowsListKey(press("esc"), e)
	if !m3Out.Leave {
		t.Error("esc did not ask the window to close the designer")
	}

	// Start ('n'), Open on nothing selected, and the plain letters all open
	// the builder or an editor.
	m4, _ := s.flowsListKey(press("n"), e)
	if !m4.creating {
		t.Errorf("expected 'n' to open the builder")
	}

	m5 := s
	m5.sel = -1

	m5, _ = m5.flowsListKey(press("enter"), e)
	if !m5.creating {
		t.Errorf("expected Open with nothing selected to open the builder")
	}

	m6 := s
	m6.sel = 0

	m6, _ = m6.flowsListKey(press("e"), e)
	if !m6.creating {
		t.Errorf("expected 'e' to edit the selected flow")
	}

	m7 := s
	m7.sel = 0
	m7, m7Out := m7.flowsListKey(press("d"), e)
	wantBand(t, m7Out, "cannot be deleted")
}

func TestFlowsFormKeyConfirmDiscard(t *testing.T) {
	s, e := designer(t)
	s = s.startCreateFlow(e).OnFields()
	s.flowName = "typed-something"

	// Back with something typed asks first, rather than discarding at once.
	m2, _ := s.flowsFormKey(press("esc"), e)

	if !m2.confirmDiscard {
		t.Fatalf("expected Back to ask for confirmation once something is typed")
	}

	// A non-yes key resumes editing.
	m3, m3Out := m2.flowsFormKey(press("x"), e)

	if m3.confirmDiscard || !m3.creating {
		t.Fatalf("expected editing to resume on any other key")
	}

	wantBand(t, m3Out, "resumed")

	// "y" discards and closes the builder.
	m4, _ := m2.flowsFormKey(press("y"), e)

	if m4.creating {
		t.Fatalf("expected 'y' to discard and close the builder")
	}

	// Back with nothing typed closes without asking.
	m5, e := designer(t)
	m5 = m5.startCreateFlow(e).OnFields()
	m6, _ := m5.flowsFormKey(press("esc"), e)

	if m6.creating || m6.confirmDiscard {
		t.Fatalf("expected a blank form to close without confirmation")
	}
}

func TestFlowsFormKeyTextFields(t *testing.T) {
	s, e := designer(t)
	s = s.startCreateFlow(e).OnFields()

	s.field = flowFieldName
	m2, _ := s.flowsFormKey(press("x"), e)

	if m2.flowName != "x" {
		t.Fatalf("flowName after typing = %q", m2.flowName)
	}

	m3, _ := m2.flowsFormKey(tea.KeyPressMsg{Code: tea.KeyBackspace}, e)

	if m3.flowName != "" {
		t.Errorf("flowName after backspace = %q, want empty", m3.flowName)
	}

	m3.field = flowFieldPhaseName
	m4, _ := m3.flowsFormKey(press("y"), e)

	if m4.cur().Name != "1-implementy" {
		t.Errorf("phase name after typing = %q", m4.cur().Name)
	}

	m5, _ := m4.flowsFormKey(tea.KeyPressMsg{Code: tea.KeyBackspace}, e)

	if m5.cur().Name != "1-implement" {
		t.Errorf("phase name after backspace = %q", m5.cur().Name)
	}

	m5.field = flowFieldPrompt
	m6, _ := m5.flowsFormKey(press("z"), e)

	if m6.cur().Prompt != "z" {
		t.Errorf("prompt after typing = %q", m6.cur().Prompt)
	}

	m7, _ := m6.flowsFormKey(tea.KeyPressMsg{Code: tea.KeyBackspace}, e)

	if m7.cur().Prompt != "" {
		t.Errorf("prompt after backspace = %q, want empty", m7.cur().Prompt)
	}
}

func TestFlowsFormKeyTabAndArrows(t *testing.T) {
	s, e := designer(t)
	s = s.startCreateFlow(e).OnFields()

	m2, _ := s.flowsFormKey(press("tab"), e)
	if m2.field != 1 {
		t.Errorf("field after Tab = %d, want 1", m2.field)
	}

	m3, _ := m2.flowsFormKey(press("shift+tab"), e)
	if m3.field != 0 {
		t.Errorf("field after Shift-Tab = %d, want 0", m3.field)
	}

	// Two engines, because a dial with one option cycles onto itself.
	e.Engines = func() []string { return []string{"zeta", "omega"} }

	m4 := s
	m4.field = flowFieldEngine
	before := m4.cur().Engine
	m5, _ := m4.flowsFormKey(press("right"), e)

	if m5.cur().Engine == before {
		t.Errorf("expected Right to cycle the engine field")
	}

	m6, _ := m5.flowsFormKey(press("left"), e)

	if m6.cur().Engine != before {
		t.Errorf("expected Left to cycle back")
	}

	m7 := s
	m7.field = flowFieldFeedOutput
	m8, _ := m7.flowsFormKey(press(" "), e)

	if !m8.cur().FeedOutput {
		t.Errorf("expected space on FeedOutput to toggle it on")
	}
}
