package ui

import (
	"testing"

	"charm.land/bubbletea/v2"
)

func TestFlowsBuilderFullLifecycle(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	// 1. Open flows screen
	m = m.openFlows()
	if m.screen != screenFlows {
		t.Fatalf("screen = %v, want screenFlows", m.screen)
	}

	sendKey := func(k rune, code rune, text string) {
		var msg tea.Msg

		switch {
		case text != "":
			msg = tea.KeyPressMsg{Code: code, Text: text}
		case code != 0:
			msg = tea.KeyPressMsg{Code: code}
		default:
			msg = tea.KeyPressMsg{Code: k, Text: string(k)}
		}

		updated, _ := m.Update(msg)
		m = asModel(t, updated)
	}

	// 2. Press 'n' to enter flow builder (creating = true)
	sendKey('n', 0, "")

	if !m.flows.creating {
		t.Error("expected creating to be true in flowsState")
	}

	// A new flow opens on the tab that writes one from a sentence; the rest
	// of this walk is about the fields.
	m = m.onFields()

	// 3. Render builder view
	v := m.View()
	if len(v.Content) == 0 {
		t.Error("expected non-empty builder view")
	}

	// 4. Navigate through all builder fields with Tab
	for range flowFieldCount + 2 {
		sendKey(0, tea.KeyTab, "tab")
	}

	// 5. Navigate backwards with Shift-Tab
	for range 3 {
		sendKey(0, tea.KeyTab, "shift+tab")
	}

	// 6. Left / Right on fields to cycle options
	m.flows.field = flowFieldTemplate

	sendKey(0, tea.KeyRight, "")
	sendKey(0, tea.KeyLeft, "")

	m.flows.field = flowFieldEngine

	sendKey(0, tea.KeyRight, "")
	sendKey(0, tea.KeyLeft, "")

	m.flows.field = flowFieldFeedOutput

	sendKey(0, tea.KeyRight, "")

	m.flows.field = flowFieldWait

	sendKey(0, tea.KeyRight, "")

	// 7. Add a phase
	m.flows.field = flowFieldAddPhase

	sendKey(0, tea.KeyEnter, "")

	if len(m.flows.phases) < 2 {
		t.Errorf("expected at least 2 phases after AddPhase, got %d", len(m.flows.phases))
	}

	// 8. Delete a phase
	m.flows.field = flowFieldDelPhase

	sendKey(0, tea.KeyEnter, "")

	// 9. Save flow
	m.flows.field = flowFieldSave
	m.flows.flowName = "my-custom-test-flow"

	sendKey(0, tea.KeyEnter, "")

	// 10. Escape to close
	sendKey(0, tea.KeyEsc, "esc")
}

// TestFlowsBuilderRowsEditingWithPromptAndToggles renders the builder in
// the shapes TestFlowsBuilderFullLifecycle never does: editing an existing
// flow rather than creating one, a phase with a prompt long enough to
// wrap, and both toggles (feed output, wait for human) turned on.
func TestFlowsBuilderRowsEditingWithPromptAndToggles(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m2, _ := m.editFlow("careful")
	m2.screen = screenFlows
	m2.flows.phases[0].Prompt = "a prompt long enough that it should wrap across more than one line of the box"
	m2.flows.phases[0].FeedOutput = true
	m2.flows.phases[0].Wait = true
	m2.flows.field = flowFieldPrompt

	rows := m2.flowsBuilderRows(m2.frame.Body.H, m2.frame.Body.W)
	if len(rows) == 0 {
		t.Fatal("expected a non-empty render")
	}

	v := m2.View()
	if len(v.Content) == 0 {
		t.Error("expected non-empty view content while editing")
	}
}
