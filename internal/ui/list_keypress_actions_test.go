package ui

import (
	"testing"

	"charm.land/bubbletea/v2"
	"github.com/e1i0r/orbit/internal/view"
)

func TestAllListKeypressActions(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.board.Tasks = []view.Task{
		{
			ID:       "PAY-1",
			Repo:     "payments",
			RepoPath: "/path/to/payments",
			Engine:   "claude",
		},
		{
			ID:       "PAY-2",
			Repo:     "payments",
			RepoPath: "/path/to/payments",
			Engine:   "claude",
		},
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

	// 1. Navigation keys
	sendKey('j', 0, "") // Down
	sendKey('k', 0, "") // Up
	sendKey('G', 0, "") // Last
	sendKey('g', 0, "") // First
	sendKey(0, tea.KeyPgDown, "pgdown")
	sendKey(0, tea.KeyPgUp, "pgup")

	// 2. Tab jumps (keys 1-9)
	for r := '1'; r <= '9'; r++ {
		sendKey(r, 0, "")
	}

	// 3. Modals and switches
	sendKey('/', 0, "") // Filter
	if !m.filtering {
		t.Error("expected filtering to be true")
	}
	sendKey(0, tea.KeyEsc, "esc") // cancel filter

	sendKey('a', 0, "") // Autopilot toggle
	sendKey('L', 0, "") // Language toggle
	sendKey('?', 0, "") // Help screen
	if m.screen != screenHelp {
		t.Errorf("screen after ? = %v, want screenHelp", m.screen)
	}
	sendKey(0, tea.KeyEsc, "esc") // back to list

	sendKey('M', 0, "") // EngineKnobs (capital 'M')
	if m.screen != screenEngines {
		t.Errorf("screen after M = %v, want screenEngines", m.screen)
	}
	sendKey(0, tea.KeyEsc, "esc") // back to list

	// 4. Task actions
	sendKey('p', 0, "") // Pause
	sendKey('r', 0, "") // Resume
	sendKey('h', 0, "") // HandBack
	sendKey('x', 0, "") // Cancel / Ask confirm
	sendKey('t', 0, "") // Take
	sendKey('v', 0, "") // MarkRead
	sendKey('A', 0, "") // Ask
	sendKey('n', 0, "") // Start dialog
	sendKey(0, tea.KeyEsc, "esc")
}
