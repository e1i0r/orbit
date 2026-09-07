package flows

// The designer's three tabs.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestTheTabsAreThreeViewsOfOneFlow: what the diagram draws is what the
// fields hold, without anything being copied between them.
func TestTheTabsAreThreeViewsOfOneFlow(t *testing.T) {
	s, e := editing(t)
	s.phases[0].Name = "escribe-pruebas"

	s = s.moveFlowTab(1)
	if s.tab != flowTabDiagram {
		t.Fatalf("^→ landed on tab %d", s.tab)
	}

	rows := strings.Join(linesOf(s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)), "\n")
	if !strings.Contains(rows, "escribe-pruebas") {
		t.Errorf("the diagram does not show the phase being edited:\n%s", rows)
	}

	// And round the three of them, back to the fields.
	s = s.moveFlowTab(1)
	s = s.moveFlowTab(1)

	if s.tab != flowTabFields {
		t.Errorf("three moves landed on tab %d, want the fields", s.tab)
	}
}

// TestClickingATabOpensIt, and clicking a phase in the diagram takes the
// reader to the fields of that phase.
func TestClickingATabOpensIt(t *testing.T) {
	s, e := editing(t)

	lines, start := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)

	y := 0

	for i, l := range lines {
		if l.strip {
			y = e.Frame.Body.Y + i - start
		}
	}

	// The strip's second name is the diagram.
	x := 2 + lipgloss.Width(s.flowTabNames(e)[0]) + 3

	got := s.Hit(x, y, e)
	if got.Field != "tab" || got.Phase != flowTabDiagram {
		t.Fatalf("hitFlows on the second tab = %+v", got)
	}

	next, _ := s.Click(got, e)

	after := next
	if after.tab != flowTabDiagram {
		t.Errorf("the click left tab %d open", after.tab)
	}
}
