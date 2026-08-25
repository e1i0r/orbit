package ui

import (
	"testing"

	"charm.land/bubbletea/v2"
)

func TestAllScreensKeyboardFlows(t *testing.T) {
	m, _ := testModel(t, 120, 40)

	// 1. Help screen keyboard flow
	m.screen = screenHelp
	mNext, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if asModel(t, mNext).screen != screenList {
		t.Error("expected Esc from help screen to return to screenList")
	}

	// 2. Engines screen keyboard flow
	m.screen = screenEngines
	mNext, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if asModel(t, mNext).screen != screenList {
		t.Error("expected Esc from engines screen to return to screenList")
	}

	// 3. Repos screen keyboard flow
	m.screen = screenRepos
	mNext, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if asModel(t, mNext).screen != screenList {
		t.Error("expected Esc from repos screen to return to screenList")
	}

	// 4. Flows screen keyboard flow
	m.screen = screenFlows
	mNext, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyEsc}) // confirm discard prompt
	mNext, _ = mNext.Update(tea.KeyPressMsg{Text: "y"})        // confirm discard
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyEsc}) // exits screenFlows
	if asModel(t, mNext).screen != screenList {
		t.Error("expected Esc from flows screen to return to screenList")
	}

	// 5. Settings screen keyboard flow
	m.screen = screenSettings
	mNext, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if asModel(t, mNext).screen != screenList {
		t.Error("expected Esc from settings screen to return to screenList")
	}

	// 6. Start task dialog keyboard flow
	m.screen = screenStart
	mNext, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Text: " "}) // toggle autopilot
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if asModel(t, mNext).screen != screenList {
		t.Error("expected Esc from start dialog to return to screenList")
	}

	// 7. Compose task dialog keyboard flow
	m.screen = screenCompose
	mNext, _ = m.Update(tea.KeyPressMsg{Text: "P"})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Text: "A"})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Text: "Y"})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Text: "T"})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Text: "a"})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Text: "s"})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Text: "k"})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if asModel(t, mNext).screen != screenList {
		t.Error("expected Esc from compose dialog to return to screenList")
	}

	// 8. Detail screen keyboard flow (switching tabs 1..9, 0, w)
	m.screen = screenDetail
	for _, keyStr := range []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0", "w"} {
		mNext, _ = m.Update(tea.KeyPressMsg{Text: keyStr})
	}
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	mNext, _ = mNext.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if asModel(t, mNext).screen != screenList {
		t.Error("expected Esc from detail screen to return to screenList")
	}

	// 9. Filter typing flow on screenList
	m.screen = screenList
	mFilter, _ := m.Update(tea.KeyPressMsg{Text: "/"})
	mFilter, _ = mFilter.Update(tea.KeyPressMsg{Text: "p"})
	mFilter, _ = mFilter.Update(tea.KeyPressMsg{Text: "a"})
	mFilter, _ = mFilter.Update(tea.KeyPressMsg{Text: "y"})
	mFilter, _ = mFilter.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	mFilter, _ = mFilter.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mFilterTyped := asModel(t, mFilter)
	if mFilterTyped.filtering {
		t.Error("expected filtering to end on Enter")
	}
}
