package ui

import (
	"testing"
)

func TestAllScreensAndPanesViewRendering(t *testing.T) {
	m := openOn(t, "ACME-2690")

	// Inject diff via diffMsg
	next, _ := m.Update(diffMsg{
		ID:   "ACME-2690",
		Text: "+ func NewFeature() bool {\n+   return true\n+ }\n- oldFunction()\n",
	})
	m = asModel(t, next)

	// 1. screenList (Default board view)
	m.screen = screenList

	vList := m.View()
	if vList.Content == "" {
		t.Error("screenList View returned empty string")
	}

	// 2. screenDetail across ALL 11 tabs
	m.screen = screenDetail

	allTabs := []tab{
		tabOverview, tabFlow, tabGates, tabCost, tabRefused,
		tabTimeline, tabReport, tabArtifacts, tabNotes, tabDiff, tabThinking,
	}
	for _, tb := range allTabs {
		m.tab = tb

		vTab := m.View()
		if vTab.Content == "" {
			t.Errorf("screenDetail tab %d View returned empty string", tb)
		}
	}

	// 3. screenStart
	m.screen = screenStart
	if v := m.View(); v.Content == "" {
		t.Error("screenStart View returned empty string")
	}

	// 4. screenCompose
	m.screen = screenCompose
	if v := m.View(); v.Content == "" {
		t.Error("screenCompose View returned empty string")
	}

	// 5. screenSettings
	m.screen = screenSettings
	if v := m.View(); v.Content == "" {
		t.Error("screenSettings View returned empty string")
	}

	// 6. screenFlows
	m.screen = screenFlows
	if v := m.View(); v.Content == "" {
		t.Error("screenFlows View returned empty string")
	}

	// 7. screenRepos
	m.screen = screenRepos
	if v := m.View(); v.Content == "" {
		t.Error("screenRepos View returned empty string")
	}

	// 8. screenEngines
	m.screen = screenEngines
	if v := m.View(); v.Content == "" {
		t.Error("screenEngines View returned empty string")
	}

	// 9. screenHelp
	m.screen = screenHelp
	if v := m.View(); v.Content == "" {
		t.Error("screenHelp View returned empty string")
	}

	// 10. Modals & Overlays over screenList
	m.screen = screenList
	// Palette modal open
	m = m.openPalette()

	m.palette.typed = "rec"
	if v := m.View(); v.Content == "" {
		t.Error("palette modal View returned empty string")
	}

	m = m.closePalette()

	// Menu modal open
	m = m.openMenu("ACME-2690")
	if v := m.View(); v.Content == "" {
		t.Error("menu modal View returned empty string")
	}

	m = m.closeMenu()

	// Confirmation modal open
	m.confirm = confirmCancel
	if v := m.View(); v.Content == "" {
		t.Error("confirmCancel modal View returned empty string")
	}

	m.confirm = confirmNone
}
