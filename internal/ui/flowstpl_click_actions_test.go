package ui

// flowstpl_click_coverage_test.go is handleFlowClick's whole dispatch table
// — one Target.Field per mouse affordance the builder draws — plus
// readClipboard, which handleFlowClick's "paste_prompt" field is the only
// caller of.

import (
	"strings"
	"testing"
)

func TestHandleFlowClickListActions(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// "create" opens the builder from the list, same as pressing n.
	m2raw, _ := m.handleFlowClick(Target{Field: "create"})

	m2 := asModel(t, m2raw)
	if !m2.flows.creating {
		t.Fatalf("expected the builder open after a create click")
	}

	// "edit" opens a named flow in edit mode.
	m3raw, _ := m.handleFlowClick(Target{Field: "edit", ID: "careful"})

	m3 := asModel(t, m3raw)
	if !m3.flows.creating || m3.flows.flowName != "careful" {
		t.Fatalf("expected edit mode on careful, got creating=%v name=%q", m3.flows.creating, m3.flows.flowName)
	}

	// "delete" asks for confirmation.
	m4raw, _ := m.handleFlowClick(Target{Field: "delete", ID: "careful"})

	m4 := asModel(t, m4raw)
	if !m4.flows.confirmDelete {
		t.Fatalf("expected a delete click to ask for confirmation")
	}
}

func TestHandleFlowClickBuilderButtons(t *testing.T) {
	base, _ := testModel(t, 100, 30)
	base = base.startCreateFlow()

	// "add_phase" grows the phase list and moves the field onto its name.
	m2raw, _ := base.handleFlowClick(Target{Field: "add_phase"})

	m2 := asModel(t, m2raw)
	if len(m2.flows.phases) != 2 {
		t.Fatalf("expected 2 phases after add_phase, got %d", len(m2.flows.phases))
	}

	if m2.flows.field != flowFieldPhaseName {
		t.Errorf("field after add_phase = %d, want flowFieldPhaseName", m2.flows.field)
	}

	// "del_phase" on a single-phase flow refuses rather than emptying it.
	m3raw, _ := base.handleFlowClick(Target{Field: "del_phase"})
	m3 := asModel(t, m3raw)
	wantBand(t, m3, "at least one phase")

	if len(m3.flows.phases) != 1 {
		t.Fatalf("expected the lone phase to survive del_phase")
	}

	// "del_phase" on a two-phase flow actually removes one.
	m4raw, _ := m2.handleFlowClick(Target{Field: "del_phase"})

	m4 := asModel(t, m4raw)
	if len(m4.flows.phases) != 1 {
		t.Fatalf("expected del_phase to drop back to 1 phase, got %d", len(m4.flows.phases))
	}

	// "save" runs the same path as pressing enter on the save button.
	m5 := base
	m5.flows.flowName = "click-to-save"
	m5.opts.Flows = flowsTestDir(t.TempDir())
	m6raw, _ := m5.handleFlowClick(Target{Field: "save"})

	m6 := asModel(t, m6raw)
	if m6.flows.creating {
		t.Errorf("expected a successful save to close the builder")
	}

	// "select_phase" moves the active phase and names it in the band.
	twoPhase := m2
	m7raw, _ := twoPhase.handleFlowClick(Target{Field: "select_phase", Phase: 1})

	m7 := asModel(t, m7raw)
	if m7.flows.activePhase != 1 || m7.flows.field != flowFieldPhaseSelect {
		t.Errorf("select_phase did not move to phase 1: activePhase=%d field=%d", m7.flows.activePhase, m7.flows.field)
	}
}

func TestHandleFlowClickPromptButtons(t *testing.T) {
	base, _ := testModel(t, 100, 30)
	base = base.startCreateFlow()

	// "clear_prompt" empties whatever was there and moves the field onto it.
	withPrompt := base
	withPrompt.flows.phases[0].Prompt = "existing text"
	m2raw, _ := withPrompt.handleFlowClick(Target{Field: "clear_prompt"})

	m2 := asModel(t, m2raw)
	if m2.flows.cur().Prompt != "" {
		t.Errorf("expected clear_prompt to empty the phase's prompt")
	}

	if m2.flows.field != flowFieldPrompt {
		t.Errorf("expected clear_prompt to select the prompt field")
	}

	// "autogen_prompt" writes a generated prompt, worded differently
	// depending on whether a draft was already there.
	m3raw, _ := base.handleFlowClick(Target{Field: "autogen_prompt"})

	m3 := asModel(t, m3raw)
	if m3.flows.cur().Prompt == "" {
		t.Errorf("expected autogen_prompt to fill in a prompt from the role")
	}

	withDraft := base
	withDraft.flows.phases[0].Prompt = "fix the flaky test"
	m4raw, _ := withDraft.handleFlowClick(Target{Field: "autogen_prompt"})

	m4 := asModel(t, m4raw)
	if !strings.Contains(m4.flows.cur().Prompt, "flaky test") {
		t.Errorf("expected autogen_prompt to build on the existing draft, got %q", m4.flows.cur().Prompt)
	}

	// "paste_prompt" reads the system clipboard, whose content this test
	// cannot pin down — only that the call does not panic and lands the
	// field on the prompt when it pastes something.
	m5raw, _ := base.handleFlowClick(Target{Field: "paste_prompt"})

	m5 := asModel(t, m5raw)
	if m5.message == "" {
		t.Errorf("expected paste_prompt to say something either way")
	}
}

// TestHandleFlowClickFieldDefault is the branch handleFlowClick falls to
// when the target names no special field: t.Phase is a flowField* constant
// and the click acts as if that field's own action key had been pressed.
func TestHandleFlowClickFieldDefault(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.startCreateFlow()
	before := m.flows.cur().Engine
	m2raw, _ := m.handleFlowClick(Target{Phase: flowFieldEngine})

	m2 := asModel(t, m2raw)
	if m2.flows.field != flowFieldEngine {
		t.Fatalf("expected the click to select the engine field")
	}

	if m2.flows.cur().Engine == before {
		t.Errorf("expected clicking the engine field to cycle it, stayed %q", before)
	}
}

// TestReadClipboard only asserts that reading the clipboard on this
// platform's own code path returns without panicking — its content is
// whatever the host machine's pasteboard happens to hold, which a
// hermetic test has no business asserting on.
func TestReadClipboard(t *testing.T) {
	_ = readClipboard()
}
