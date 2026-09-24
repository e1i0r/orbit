package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestNewFlowUnderTheRowOpensTheDesigner. "[+] new flow" is drawn under the
// flows and answered as the row, so a click on it changed the flow. It now
// does what + does, and the rest of that line changes nothing.
func TestNewFlowUnderTheRowOpensTheDesigner(t *testing.T) {
	m := startedOn(t, "ACME-2698")

	p := m.startLayout(m.frame.Body.W)
	line := p.flow + p.nFlow - 1
	y := m.frame.Body.Y + line

	drawn := ansi.Strip(m.flowBlock(m.frame.Body.W)[p.nFlow-1])

	at := strings.Index(drawn, "new flow")
	if at < 0 {
		t.Fatalf("the last line of the flow block is %q, want [+] new flow", drawn)
	}

	before := m.start.at

	next := clicked(t, m, m.hit(ansi.StringWidth(drawn[:at]), y))
	if next.screen != screenFlows {
		t.Errorf("a click on new flow left the screen on %v, want the flows designer", next.screen)
	}

	rest := clicked(t, m, m.hit(len(drawn)+2, y))
	if rest.start.at != before || rest.screen != screenStart {
		t.Errorf("a click past new flow moved the choice to %d on screen %v", rest.start.at, rest.screen)
	}
}
