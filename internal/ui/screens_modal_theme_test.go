package ui

import (
	"testing"
)

func TestModalScreensNavigationAndRender(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Command Palette
	m.palette.open = true
	if lines := m.paletteRows(20, 100); len(lines) == 0 {
		t.Error("expected paletteRows to render")
	}

	m.palette.open = false

	// 2. Menu Rows
	m.menu.open = true
	if lines := m.menuRows(20, 100); len(lines) == 0 {
		t.Error("expected menuRows to render")
	}

	m.menu.open = false

	// 3. Engines screen
	m.screen = screenEngines
	if lines := m.enginesRows(20, 100); len(lines) == 0 {
		t.Error("expected enginesRows to render")
	}

	// 4. Flows screen & Flows Builder
	m.screen = screenFlows
	if lines := m.flowsRows(20, 100); len(lines) == 0 {
		t.Error("expected flowsRows to render")
	}

	// 5. Repos screen
	m.screen = screenRepos
	if lines := m.repolistRows(20, 100); len(lines) == 0 {
		t.Error("expected repolistRows to render")
	}

	// 6. Help screen
	m.screen = screenHelp
	if lines := m.helpRows(20, 100); len(lines) == 0 {
		t.Error("expected helpRows to render")
	}

	// 7. Settings screen
	m.screen = screenSettings
	if lines := m.settingsRows(20, 100); len(lines) == 0 {
		t.Error("expected settingsRows to render")
	}

	// 8. Compose screen
	m.screen = screenCompose
	if lines := m.composeRows(20, 100); len(lines) == 0 {
		t.Error("expected composeRows to render")
	}
}

func TestStartDialogDialsAndKeys(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	newM, _ := m.openStart()
	m = asModel(t, newM)

	if m.screen == screenStart {
		// Cycle dials
		m = m.cycleFlow()
		m = m.cycleEffort()
		m = m.cycleThinking()

		// Render start lines
		lines := m.startRows(m.frame.Body.H, m.frame.Body.W)
		if len(lines) == 0 {
			t.Error("expected startRows to render")
		}
	}
}

func TestThemeSwitchingAndPalettes(t *testing.T) {
	for _, themeName := range AvailableThemes() {
		SetCurrentTheme(themeName)

		palette := currentPalette()
		if palette.OK == "" || palette.Accent == "" {
			t.Errorf("theme %q has empty OK or Accent tokens", themeName)
		}
	}
	// Reset to default frauddi
	SetCurrentTheme("frauddi")
}
