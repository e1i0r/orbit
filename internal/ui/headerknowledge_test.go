package ui

// The chip that says how much Orbit knows.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/charmbracelet/x/ansi"
	"github.com/e1i0r/orbit/internal/knowledge"
)

func withFacts(t *testing.T, n int) Model {
	t.Helper()

	facts := make([]knowledge.Fact, 0, n)
	for range n {
		facts = append(facts, knowledge.Fact{
			Scope: knowledge.Scope{Kind: knowledge.General}, Source: knowledge.Human, Phrase: "something",
		})
	}

	m, _ := testModel(t, 140, 30)
	m.opts.KnowsAll = func() []knowledge.Fact { return facts }

	// The chip counts what was loaded, never the port: the header is drawn
	// on every frame and the port walks every repository.
	return m.syncKnowledge()
}

// TestTheHeaderSaysHowMuchOrbitKnows, beside how many repositories there are.
//
// Both are standing facts about the workspace and neither replaces the other:
// the repository count says how much there is, and this says how much has
// been learned about it. It is also the door a pointer has onto the screen.
func TestTheHeaderSaysHowMuchOrbitKnows(t *testing.T) {
	drawn := ansi.Strip(withFacts(t, 3).headerLine(140))

	if !strings.Contains(drawn, "3") {
		t.Errorf("the header does not say how many facts there are:\n%s", drawn)
	}

	if !strings.Contains(drawn, "repo") {
		t.Errorf("the knowledge chip pushed the repositories out:\n%s", drawn)
	}
}

// TestNothingKnownDrawsNoChip. A fresh install has learned nothing, and a
// zero on the header is a number nobody needs to read.
func TestNothingKnownDrawsNoChip(t *testing.T) {
	m, _ := testModel(t, 140, 30)

	for _, field := range m.headerFields() {
		if field.name == "knowledge" {
			t.Error("a window that knows nothing still drew the chip")
		}
	}
}

// TestTheChipIsAWayIn. Every chip on this header is something a pointer can
// press, and this one opens the screen the key K opens.
func TestTheChipIsAWayIn(t *testing.T) {
	m := withFacts(t, 2)

	found := false

	for _, field := range m.headerFields() {
		if field.name == "knowledge" {
			found = true
		}
	}

	if !found {
		t.Fatal("the header has no knowledge chip to press")
	}
}

// TestPressingTheChipOpensTheScreen. Every chip on this header is a door, and
// one that draws a number and answers nothing to a click is a button that
// looks broken.
func TestPressingTheChipOpensTheScreen(t *testing.T) {
	m := withFacts(t, 2)

	at := headerCell(t, m, "🧩")
	if got := m.hitHeader(at, m.frame.HeaderLineY()); got.Kind != TargetHeaderField || got.Field != "knowledge" {
		t.Fatalf("the chip's own cell answers %+v, want the knowledge field", got)
	}

	// The press arms it and the release acts, which is how every other chip
	// on this line answers a pointer.
	down, _ := m.mouse(tea.MouseClickMsg{X: at, Y: m.frame.HeaderLineY(), Button: tea.MouseLeft})

	armed, ok := down.(Model)
	if !ok {
		t.Fatalf("the press answered %T", down)
	}

	next, _ := armed.mouse(tea.MouseReleaseMsg{X: at, Y: m.frame.HeaderLineY(), Button: tea.MouseLeft})

	after, ok := next.(Model)
	if !ok {
		t.Fatalf("mouse answered %T", next)
	}

	if after.screen != screenKnowledge {
		t.Errorf("pressing the chip left the window on %v", after.screen)
	}
}
