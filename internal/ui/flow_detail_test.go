package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

func TestFlowDetailRowsRendering(t *testing.T) {
	m, _ := testModel(t, 100, 50)
	m = m.openFlowPreview("tdd-fuzz-pr")

	if !m.flows.showingDetail {
		t.Fatalf("expected showingDetail to be true")
	}

	rows := m.flowDetailRows(m.frame.Body.H, m.frame.Body.W)
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
	if m.flows.fromScreen != screenCompose || !m.flows.showingDetail {
		t.Fatalf("expected showingDetail from screenCompose")
	}

	// 2. Select & Return with Enter
	res, _ := m.flowsKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	mSelected := asModel(t, res)
	if mSelected.screen != screenCompose {
		t.Errorf("expected return to compose after Enter, got %v", mSelected.screen)
	}
	if mSelected.compose.flows[mSelected.compose.flowIdx] != "careful" {
		t.Errorf("expected compose flow to be set to careful, got %s",
			mSelected.compose.flows[mSelected.compose.flowIdx])
	}

	// 3. Edit with 'e'
	m = m.openFlowPreview("quick")
	resEdit, _ := m.flowsKey(tea.KeyPressMsg{Text: "e"})
	mEditing := asModel(t, resEdit)
	if !mEditing.flows.creating || !mEditing.flows.isEditing {
		t.Errorf("expected creating & isEditing after pressing 'e'")
	}
	if mEditing.flows.flowName != "quick" {
		t.Errorf("expected flowName 'quick', got %s", mEditing.flows.flowName)
	}

	// 4. Back with Esc from flow list
	mList := m.openFlows()
	mList = mList.openFlowPreview("task")
	resBack, _ := mList.flowsKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	mBack := asModel(t, resBack)
	if mBack.flows.showingDetail {
		t.Errorf("expected showingDetail to be false after Esc from flow list")
	}
}

func FuzzRenderFlowDiagram(f *testing.F) {
	f.Add("phase-1", "claude", "opus", true, false, 80)
	f.Add("p1", "codex", "sonnet", false, true, 40)
	f.Add("long-phase-name-test", "opencode", "default", true, true, 120)

	f.Fuzz(func(t *testing.T, name, engine, model string, feed, wait bool, width int) {
		if width < 10 || width > 300 {
			return
		}
		phases := []flow.Phase{
			{Name: name, Engine: engine, Model: model, FeedOutput: feed, Wait: wait},
			{Name: "2-" + name, Engine: engine, Model: model, FeedOutput: !feed, Wait: !wait},
		}
		lines := renderFlowDiagram(phases, width)
		if len(lines) == 0 {
			t.Errorf("expected diagram lines, got 0")
		}
	})
}
