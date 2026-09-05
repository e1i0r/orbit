package ui

// The designer's three tabs.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestTheTabsAreThreeViewsOfOneFlow: what the diagram draws is what the
// fields hold, without anything being copied between them.
func TestTheTabsAreThreeViewsOfOneFlow(t *testing.T) {
	m := builderModel(t)
	m.flows.phases[0].Name = "escribe-pruebas"

	m = m.moveFlowTab(1)
	if m.flows.tab != flowTabDiagram {
		t.Fatalf("^→ landed on tab %d", m.flows.tab)
	}

	rows := strings.Join(m.flowsBuilderRows(m.frame.Body.H, m.frame.Body.W), "\n")
	if !strings.Contains(rows, "escribe-pruebas") {
		t.Errorf("the diagram does not show the phase being edited:\n%s", rows)
	}

	// And round the three of them, back to the fields.
	m = m.moveFlowTab(1)
	m = m.moveFlowTab(1)

	if m.flows.tab != flowTabFields {
		t.Errorf("three moves landed on tab %d, want the fields", m.flows.tab)
	}
}

// TestClickingATabOpensIt, and clicking a phase in the diagram takes the
// reader to the fields of that phase.
func TestClickingATabOpensIt(t *testing.T) {
	m := builderModel(t)

	lines, start := m.builderView(m.frame.Body.H, m.frame.Body.W)

	y := 0

	for i, l := range lines {
		if l.strip {
			y = m.frame.Body.Y + i - start
		}
	}

	// The strip's second name is the diagram.
	x := 2 + lipgloss.Width(m.flowTabNames()[0]) + 3

	got := m.hitFlows(x, y)
	if got.Field != "tab" || got.Phase != flowTabDiagram {
		t.Fatalf("hitFlows on the second tab = %+v", got)
	}

	next, _ := m.handleFlowClick(got)

	after := asModel(t, next)
	if after.flows.tab != flowTabDiagram {
		t.Errorf("the click left tab %d open", after.flows.tab)
	}
}
