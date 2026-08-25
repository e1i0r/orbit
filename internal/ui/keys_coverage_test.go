package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestPaletteKeyNavigationAndExecution(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Open palette with :
	m.palette.open = true
	m.palette.typed = "settings"

	// 2. Down / Up navigation in palette
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	newM, _ := m.paletteKey(downKey)
	m = asModel(t, newM)

	upKey := tea.KeyPressMsg{Code: tea.KeyUp}
	newM, _ = m.paletteKey(upKey)
	m = asModel(t, newM)

	// 3. Backspace in palette
	backKey := tea.KeyPressMsg{Code: tea.KeyBackspace}
	newM, _ = m.paletteKey(backKey)
	m = asModel(t, newM)

	// 4. Enter key in palette
	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}
	newM, _ = m.paletteKey(enterKey)
	m = asModel(t, newM)

	// 5. Escape closes palette
	m.palette.open = true
	escKey := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM, _ = m.paletteKey(escKey)
	m = asModel(t, newM)
	if m.palette.open {
		t.Error("expected palette to be closed after Esc")
	}
}

func TestEnginesKeyNavigationAndToggles(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.screen = screenEngines

	// Navigation in engines screen
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	newM, _ := m.enginesKey(downKey)
	m = asModel(t, newM)

	// Enter selects engine/model/effort/thinking
	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}
	newM, _ = m.enginesKey(enterKey)
	m = asModel(t, newM)

	// Back key closes engines
	escKey := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM, _ = m.enginesKey(escKey)
	m = asModel(t, newM)
	if m.screen == screenEngines {
		t.Error("expected screenEngines to close after Esc")
	}
}

func TestFlowsKeyNavigationAndEditor(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.screen = screenFlows

	// 1. Navigate flows list
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	newM, _ := m.flowsKey(downKey)
	m = asModel(t, newM)

	// 2. Open flows builder with 'n'
	nKey := tea.KeyPressMsg{Code: 'n', Text: "n"}
	newM, _ = m.flowsKey(nKey)
	m = asModel(t, newM)
	if !m.flows.creating {
		t.Error("expected m.flows.creating to be true after 'n'")
	}

	// 3. Navigate inside flows editor
	newM, _ = m.flowsKey(downKey)
	m = asModel(t, newM)

	// 4. Exit builder with Esc
	escKey := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM, _ = m.flowsKey(escKey)
	m = asModel(t, newM)
}

func TestSettingsKeyNavigationAndToggles(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.screen = screenSettings

	// Navigate settings rows
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	newM, _ := m.settingsKey(downKey)
	m = asModel(t, newM)

	// Space/Right cycle setting value
	spaceKey := tea.KeyPressMsg{Code: ' ', Text: " "}
	newM, _ = m.settingsKey(spaceKey)
	m = asModel(t, newM)

	rightKey := tea.KeyPressMsg{Code: tea.KeyRight}
	newM, _ = m.settingsKey(rightKey)
	m = asModel(t, newM)

	// Esc closes settings
	escKey := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM, _ = m.settingsKey(escKey)
	m = asModel(t, newM)
	if m.screen == screenSettings {
		t.Error("expected screenSettings to close after Esc")
	}
}

func TestHelpAndMenuKeys(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Help screen
	m.screen = screenHelp
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	newM, _ := m.helpKey(downKey)
	m = asModel(t, newM)

	escKey := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM, _ = m.helpKey(escKey)
	m = asModel(t, newM)

	// 2. Menu popup
	m.menu.open = true
	m.menu.sel = 0
	newM, _ = m.menuKey(downKey)
	m = asModel(t, newM)

	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}
	newM, _ = m.menuKey(enterKey)
	m = asModel(t, newM)
}

func TestFilterAndComposeKeys(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Filter typing and clearing
	m.filtering = true
	charKey := tea.KeyPressMsg{Code: 'p', Text: "p"}
	newM, _ := m.filterKey(charKey)
	m = asModel(t, newM)

	escKey := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM, _ = m.filterKey(escKey)
	m = asModel(t, newM)
	if m.filtering {
		t.Error("expected filtering to be false after Esc")
	}

	// 2. Compose screen
	m.screen = screenCompose
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	newM, _ = m.composeKey(downKey)
	m = asModel(t, newM)

	newM, _ = m.composeKey(escKey)
	m = asModel(t, newM)
}
