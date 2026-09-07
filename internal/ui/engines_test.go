package ui

// The engine knobs as the window reaches them: the key that opens them, the
// keystrokes routed to them, and the chip they leave in the header. What the
// screen itself does is tested in internal/ui/engines.

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/engines"
	"github.com/e1i0r/orbit/internal/ui/roster"
	"github.com/e1i0r/orbit/internal/words"
)

// enginesTestList is a small engine roster with one available engine that
// has both dials, one disabled engine with setup steps, and one available
// engine with neither dial — every shape collectEngineRows branches on,
// none of which the fixture's default single-engine fallback offers.
func enginesTestList() []roster.Engine {
	return []roster.Engine{
		{
			Name:      "claude",
			Available: true,
			Models:    []roster.Choice{{ID: "", Label: "default"}, {ID: "opus", Label: "opus"}, {ID: "sonnet", Label: "sonnet"}},
			Efforts:   []roster.Choice{{ID: "", Label: "default"}, {ID: "low", Label: "low"}, {ID: "high", Label: "high"}},
			CanThink:  true,
		},
		{
			// Not installed, and still carrying its dials: what an engine
			// offers and whether this machine can run it are two facts,
			// and the port answers both for every engine.
			Name:      "codex",
			Available: false,
			Setup:     func(*words.Printer) []string { return []string{"install codex", "run codex login"} },
			Models:    []roster.Choice{{ID: "", Label: "default"}, {ID: "o3", Label: "o3"}, {ID: "o3-mini", Label: "o3-mini"}},
			Efforts:   []roster.Choice{{ID: "", Label: "default"}, {ID: "low", Label: "low"}, {ID: "high", Label: "high"}},
		},
		{
			Name:      "bare",
			Available: true,
		},
	}
}

// TestTheKnobsOpenAndCloseAndLeaveTheirChipBehind, which is the whole of the
// window's part in this screen: routing the keys and keeping the dials.
func TestTheKnobsOpenAndCloseAndLeaveTheirChipBehind(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.opts.Engines = enginesTestList

	m = m.openEngines()
	if m.screen != screenEngines {
		t.Fatalf("screen = %v, want the knobs", m.screen)
	}

	send := func(msg tea.Msg) {
		t.Helper()

		updated, _ := m.Update(msg)
		m = asModel(t, updated)
	}

	// The arrows walk the list and ⏎ takes what is under the cursor.
	send(tea.KeyPressMsg{Code: 'j', Text: "j"})
	send(tea.KeyPressMsg{Code: 'j', Text: "j"})
	send(tea.KeyPressMsg{Code: 'k', Text: "k"})
	send(tea.KeyPressMsg{Code: tea.KeyEnter})

	// The dials are the window's, and the chip is what the header says of
	// them.
	m.knobs = engines.Knobs{Engine: "claude", Model: "sonnet", Effort: "high", Thinking: "adaptive"}
	m.engines = engines.Open(int(screenList), m.knobs)

	if chip := m.knobChip(); chip == "" {
		t.Error("dials that are all set draw no chip")
	}

	if v := m.View(); len(v.Content) == 0 {
		t.Error("the knobs screen drew nothing")
	}

	// The cells the list is drawn in answer a click.
	_ = m.hit(10, 4)
	_ = m.hit(10, 8)

	// Escape closes it. The walk above may have landed ⏎ on an engine that
	// needs setting up, which puts its steps on screen; the first esc takes
	// those down and the second closes the knobs.
	send(tea.KeyPressMsg{Code: tea.KeyEsc})
	send(tea.KeyPressMsg{Code: tea.KeyEsc})

	if m.screen != screenList {
		t.Errorf("screen after esc = %v, want the board", m.screen)
	}
}
