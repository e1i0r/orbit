package ui

import (
	"testing"

	"charm.land/bubbletea/v2"
)

func TestEnginesScreenFullLifecycle(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	// 1. Open engines screen
	m = m.openEngines()
	if m.screen != screenEngines {
		t.Fatalf("screen = %v, want screenEngines", m.screen)
	}

	sendKey := func(k rune, code rune, text string) {
		var msg tea.Msg

		switch {
		case text != "":
			msg = tea.KeyPressMsg{Code: code, Text: text}
		case code != 0:
			msg = tea.KeyPressMsg{Code: code}
		default:
			msg = tea.KeyPressMsg{Code: k, Text: string(k)}
		}

		updated, _ := m.Update(msg)
		m = asModel(t, updated)
	}

	// 2. Navigate rows with Down / Up
	sendKey('j', 0, "")
	sendKey('j', 0, "")
	sendKey('k', 0, "")

	// 3. Select with Enter
	sendKey(0, tea.KeyEnter, "")

	// 4. knobChip testing
	m.knobs = Knobs{
		Engine:   "claude",
		Model:    "sonnet-3.7",
		Effort:   "high",
		Thinking: "adaptive",
	}

	chip := m.knobChip()
	if chip == "" {
		t.Error("expected non-empty knobChip with overrides")
	}

	// 5. Render engines view
	v := m.View()
	if len(v.Content) == 0 {
		t.Error("expected non-empty engines view content")
	}

	// 6. Hit detection on engines screen
	_ = m.hit(10, 4)
	_ = m.hit(10, 8)

	// 7. Escape to close. The walk above may have landed ⏎ on an engine
	// that needs setting up, which puts its steps on screen; the first Esc
	// takes those down and the second closes the knobs.
	if m.engines.showingSetup {
		sendKey(0, tea.KeyEsc, "esc")
	}

	sendKey(0, tea.KeyEsc, "esc")

	if m.screen != screenList {
		t.Errorf("screen after Esc = %v, want screenList", m.screen)
	}
}
