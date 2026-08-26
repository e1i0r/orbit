package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestComposeInspectFlow(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openCompose()
	m.compose.field = composeFlow

	// 1. Pressing 'i' opens flow inspector preview
	res, _ := m.composeKey(tea.KeyPressMsg{Text: "i"})
	mInspector := asModel(t, res)
	if mInspector.screen != screenFlows || !mInspector.flows.readOnly {
		t.Fatalf("expected screenFlows with readOnly, got screen=%v readOnly=%v",
			mInspector.screen, mInspector.flows.readOnly)
	}

	// 2. Pressing Esc returns to screenCompose
	resBack, _ := mInspector.flowsKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	mBack := asModel(t, resBack)
	if mBack.screen != screenCompose {
		t.Errorf("expected screenCompose after Esc from inspector, got %v", mBack.screen)
	}
}

func TestComposeInspectFlowMouseClick(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openCompose()

	// Mouse click on inspect button
	target := Target{Kind: TargetComposeInspectFlow}
	res, _ := m.handleComposeClick(target)
	mInspector := asModel(t, res)
	if mInspector.screen != screenFlows || !mInspector.flows.readOnly {
		t.Fatalf("expected screenFlows with readOnly after click, got screen=%v", mInspector.screen)
	}
}
