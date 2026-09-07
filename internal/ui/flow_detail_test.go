package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestFlowDetailRowsRendering(t *testing.T) {
	m, _ := testModel(t, 100, 50)
	m = m.openFlowPreview("tdd-fuzz-pr")

	if !m.flows.Previewing() {
		t.Fatalf("the preview did not open")
	}

	rows := m.flowsRows(m.frame.Body.H, m.frame.Body.W)
	joined := strings.Join(rows, "\n")

	// Verify Header & Origin Badge
	if !strings.Contains(joined, "tdd-fuzz-pr") {
		t.Errorf("expected flow name in detail view")
	}

	// Verify Purpose / Description is rendered
	if !strings.Contains(joined, "test-driven") {
		t.Errorf("expected purpose description in detail view")
	}

	// Verify Phases Breakdown
	if !strings.Contains(joined, "1-plan") || !strings.Contains(joined, "2-implement-fuzz") ||
		!strings.Contains(joined, "3-review-pr") {
		t.Errorf("expected phase names in detail view")
	}

	// Verify Action Buttons
	if !strings.Contains(joined, "Select & Return") && !strings.Contains(joined, "Seleccionar y Volver") {
		t.Errorf("expected select button in footer")
	}
}

func TestFlowDetailKeyNavigation(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openCompose()

	// 1. Inspect from compose
	m = m.openFlowPreview("careful")
	if !m.flows.Previewing() {
		t.Fatal("inspecting a flow from the compose form did not open the preview")
	}

	// 2. Select & Return with Enter
	res, _ := m.flowsKey(tea.KeyPressMsg{Code: tea.KeyEnter})

	mSelected := asModel(t, res)
	if mSelected.screen != screenCompose {
		t.Errorf("expected return to compose after Enter, got %v", mSelected.screen)
	}

	if got := mSelected.compose.Flow(); got != "careful" {
		t.Errorf("the form's flow is %q, want the one that was chosen", got)
	}

	// 3. Edit with 'e'
	m = m.openFlowPreview("quick")
	resEdit, _ := m.flowsKey(tea.KeyPressMsg{Text: "e"})

	mEditing := asModel(t, resEdit)
	if !mEditing.flows.Creating() || mEditing.flows.Editing() != "quick" {
		t.Errorf("e opened %q rather than the form on quick", mEditing.flows.Editing())
	}

	// 4. Back with Esc from flow list
	mList := m.openFlows()
	mList = mList.openFlowPreview("task")
	resBack, _ := mList.flowsKey(tea.KeyPressMsg{Code: tea.KeyEscape})

	mBack := asModel(t, resBack)
	if mBack.flows.Previewing() {
		t.Errorf("expected showingDetail to be false after Esc from flow list")
	}
}
