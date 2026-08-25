package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/e1i0r/orbit/internal/view"
)

func TestMouseClickRoutingAcrossViews(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Click on header fields
	newM, _ := m.leftClick(Target{Kind: TargetHeaderField, Field: "autopilot"})
	m = asModel(t, newM)

	newM, _ = m.leftClick(Target{Kind: TargetHeaderField, Field: "lang"})
	m = asModel(t, newM)

	newM, _ = m.leftClick(Target{Kind: TargetHeaderField, Field: "orbit"})
	m = asModel(t, newM)

	// 2. Click on task row to move and open
	newM, _ = m.leftClick(Target{Kind: TargetTask, ID: "ACME-2662"})
	m = asModel(t, newM)
	newM, _ = m.leftClick(Target{Kind: TargetTask, ID: "ACME-2662"})
	m = asModel(t, newM)
	if m.screen != screenDetail || m.detail != "ACME-2662" {
		t.Fatalf("expected screenDetail on ACME-2662, got screen=%v, detail=%q", m.screen, m.detail)
	}

	// 3. Click on tabs in detail
	newM, _ = m.leftClick(Target{Kind: TargetPaneTab, Pane: int(tabFlow)})
	m = asModel(t, newM)
	if m.tab != tabFlow {
		t.Errorf("expected tabFlow, got %v", m.tab)
	}

	newM, _ = m.leftClick(Target{Kind: TargetPaneTab, Pane: int(tabCost)})
	m = asModel(t, newM)
	if m.tab != tabCost {
		t.Errorf("expected tabCost, got %v", m.tab)
	}

	// 4. Click band header to toggle fold
	newM, _ = m.leftClick(Target{Kind: TargetBandHeader, Band: view.NeedsYou})
	m = asModel(t, newM)

	// 5. Open settings screen and click option pill
	m.screen = screenSettings
	newM, _ = m.leftClick(Target{Kind: TargetSettingsRow, Pane: 0, Field: "es"})
	m = asModel(t, newM)

	// 6. Right click on task opens menu
	m.screen = screenList
	newM, _ = m.rightClick(Target{Kind: TargetTask, ID: "ACME-2662"})
	m = asModel(t, newM)
}

func TestMouseWheelAndKeyHintClick(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// Key hint click
	newM, _ := m.leftClick(Target{Kind: TargetBarHint, Key: "?"})
	m = asModel(t, newM)

	// Wheel scrolling
	wheelDown := tea.MouseWheelMsg{
		X: 50, Y: 10,
		Button: tea.MouseWheelDown,
	}
	mWheel := m.wheel(wheelDown.Mouse())
	if mWheel.cursor < 0 {
		t.Error("cursor should be >= 0 after wheel")
	}

	wheelUp := tea.MouseWheelMsg{
		X: 50, Y: 10,
		Button: tea.MouseWheelUp,
	}
	_ = m.wheel(wheelUp.Mouse())
}
