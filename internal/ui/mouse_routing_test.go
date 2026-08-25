package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/e1i0r/orbit/internal/view"
)

func TestMouseClickRoutingAcrossViews(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Click on header fields
	newM, _ := m.leftClick(Target{Kind: TargetHeaderField, Field: "autopilot"})
	m = asModel(t, newM)

	newM, _ = m.leftClick(Target{Kind: TargetHeaderField, Field: "lang"})
	m = asModel(t, newM)

	newM, _ = m.leftClick(Target{Kind: TargetHeaderField, Field: "orbit"})
	m = asModel(t, newM)

	// 2. Click on task row to move and open
	newM, _ = m.leftClick(Target{Kind: TargetTask, ID: "ACME-2662"})
	m = asModel(t, newM)
	newM, _ = m.leftClick(Target{Kind: TargetTask, ID: "ACME-2662"})
	m = asModel(t, newM)
	if m.screen != screenDetail || m.detail != "ACME-2662" {
		t.Fatalf("expected screenDetail on ACME-2662, got screen=%v, detail=%q", m.screen, m.detail)
	}

	// 3. Click on tabs in detail
	newM, _ = m.leftClick(Target{Kind: TargetPaneTab, Pane: int(tabFlow)})
	m = asModel(t, newM)
	if m.tab != tabFlow {
		t.Errorf("expected tabFlow, got %v", m.tab)
	}

	newM, _ = m.leftClick(Target{Kind: TargetPaneTab, Pane: int(tabCost)})
	m = asModel(t, newM)
	if m.tab != tabCost {
		t.Errorf("expected tabCost, got %v", m.tab)
	}

	// 4. Click band header to toggle fold
	newM, _ = m.leftClick(Target{Kind: TargetBandHeader, Band: view.NeedsYou})
	m = asModel(t, newM)

	// 5. Open settings screen and click option pill
	m.screen = screenSettings
	newM, _ = m.leftClick(Target{Kind: TargetSettingsRow, Pane: 0, Field: "es"})
	m = asModel(t, newM)

	// 6. Right click on task opens menu
	m.screen = screenList
	newM, _ = m.rightClick(Target{Kind: TargetTask, ID: "ACME-2662"})
	m = asModel(t, newM)
}

func TestMouseWheelAndKeyHintClick(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// Key hint click
	newM, _ := m.leftClick(Target{Kind: TargetBarHint, Key: "?"})
	m = asModel(t, newM)

	// Wheel scrolling
	wheelDown := tea.MouseWheelMsg{
		X: 50, Y: 10,
		Button: tea.MouseWheelDown,
	}
	mWheel := m.wheel(wheelDown.Mouse())
	if mWheel.cursor < 0 {
		t.Error("cursor should be >= 0 after wheel")
	}

	wheelUp := tea.MouseWheelMsg{
		X: 50, Y: 10,
		Button: tea.MouseWheelUp,
	}
	_ = m.wheel(wheelUp.Mouse())
}

func TestMouseIsSwallowedByAConfirmAndIgnoresMotion(t *testing.T) {
	// 1. A question on screen swallows every pointer event, click included.
	m, _ := testModel(t, 100, 30)
	m.confirm = confirmCancel
	next, cmd := m.mouse(tea.MouseClickMsg{X: 5, Y: 5, Button: tea.MouseLeft})
	if cmd != nil {
		t.Error("mouse during a confirm produced a command")
	}
	if asModel(t, next).held.down {
		t.Error("mouse during a confirm still recorded the press")
	}

	// 2. A plain click and release without a confirm records, then clears,
	// the hold.
	m2, _ := testModel(t, 100, 30)
	next2, _ := m2.mouse(tea.MouseClickMsg{X: 5, Y: m2.frame.Body.Y, Button: tea.MouseLeft})
	after2 := asModel(t, next2)
	if !after2.held.down {
		t.Error("a plain click did not record the press")
	}

	// 3. Motion is ignored outright.
	next3, cmd3 := after2.mouse(tea.MouseMotionMsg{X: 5, Y: 5})
	if cmd3 != nil || asModel(t, next3).held != after2.held {
		t.Error("mouse motion should leave the window exactly as it was")
	}
}

