package ui

// mouseroute_more_coverage_test.go is the routing table in mouseroute.go: a
// clicked hint reaching the same map a keystroke does, a start-dialog switch
// flipped by the pointer, a right click opening the menu, and the header's
// band chips jumping the cursor.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

func TestSendKeyRoutesByScreen(t *testing.T) {
	// 1. Filtering swallows the keystroke whole.
	m, _ := testModel(t, 100, 30)
	m.filtering = true

	next, _ := m.sendKey(keystroke("j"))
	if asModel(t, next).filtering != true {
		t.Error("sendKey while filtering should leave the window filtering")
	}

	// 2. The start dialog gets its own map.
	m2, _ := testModel(t, 100, 30)
	m2, _ = dialog(t, m2, "ACME-2662")

	next2, _ := m2.sendKey(keystroke(m2.keys.EngineKnobs.Help().Key))
	if asModel(t, next2).screen != screenEngines {
		t.Errorf("sendKey(%q) from the start dialog = %v, want screenEngines", m2.keys.EngineKnobs.Help().Key, asModel(t, next2).screen)
	}

	// 3. The task view gets its own map too.
	m3 := openOn(t, "ACME-2705")

	next3, _ := m3.sendKey(keystroke(m3.keys.Back.Help().Key))
	if asModel(t, next3).screen != screenList {
		t.Errorf("sendKey(back) from the task view = %v, want screenList", asModel(t, next3).screen)
	}

	// 4. Anything else reaches the board's own map.
	m4, _ := testModel(t, 100, 30)
	m4 = onRow(t, m4, "ACME-2705")

	next4, cmd := m4.sendKey(keystroke("p"))
	if cmd == nil {
		t.Error("sendKey(p) on the board answered with no command")
	}

	_ = next4
}

func TestFlipTheStartDialogSwitches(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m, _ = dialog(t, m, "ACME-2662")

	// 1. The flow switch cycles the dialog's flow.
	before := m.start.chosen().name

	next, _ := m.flip(fieldFlow)
	if asModel(t, next).start.chosen().name == before {
		t.Error("flip(fieldFlow) left the chosen flow unchanged")
	}

	// 2. Autopilot on, clicking the "on" half does nothing further...
	m.opts.Settings.(*settings).autopilot = false //nolint:errcheck

	nextOn, _ := m.flip(fieldAutopilotOn)
	if !asModel(t, nextOn).autopilotOn() {
		t.Error("flip(fieldAutopilotOn) with autopilot off should turn it on")
	}

	// 3. ...and clicking the "off" half while it is already on turns it off.
	m.opts.Settings.(*settings).autopilot = true //nolint:errcheck

	nextOff, _ := m.flip(fieldAutopilotOff)
	if asModel(t, nextOff).autopilotOn() {
		t.Error("flip(fieldAutopilotOff) with autopilot on should turn it off")
	}

	// 4. A field flip knows nothing about is left alone.
	same, cmd := m.flip("something-else")
	if cmd != nil {
		t.Error("flip on an unknown field produced a command")
	}

	if asModel(t, same).start.chosen().name != m.start.chosen().name {
		t.Error("flip on an unknown field changed the dialog")
	}
}

func TestJumpToBandMovesTheCursorAndExpandsIt(t *testing.T) {
	// 1. A band with rows: the cursor lands on its first row, expanded.
	m, _ := testModel(t, 100, 30)
	m.expanded[view.Done] = false

	next, cmd := m.jumpToBand(view.Done)
	if cmd != nil {
		t.Error("jumpToBand produced a command, want none")
	}

	after := asModel(t, next)
	if !after.expanded[view.Done] {
		t.Error("jumpToBand did not expand the band it jumped to")
	}

	r, ok := after.selected()
	if !ok || r.band != view.Done {
		t.Errorf("jumpToBand landed on band %v, want view.Done", r.band)
	}

	// 2. A band with no rows at all leaves the window as it was.
	empty, _ := testModel(t, 100, 30)
	empty.board.Tasks = nil

	same, cmd2 := empty.jumpToBand(view.Band(99))
	if cmd2 != nil {
		t.Error("jumpToBand on an empty band produced a command")
	}

	_ = same
}

func TestRightClickOnThePaneBodyAndElsewhere(t *testing.T) {
	// 1. The task view open on a subject: right click opens the menu on it.
	m := openOn(t, "ACME-2705")
	next, _ := m.rightClick(Target{Kind: TargetPaneBody})

	after := asModel(t, next)
	if !after.menu.open || after.menu.taskID != "ACME-2705" {
		t.Errorf("rightClick(TargetPaneBody) = open=%v taskID=%q, want the menu open on ACME-2705", after.menu.open, after.menu.taskID)
	}

	// 2. No subject at all: right click on the pane body does nothing.
	m2, _ := testModel(t, 100, 30)

	next2, cmd2 := m2.rightClick(Target{Kind: TargetPaneBody})
	if cmd2 != nil || asModel(t, next2).menu.open {
		t.Error("rightClick(TargetPaneBody) with no subject opened the menu")
	}

	// 3. A target that maps to no row at all is left alone.
	m3, _ := testModel(t, 100, 30)

	next3, cmd3 := m3.rightClick(Target{Kind: TargetTask, ID: "no-such-task"})
	if cmd3 != nil || asModel(t, next3).menu.open {
		t.Error("rightClick on a target with no row opened the menu")
	}

	// 4. A band header: right click moves the cursor there but does not
	// open the menu — only a task row does.
	m4, _ := testModel(t, 100, 30)
	next4, _ := m4.rightClick(Target{Kind: TargetBandHeader, Band: view.Done})

	after4 := asModel(t, next4)
	if after4.menu.open {
		t.Error("rightClick on a band header opened the menu")
	}

	r, ok := after4.selected()
	if !ok || r.band != view.Done || !r.head {
		t.Error("rightClick on a band header did not move the cursor there")
	}
}
