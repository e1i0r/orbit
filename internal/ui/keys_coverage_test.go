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
	m = newM.(Model)

	upKey := tea.KeyPressMsg{Code: tea.KeyUp}
	newM, _ = m.paletteKey(upKey)
	m = newM.(Model)

	// 3. Backspace in palette
	backKey := tea.KeyPressMsg{Code: tea.KeyBackspace}
	newM, _ = m.paletteKey(backKey)
	m = newM.(Model)

	// 4. Enter key in palette
	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}
	newM, _ = m.paletteKey(enterKey)
	m = newM.(Model)

	// 5. Escape closes palette
	m.palette.open = true
	escKey := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM, _ = m.paletteKey(escKey)
	m = newM.(Model)
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
	m = newM.(Model)

	// Enter selects engine/model/effort/thinking
	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}
	newM, _ = m.enginesKey(enterKey)
	m = newM.(Model)

	// Back key closes engines
	escKey := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM, _ = m.enginesKey(escKey)
	m = newM.(Model)
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
	m = newM.(Model)

	// 2. Open flows builder with 'n'
	nKey := tea.KeyPressMsg{Code: 'n', Text: "n"}
	newM, _ = m.flowsKey(nKey)
	m = newM.(Model)
	if !m.flows.creating {
		t.Error("expected m.flows.creating to be true after 'n'")
	}

	// 3. Navigate inside flows editor
	newM, _ = m.flowsKey(downKey)
	m = newM.(Model)

	// 4. Exit builder with Esc
	escKey := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM, _ = m.flowsKey(escKey)
	m = newM.(Model)
}

func TestSettingsKeyNavigationAndToggles(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.screen = screenSettings

	// Navigate settings rows
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	newM, _ := m.settingsKey(downKey)
	m = newM.(Model)

	// Space/Right cycle setting value
	spaceKey := tea.KeyPressMsg{Code: ' ', Text: " "}
	newM, _ = m.settingsKey(spaceKey)
	m = newM.(Model)

	rightKey := tea.KeyPressMsg{Code: tea.KeyRight}
	newM, _ = m.settingsKey(rightKey)
	m = newM.(Model)

	// Esc closes settings
	escKey := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM, _ = m.settingsKey(escKey)
	m = newM.(Model)
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
	m = newM.(Model)

	escKey := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM, _ = m.helpKey(escKey)
	m = newM.(Model)

	// 2. Menu popup
	m.menu.open = true
	m.menu.sel = 0
	newM, _ = m.menuKey(downKey)
	m = newM.(Model)

	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}
	newM, _ = m.menuKey(enterKey)
	m = newM.(Model)
}

func TestFilterAndComposeKeys(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Filter typing and clearing
	m.filtering = true
	charKey := tea.KeyPressMsg{Code: 'p', Text: "p"}
	newM, _ := m.filterKey(charKey)
	m = newM.(Model)

	escKey := tea.KeyPressMsg{Code: tea.KeyEsc}
	newM, _ = m.filterKey(escKey)
	m = newM.(Model)
	if m.filtering {
		t.Error("expected filtering to be false after Esc")
	}

	// 2. Compose screen
	m.screen = screenCompose
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	newM, _ = m.composeKey(downKey)
	m = newM.(Model)

	newM, _ = m.composeKey(escKey)
	m = newM.(Model)
}