func TestReleaseOnlyActsWhenThePressAndReleaseAgree(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. A release with no press held down does nothing.
	next, cmd := m.release(tea.Mouse{X: 5, Y: 5, Button: tea.MouseLeft})
	if cmd != nil || asModel(t, next).screen != screenList {
		t.Error("release with nothing held should do nothing")
	}

	// 2. A press held on one target, released over another: the gesture is
	// cancelled, exactly the way dragging off a button cancels a click.
	m2 := onRow(t, m, "ACME-2662")
	m2.held = hold{target: Target{Kind: TargetTask, ID: "ACME-2662"}, button: tea.MouseLeft, down: true}
	next2, _ := m2.release(tea.Mouse{X: 5, Y: 5, Button: tea.MouseLeft})
	if asModel(t, next2).screen != screenList {
		t.Error("release over a different target should not open the task")
	}

	// 3. A button this window has no gesture for (the wheel button read as
	// a click) answers nothing even when press and release agree.
	m3 := onRow(t, m, "ACME-2662")
	tgt := Target{Kind: TargetTask, ID: "ACME-2662"}
	m3.held = hold{target: tgt, button: tea.MouseMiddle, down: true}
	i, ok := m3.rowOf(tgt)
	if !ok {
		t.Fatal("no row for ACME-2662")
	}
	y := m3.frame.Body.Y + i
	next3, cmd3 := m3.release(tea.Mouse{X: 5, Y: y, Button: tea.MouseMiddle})
	if cmd3 != nil || asModel(t, next3).screen != screenList {
		t.Error("release of a button with no gesture should do nothing")
	}
}

func TestLeftClickSettingsEngineAndCommandBranches(t *testing.T) {
	// TargetSettingsRow with no field, or out of range: cycles instead of
	// applying a specific option.
	m, _ := testModel(t, 100, 30)
	m.screen = screenSettings
	before := m.opts.Settings.(*settings).lang //nolint:errcheck
	next, _ := m.leftClick(Target{Kind: TargetSettingsRow, Pane: 0, Field: ""})
	after := asModel(t, next).opts.Settings.(*settings).lang //nolint:errcheck
	if after == before {
		t.Error("clicking a settings row with no field should still cycle it")
	}
	next2, cmd2 := m.leftClick(Target{Kind: TargetSettingsRow, Pane: 999, Field: ""})
	if cmd2 != nil {
		t.Error("clicking a settings row past the list produced a command")
	}
	_ = next2

	// TargetEngineRow: a valid pane applies a choice; an out-of-range one
	// leaves the window alone.
	m2, _ := testModel(t, 100, 30)
	next3, _ := m2.leftClick(Target{Kind: TargetEngineRow, Pane: 0})
	if asModel(t, next3).knobs.Engine == "" {
		t.Error("clicking the first engine row should have chosen an engine")
	}
	next4, cmd4 := m2.leftClick(Target{Kind: TargetEngineRow, Pane: 999})
	if cmd4 != nil || asModel(t, next4).knobs.Engine != "" {
		t.Error("clicking an engine row past the list should do nothing")
	}

	// TargetCommand: not found, found and already selected (runs it), and
	// found but not yet selected (only selects it).
	m3, _ := testModel(t, 100, 30)
	m3.opts.Commands = []Command{{Name: "new"}, {Name: "repos"}}
	m3 = m3.openPalette()
	if _, cmd := m3.leftClick(Target{Kind: TargetCommand, Key: "nope"}); cmd != nil {
		t.Error("clicking a command not in the filtered list produced a command")
	}
	next5, _ := m3.leftClick(Target{Kind: TargetCommand, Key: "new"})
	if asModel(t, next5).screen != screenCompose {
		t.Error("clicking the already-selected command should have run it")
	}
	next6, _ := m3.leftClick(Target{Kind: TargetCommand, Key: "repos"})
	after6 := asModel(t, next6)
	if after6.screen == screenCompose || !after6.palette.open {
		t.Error("clicking a different command should only select it, not run it")
	}
}

