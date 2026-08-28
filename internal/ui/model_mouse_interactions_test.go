package ui

import (
	"testing"

	"charm.land/bubbletea/v2"
)

func TestModelInit(t *testing.T) {
	m, _ := testModel(t, 120, 40)

	cmd := m.Init()
	if cmd == nil {
		t.Error("expected non-nil tea.Cmd from Init()")
	}
}

func TestModelMouseInteractions(t *testing.T) {
	m, _ := testModel(t, 120, 40)

	// 1. Mouse wheel up / down
	mUp, _ := m.Update(tea.MouseWheelMsg{
		X:      10,
		Y:      10,
		Button: tea.MouseWheelUp,
	})
	if mUp == nil {
		t.Error("MouseWheelUp returned nil model")
	}

	mDown, _ := m.Update(tea.MouseWheelMsg{
		X:      10,
		Y:      10,
		Button: tea.MouseWheelDown,
	})
	if mDown == nil {
		t.Error("MouseWheelDown returned nil model")
	}

	// 2. Mouse Click & Release on a Row (Left Click selects row)
	mRowClick, _ := m.Update(tea.MouseClickMsg{
		X:      15,
		Y:      5,
		Button: tea.MouseLeft,
	})

	mRowRel, _ := mRowClick.Update(tea.MouseReleaseMsg{
		X:      15,
		Y:      5,
		Button: tea.MouseLeft,
	})
	if mRowRel == nil {
		t.Error("Mouse click/release on row failed")
	}

	// 3. Right Click on a Row (Opens Menu)
	mRightClick, _ := m.Update(tea.MouseClickMsg{
		X:      15,
		Y:      5,
		Button: tea.MouseRight,
	})
	mRightRel, _ := mRightClick.Update(tea.MouseReleaseMsg{
		X:      15,
		Y:      5,
		Button: tea.MouseRight,
	})

	mRightTyped := asModel(t, mRightRel)
	if mRightTyped.menu.open {
		// Context menu opened!
		mClose, _ := mRightTyped.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
		if asModel(t, mClose).menu.open {
			t.Error("expected menu to close on Esc")
		}
	}

	// 4. Mouse Motion (ignored safely)
	mMotion, _ := m.Update(tea.MouseMotionMsg{X: 10, Y: 10})
	if mMotion == nil {
		t.Error("MouseMotion returned nil model")
	}
}

func TestDetailMouseTabsAndButtons(t *testing.T) {
	m, _ := testModel(t, 120, 40)
	// Open detail screen
	m.screen = screenDetail
	m.tab = tabReport

	// Click on tab coordinate
	mTabClick, _ := m.Update(tea.MouseClickMsg{
		X:      20,
		Y:      2,
		Button: tea.MouseLeft,
	})

	mTabRel, _ := mTabClick.Update(tea.MouseReleaseMsg{
		X:      20,
		Y:      2,
		Button: tea.MouseLeft,
	})
	if mTabRel == nil {
		t.Error("tab click failed")
	}
}

func TestSettingsSubmitAndEditing(t *testing.T) {
	m, _ := testModel(t, 120, 40)
	m.screen = screenSettings
	m.settings.editing = true
	m.settings.typed = "es"
	m.settings.sel = 0 // Language setting row

	// Press Enter to submit edited setting
	mSub, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	mSubTyped := asModel(t, mSub)
	if mSubTyped.settings.editing {
		t.Error("expected editing to be false after submit")
	}
}

func TestThemePillActive(t *testing.T) {
	activePill := PillActive("ACTIVE", "#FFFFFF", "#000000")
	if activePill == "" {
		t.Error("PillActive returned empty string")
	}
}
