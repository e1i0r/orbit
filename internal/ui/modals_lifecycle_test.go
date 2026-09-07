package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/palette"
)

func TestPaletteOperations(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = []Command{
		{Name: "settings"},
		{Name: "flows"},
		{Name: "repos"},
	}

	// 1. Open and close palette
	m = m.openPalette()
	if !m.palette.Up() {
		t.Error("expected palette to be open")
	}

	// 2. Typing into palette
	m.palette = palette.OpenWith("set")

	if rows := strings.Join(m.paletteRows(20, 100), "\n"); !strings.Contains(rows, "settings") {
		t.Errorf("typing set offered:\n%s", rows)
	}

	// 3. Navigation with up/down keys
	_, _ = m.paletteKey(tea.KeyPressMsg{Text: "down"})
	_, _ = m.paletteKey(tea.KeyPressMsg{Text: "up"})
	_, _ = m.paletteKey(tea.KeyPressMsg{Text: "tab"})

	// 4. Backspace
	_, _ = m.paletteKey(tea.KeyPressMsg{Code: tea.KeyBackspace})

	// 5. Render palette rows
	rows := m.paletteRows(20, 100)
	if len(rows) == 0 {
		t.Error("expected paletteRows to render lines")
	}

	// 6. Hit testing on palette
	_ = m.hitPalette(10, 8)

	// 7. Escape takes the line down
	left, _ := m.paletteKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if asModel(t, left).palette.Up() {
		t.Error("expected palette to be closed")
	}
}

func TestMenuOperations(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Open context menu
	m = m.openMenuForContext()
	if !m.menu.Up() {
		t.Error("expected menu to be open")
	}

	// 2. Entries
	entries := m.menu.Entries(m.menuEnv())
	if len(entries) == 0 {
		t.Error("expected non-empty menu entries")
	}

	// 3. Menu navigation
	_, _ = m.menuKey(tea.KeyPressMsg{Text: "down"})
	_, _ = m.menuKey(tea.KeyPressMsg{Text: "up"})

	// 4. Hit testing
	_ = m.hitMenu(10, 5)

	// 5. Close menu
	m = m.closeMenu()
	if m.menu.Up() {
		t.Error("expected menu to be closed")
	}
}

func TestSettingsModalOperations(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Open settings
	m = m.openSettings()
	if m.screen != screenSettings {
		t.Errorf("screen = %v, want screenSettings", m.screen)
	}

	// 2. Cycle settings dials
	_, _ = m.cycleSetting(1)
	_, _ = m.cycleSetting(-1)

	// 3. Hit testing
	_ = m.hitSettings(10, 10)

	// 4. Abandon settings
	next, _ := m.settingsKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = asModel(t, next)

	if m.screen == screenSettings {
		t.Error("expected to leave screenSettings")
	}
}

func TestRepolistModalOperations(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Open repos
	m = m.openRepos()
	if m.screen != screenRepos {
		t.Errorf("screen = %v, want screenRepos", m.screen)
	}

	// 2. Repolist key navigation
	_, _ = m.repolistKey(tea.KeyPressMsg{Text: "down"})
	_, _ = m.repolistKey(tea.KeyPressMsg{Text: "up"})

	// 3. Hit testing
	_ = m.hitRepos(10, 10)

	// 4. Escape leaves it
	left, _ := m.repolistKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if asModel(t, left).screen == screenRepos {
		t.Error("expected to leave screenRepos")
	}
}

func TestWatchSessionOperations(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Write to commandWatch buffer
	w := &commandWatch{name: "test-cmd"}
	if _, err := w.Write([]byte("compiling packages...\nall tests passed\n")); err != nil {
		t.Fatal(err)
	}

	out, done := w.snapshot()
	if out == "" || done {
		t.Errorf("unexpected snapshot: out=%q, done=%v", out, done)
	}

	// 2. Finish watch
	w.finish()

	_, done = w.snapshot()
	if !done {
		t.Error("expected done=true after finish")
	}

	// 3. Render watch rows
	rows := m.watchRows(20, 100)
	if len(rows) == 0 {
		t.Error("expected watchRows to return lines")
	}

	// 4. Close and reopen watch
	m = m.closeWatch()

	m = m.reopenWatch()
	if m.screen != screenList {
		t.Errorf("unexpected screen after reopenWatch: %v", m.screen)
	}
}

func TestHitTestingCoordinates(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Header hit test
	_ = m.hitHeader(10, 0)
	_ = m.hitHeader(50, 0)

	// 2. Status bar hit test
	_ = m.hitStatus(10, 1)

	// 3. Body row hit test
	_ = m.hitRow(10, 3)

	// 4. Bar footer hit test
	_ = m.hitBar(10, 29)

	// 5. Global hit function
	_ = m.hit(10, 0)
	_ = m.hit(10, 1)
	_ = m.hit(10, 3)
	_ = m.hit(10, 29)
}
