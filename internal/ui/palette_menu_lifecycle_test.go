package ui

import (
	"testing"

	"charm.land/bubbletea/v2"
	"github.com/e1i0r/orbit/internal/view"
)

func TestPaletteAndMenuFullLifecycle(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.board.Tasks = []view.Task{
		{ID: "TASK-1", Repo: "repo", Engine: "claude"},
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

	// 1. Open Palette with ':'
	sendKey(':', 0, "")
	if !m.palette.open {
		t.Error("expected palette to be open")
	}

	// Type in palette
	sendKey('s', 0, "")
	sendKey('e', 0, "")
	sendKey('t', 0, "")

	// Render palette view
	v := m.View()
	if len(v.Content) == 0 {
		t.Error("expected non-empty palette view")
	}

	// Hit detection in palette
	_ = m.hit(10, 4)
	_ = m.hit(10, 28)

	// Backspace in palette
	sendKey(0, tea.KeyBackspace, "")

	// Up / Down in palette
	sendKey('j', 0, "")
	sendKey('k', 0, "")

	// Close palette with Esc
	sendKey(0, tea.KeyEsc, "esc")
	if m.palette.open {
		t.Error("expected palette to be closed")
	}

	// 2. Open Menu with 'm'
	sendKey('m', 0, "")
	if !m.menu.open {
		t.Error("expected menu to be open")
	}

	// Render menu view
	v = m.View()
	if len(v.Content) == 0 {
		t.Error("expected non-empty menu view")
	}

	// Hit detection in menu
	_ = m.hit(10, 4)

	// Navigate menu with Up / Down
	sendKey('j', 0, "")
	sendKey('k', 0, "")

	// Close menu with Esc
	sendKey(0, tea.KeyEsc, "esc")
	if m.menu.open {
		t.Error("expected menu to be closed")
	}

	// 3. Alias check
	if !matchesSettingsAlias("conf") || !matchesSettingsAlias("ajus") {
		t.Error("expected matchesSettingsAlias to match valid prefixes")
	}
}
