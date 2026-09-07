package ui

// Looking at a flow from the form, which is the one thing on this screen
// that opens another one.

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/point"
)

// TestTheFormOpensTheDesignerOnTheFlowItIsSetTo, and leaving the designer
// comes back to the form: the reader was in the middle of writing a task.
func TestTheFormOpensTheDesignerOnTheFlowItIsSetTo(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openCompose()

	looking, _ := m.composeKey(tea.KeyPressMsg{Text: "i"})

	inspector := asModel(t, looking)
	if inspector.screen != screenFlows || !inspector.flows.Previewing() {
		t.Fatalf("i left the window on %v, previewing=%v", inspector.screen, inspector.flows.Previewing())
	}

	back, _ := inspector.flowsKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if got := asModel(t, back); got.screen != screenCompose {
		t.Errorf("esc from the designer left the window on %v, want the form", got.screen)
	}
}

// TestPointingAtTheSummaryOpensItToo, which is the same answer the key gives.
func TestPointingAtTheSummaryOpensItToo(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openCompose()

	clicked, _ := m.handleComposeClick(point.Target{Kind: point.ComposeInspectFlow})

	inspector := asModel(t, clicked)
	if inspector.screen != screenFlows || !inspector.flows.Previewing() {
		t.Fatalf("the click left the window on %v, previewing=%v",
			inspector.screen, inspector.flows.Previewing())
	}
}