func TestLeftClickMenuFlowRepoAndQueueBranches(t *testing.T) {
	// TargetMenuEntry: not found, found but not selected, found and chosen.
	m, _ := testModel(t, 100, 30)
	m.opts.Commands = []Command{{Name: "new"}, {Name: "repos"}}
	m = m.openMenu("")
	next, _ := m.leftClick(Target{Kind: TargetMenuEntry, Key: "no-such-entry"})
	if !asModel(t, next).menu.open {
		t.Error("clicking a menu entry that does not exist should leave the menu open")
	}
	next2, _ := m.leftClick(Target{Kind: TargetMenuEntry, Key: "repos"})
	after2 := asModel(t, next2)
	if !after2.menu.open || after2.menu.sel != 1 {
		t.Errorf("clicking an unselected menu entry = open=%v sel=%v, want it only selected", after2.menu.open, after2.menu.sel)
	}
	next3, _ := m.leftClick(Target{Kind: TargetMenuEntry, Key: "new"})
	if asModel(t, next3).menu.open {
		t.Error("clicking the already-selected menu entry should have chosen it")
	}

	// TargetFlowItem passes straight through to the flow form's own click
	// handling; it should not panic even with nothing behind it.
	_, _ = m.leftClick(Target{Kind: TargetFlowItem})

	// TargetRepo: filters to a repo, clears the filter on a second click,
	// and does nothing for a name that matches no repo at all.
	m2, _ := testModel(t, 100, 30)
	m2.screen = screenRepos
	next4, _ := m2.leftClick(Target{Kind: TargetRepo, ID: "payments"})
	after4 := asModel(t, next4)
	if after4.repoFilter != "payments" || after4.screen != screenList {
		t.Errorf("clicking a repo = filter=%q screen=%v, want it filtered and the board back", after4.repoFilter, after4.screen)
	}
	after4.screen = screenRepos
	next5, _ := after4.leftClick(Target{Kind: TargetRepo, ID: "payments"})
	if asModel(t, next5).repoFilter != "" {
		t.Error("clicking the same repo again should clear the filter")
	}
	m3, _ := testModel(t, 100, 30)
	next6, cmd6 := m3.leftClick(Target{Kind: TargetRepo, ID: "no-such-repo"})
	if cmd6 != nil || asModel(t, next6).repoFilter != "" {
		t.Error("clicking a repo name that matches nothing should do nothing")
	}

	// TargetHeaderQueue: filters to a band, then clears on the same click
	// twice over.
	m4, _ := testModel(t, 100, 30)
	next7, _ := m4.leftClick(Target{Kind: TargetHeaderQueue, Band: view.Done})
	after7 := asModel(t, next7)
	if after7.queueFilter == nil || *after7.queueFilter != view.Done {
		t.Fatal("clicking the Done chip should have set the queue filter")
	}
	next8, _ := after7.leftClick(Target{Kind: TargetHeaderQueue, Band: view.Done})
	if asModel(t, next8).queueFilter != nil {
		t.Error("clicking the same chip again should clear the queue filter")
	}

	// TargetHeaderField and TargetStatusField's remaining fields.
	m5, _ := testModel(t, 100, 30)
	next9, _ := m5.leftClick(Target{Kind: TargetHeaderField, Field: "repos"})
	if asModel(t, next9).screen != screenRepos {
		t.Error("clicking the repos header field should open the repo list")
	}
	next10, _ := m5.leftClick(Target{Kind: TargetHeaderField, Field: "engine"})
	if asModel(t, next10).screen != screenEngines {
		t.Error("clicking the engine header field should open the engine knobs")
	}
	beforeAuto := m5.autopilotOn()
	next11, _ := m5.leftClick(Target{Kind: TargetStatusField, Field: "autopilot"})
	if asModel(t, next11).autopilotOn() == beforeAuto {
		t.Error("clicking the status bar's autopilot field should flip it")
	}
	next12, _ := m5.leftClick(Target{Kind: TargetStatusField, Field: "engine"})
	if asModel(t, next12).screen != screenEngines {
		t.Error("clicking the status bar's engine field should open the engine knobs")
	}

	// TargetBarHint with nothing named, and with a real key.
	if _, cmd := m5.leftClick(Target{Kind: TargetBarHint, Key: ""}); cmd != nil {
		t.Error("clicking a bar hint with no key produced a command")
	}
	next13, _ := m5.leftClick(Target{Kind: TargetBarHint, Key: "M"})
	if asModel(t, next13).screen != screenEngines {
		t.Error("clicking the engine-knobs bar hint should open the engine knobs")
	}
}
