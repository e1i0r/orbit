package flows

// flowstpl_click_coverage_test.go is handleFlowClick's whole dispatch table
// — one Target.Field per mouse affordance the builder draws — plus
// clip.Read, which handleFlowClick's "paste_prompt" field is the only
// caller of.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/clip"
	"github.com/e1i0r/orbit/internal/ui/point"
)

func TestHandleFlowClickListActions(t *testing.T) {
	s, e := designer(t)

	// "create" opens the builder from the list, same as pressing n.
	m2, _ := s.Click(point.Target{Field: "create"}, e)

	if !m2.creating {
		t.Fatalf("expected the builder open after a create click")
	}

	// "edit" opens a named flow in edit mode.
	m3, _ := s.Click(point.Target{Field: "edit", ID: "careful"}, e)

	if !m3.creating || m3.flowName != "careful" {
		t.Fatalf("expected edit mode on careful, got creating=%v name=%q", m3.creating, m3.flowName)
	}

	// "delete" asks for confirmation.
	m4, _ := s.Click(point.Target{Field: "delete", ID: "careful"}, e)

	if !m4.confirmDelete {
		t.Fatalf("expected a delete click to ask for confirmation")
	}
}

func TestHandleFlowClickBuilderButtons(t *testing.T) {
	base, e := designer(t)
	base = base.startCreateFlow(e).OnFields()

	// "add_phase" grows the phase list and moves the field onto its name.
	m2, _ := base.Click(point.Target{Field: "add_phase"}, e)

	if len(m2.phases) != 2 {
		t.Fatalf("expected 2 phases after add_phase, got %d", len(m2.phases))
	}

	if m2.field != flowFieldPhaseName {
		t.Errorf("field after add_phase = %d, want flowFieldPhaseName", m2.field)
	}

	// "del_phase" on a single-phase flow refuses rather than emptying it.
	m3, m3Out := base.Click(point.Target{Field: "del_phase"}, e)

	wantBand(t, m3Out, "at least one phase")

	if len(m3.phases) != 1 {
		t.Fatalf("expected the lone phase to survive del_phase")
	}

	// "del_phase" on a two-phase flow actually removes one.
	m4, _ := m2.Click(point.Target{Field: "del_phase"}, e)

	if len(m4.phases) != 1 {
		t.Fatalf("expected del_phase to drop back to 1 phase, got %d", len(m4.phases))
	}

	// "save" runs the same path as pressing enter on the save button.
	m5 := base
	m5.flowName = "click-to-save"
	e.Flows = flowsTestDir(t.TempDir())
	m6, _ := m5.Click(point.Target{Field: "save"}, e)

	if m6.creating {
		t.Errorf("expected a successful save to close the builder")
	}

	// "select_phase" moves the active phase and names it in the band.
	twoPhase := m2
	m7, _ := twoPhase.Click(point.Target{Field: "select_phase", Phase: 1}, e)

	if m7.activePhase != 1 || m7.field != flowFieldPhaseSelect {
		t.Errorf("select_phase did not move to phase 1: activePhase=%d field=%d", m7.activePhase, m7.field)
	}
}

func TestHandleFlowClickPromptButtons(t *testing.T) {
	base, e := designer(t)
	base = base.startCreateFlow(e).OnFields()

	// "clear_prompt" empties whatever was there and moves the field onto it.
	withPrompt := base
	withPrompt.phases[0].Prompt = "existing text"
	m2, _ := withPrompt.Click(point.Target{Field: "clear_prompt"}, e)

	if m2.cur().Prompt != "" {
		t.Errorf("expected clear_prompt to empty the phase's prompt")
	}

	if m2.field != flowFieldPrompt {
		t.Errorf("expected clear_prompt to select the prompt field")
	}

	// "autogen_prompt" writes a generated prompt, worded differently
	// depending on whether a draft was already there.
	m3, _ := base.Click(point.Target{Field: "autogen_prompt"}, e)

	if m3.cur().Prompt == "" {
		t.Errorf("expected autogen_prompt to fill in a prompt from the role")
	}

	withDraft := base
	withDraft.phases[0].Prompt = "fix the flaky test"
	m4, _ := withDraft.Click(point.Target{Field: "autogen_prompt"}, e)

	if !strings.Contains(m4.cur().Prompt, "flaky test") {
		t.Errorf("expected autogen_prompt to build on the existing draft, got %q", m4.cur().Prompt)
	}

	// "paste_prompt" reads the system clipboard, whose content this test
	// cannot pin down — only that the call does not panic and lands the
	// field on the prompt when it pastes something.
	_, m5Out := base.Click(point.Target{Field: "paste_prompt"}, e)

	if m5Out.Said == "" {
		t.Errorf("expected paste_prompt to say something either way")
	}
}

// TestHandleFlowClickFieldDefault is the branch handleFlowClick falls to
// when the target names no special field: t.Phase is a flowField* constant
// and the click acts as if that field's own action key had been pressed.
func TestHandleFlowClickFieldDefault(t *testing.T) {
	s, e := designer(t)
	s = s.startCreateFlow(e).OnFields()

	// A dial with a list behind it opens the list, rather than stepping one
	// along: an engine has sixty models, and sixty clicks is not a gesture.
	m2, _ := s.Click(point.Target{Phase: flowFieldEngine}, e)

	if m2.field != flowFieldEngine {
		t.Fatalf("expected the click to select the engine field")
	}

	if !m2.picker.open || m2.picker.field != flowFieldEngine {
		t.Errorf("expected clicking the engine field to open its list, got %+v", m2.picker)
	}

	// A switch with two positions still just turns over.
	before := s.cur().Wait
	m3, _ := s.Click(point.Target{Phase: flowFieldWait}, e)

	if m3.cur().Wait == before {
		t.Errorf("expected clicking the control field to turn it over, stayed %v", before)
	}
}

// TestReadClipboard only asserts that reading the clipboard on this
// platform's own code path returns without panicking — its content is
// whatever the host machine's pasteboard happens to hold, which a
// hermetic test has no business asserting on.
func TestReadClipboard(t *testing.T) {
	_ = clip.Read()
}
