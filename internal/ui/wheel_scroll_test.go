package ui

// wheel_coverage_test.go is the wheel's own routing: which of the screens
// it might land on it actually scrolls, and the one binding-with-no-keys
// edge firstKey exists to answer safely.

import (
	"testing"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// paneBodyY is the first row of the body that hits the task view's pane
// rather than its tabs, found by scanning rather than assumed — the tab
// strip's own height is detail.go's business, not this file's.
func paneBodyY(t *testing.T, m Model) int {
	t.Helper()
	for y := m.frame.Body.Y; y < m.frame.Body.Y+m.frame.Body.H; y++ {
		if m.hit(m.frame.Body.W/2, y).Kind == TargetPaneBody {
			return y
		}
	}
	t.Fatal("no row in the body hit the task view's pane")
	return 0
}

func TestWheelOutsideTheBodyOrWithAnUnhandledButtonDoesNothing(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Above the body entirely (the header).
	if after := m.wheel(tea.Mouse{X: 5, Y: 0, Button: tea.MouseWheelDown}); after.cursor != m.cursor {
		t.Error("wheel over the header moved the cursor")
	}

	// 2. A button the wheel does not answer for, even inside the body.
	if after := m.wheel(tea.Mouse{X: 5, Y: m.frame.Body.Y, Button: tea.MouseLeft}); after.cursor != m.cursor {
		t.Error("wheel with a non-wheel button moved the cursor")
	}
}

func TestWheelScrollsWhicheverScreenIsUnderIt(t *testing.T) {
	// 1. The task view scrolls its pane, and only when the pointer is over
	// the pane rather than its tabs.
	m := openOn(t, "ACME-2705")
	m.tab = tabLog
	// Enough entries that the pane actually has something below the fold
	// to scroll to — the fixture task view opens with none at all.
	entries := make([]view.Entry, 0, 60)
	for i := range 60 {
		entries = append(entries, view.Entry{Kind: "phase.started", Phase: "implement", At: ago(time.Duration(60-i) * time.Minute)})
	}
	m.entries = entries
	m = m.syncPanes()
	y := paneBodyY(t, m)
	afterDown := m.wheel(tea.Mouse{X: m.frame.Body.W / 2, Y: y, Button: tea.MouseWheelDown})
	if afterDown.panes[m.tab].YOffset() == m.panes[m.tab].YOffset() {
		t.Error("wheel down over the pane did not scroll it")
	}
	afterUp := afterDown.wheel(tea.Mouse{X: m.frame.Body.W / 2, Y: y, Button: tea.MouseWheelUp})
	if afterUp.panes[m.tab].YOffset() != m.panes[m.tab].YOffset() {
		t.Errorf("wheel up should have wound the pane back to %d, got %d", m.panes[m.tab].YOffset(), afterUp.panes[m.tab].YOffset())
	}
	if after := m.wheel(tea.Mouse{X: 1, Y: m.frame.Body.Y, Button: tea.MouseWheelDown}); after.panes[m.tab].YOffset() != m.panes[m.tab].YOffset() {
		t.Error("wheel over the tab strip, rather than the pane, scrolled it anyway")
	}

	// 2. The menu's selection moves under the wheel.
	menu, _ := testModel(t, 100, 30)
	menu.opts.Commands = []Command{{Name: "new"}, {Name: "repos"}, {Name: "flows"}}
	menu = menu.openMenu("")
	afterMenuDown := menu.wheel(tea.Mouse{X: 5, Y: menu.frame.Body.Y, Button: tea.MouseWheelDown})
	if afterMenuDown.menu.sel == menu.menu.sel {
		t.Error("wheel down over an open menu did not move its selection")
	}
	afterMenuUp := afterMenuDown.wheel(tea.Mouse{X: 5, Y: menu.frame.Body.Y, Button: tea.MouseWheelUp})
	if afterMenuUp.menu.sel != menu.menu.sel {
		t.Errorf("wheel up should have wound the selection back to %d, got %d", menu.menu.sel, afterMenuUp.menu.sel)
	}

	// 3. The palette's selection moves the same way.
	pal, _ := testModel(t, 100, 30)
	pal.opts.Commands = []Command{{Name: "new"}, {Name: "repos"}, {Name: "flows"}}
	pal = pal.openPalette()
	afterPalDown := pal.wheel(tea.Mouse{X: 5, Y: pal.frame.Body.Y, Button: tea.MouseWheelDown})
	if afterPalDown.palette.sel == pal.palette.sel {
		t.Error("wheel down over the palette did not move its selection")
	}

	// 4. A run's output offers nothing to scroll back for.
	watch, _ := testModel(t, 100, 30)
	watch.watchUp = true
	if after := watch.wheel(tea.Mouse{X: 5, Y: watch.frame.Body.Y, Button: tea.MouseWheelDown}); after.offset != watch.offset {
		t.Error("wheel over a run's output moved something")
	}

	// 5. A screen the wheel has no opinion about (settings) is left alone.
	settingsM, _ := testModel(t, 100, 30)
	settingsM.screen = screenSettings
	if after := settingsM.wheel(tea.Mouse{X: 5, Y: settingsM.frame.Body.Y, Button: tea.MouseWheelDown}); after.cursor != settingsM.cursor {
		t.Error("wheel over the settings screen moved the board's cursor")
	}

	// 6. The board itself scrolls by moving the cursor.
	board, _ := testModel(t, 100, 30)
	afterBoardDown := board.wheel(tea.Mouse{X: 5, Y: board.frame.Body.Y, Button: tea.MouseWheelDown})
	if afterBoardDown.cursor == board.cursor {
		t.Error("wheel down over the board did not move the cursor")
	}
	afterBoardUp := board.wheel(tea.Mouse{X: 5, Y: board.frame.Body.Y + 5, Button: tea.MouseWheelUp})
	if afterBoardUp.cursor < 0 {
		t.Error("wheel up over the board moved the cursor below zero, want it clamped")
	}
}

func TestFirstKeyAnswersEmptyForABindingWithNoKeys(t *testing.T) {
	if k := firstKey(key.NewBinding()); k != "" {
		t.Errorf("firstKey on a binding with no keys = %q, want empty", k)
	}
	m, _ := testModel(t, 100, 30)
	if k := firstKey(m.keys.Down); k == "" {
		t.Error("firstKey on a real binding answered empty")
	}
}
