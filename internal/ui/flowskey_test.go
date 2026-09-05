package ui

// The way into the flows screen from the board.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestFOpensTheFlowsScreenFromTheBoard.
//
// Flows was the only screen of its rank with no key: repositories answer to
// R, the engine knobs to M, the quota to Q and the supervisor to S, and the
// one screen that says what a run is made of could be reached from inside
// compose, start and the task view — but not from the board a reader is
// looking at. The palette knew the word and nothing on screen did.
func TestFOpensTheFlowsScreenFromTheBoard(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	m = step(t, m, "F")
	if m.screen != screenFlows {
		t.Fatalf("F left the window on %v, want the flows screen", m.screen)
	}
}

// TestTheFlowsKeyIsOnTheBoardsBar. A key nobody is shown is a key nobody
// presses, which is the argument the supervisor's own hint was added under.
//
// The width is wide because the bar drops hints from the end until what is
// left fits, and this one and the supervisor's are at that end: on an eighty
// or a hundred column terminal neither is drawn, and the task verbs before
// them are. Whether that order is the right one is a question about the bar
// and not about this key.
func TestTheFlowsKeyIsOnTheBoardsBar(t *testing.T) {
	m, _ := testModel(t, 180, 30)

	bar, _, _ := m.barLayout(m.frame.Bar.W)
	if drawn := ansi.Strip(bar); !strings.Contains(drawn, "["+m.keys.Flows.Help().Key+"]") {
		t.Errorf("the board's key bar does not name the flows key; it says:\n%s", drawn)
	}
}

// TestTheFlowsKeyAnswersWhatItDoes. Every key the bar draws is one a reader
// can point ? at and be told about.
func TestTheFlowsKeyAnswersWhatItDoes(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	if said := m.meaning(press(m.keys.Flows.Help().Key)); said == "" {
		t.Error("? on the flows key says nothing about what it does")
	}
}
