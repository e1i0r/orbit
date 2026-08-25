package ui

import (
	"testing"

	"charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

func TestTargetDialogHitDetection(t *testing.T) {
	m, _ := testModel(t, 120, 30)

	// 1. screenDetail hit
	m.screen = screenDetail
	m.tab = tabLog
	_ = m.hit(10, 4) // tab strip
	_ = m.hit(10, 8) // pane body

	// 2. screenStart hit
	m.screen = screenStart
	_ = m.hit(10, 4)
	_ = m.hit(10, 6)
	_ = m.hit(10, 10)

	// 3. screenSettings hit
	m.screen = screenSettings
	_ = m.hit(10, 4)
	_ = m.hit(25, 4)

	// 4. screenRepos hit
	m.screen = screenRepos
	_ = m.hit(10, 4)
	_ = m.hit(10, 8)
}

func TestSettingsScreenInteractions(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.screen = screenSettings

	sendKey := func(k rune, code rune) {
		var msg tea.Msg
		if code != 0 {
			msg = tea.KeyPressMsg{Code: code}
		} else {
			msg = tea.KeyPressMsg{Code: k, Text: string(k)}
		}
		updated, _ := m.Update(msg)
		m = asModel(t, updated)
	}

	// Up / Down navigation
	sendKey(0, tea.KeyDown)
	sendKey(0, tea.KeyUp)

	// Enter to cycle setting
	sendKey(0, tea.KeyEnter)

	// Left / Right to cycle options
	sendKey(0, tea.KeyRight)
	sendKey(0, tea.KeyLeft)

	// Number keys for unread cap
	sendKey('5', 0)
	sendKey('0', 0)

	// Escape to close
	sendKey(0, tea.KeyEsc)
	if m.screen != screenList {
		t.Errorf("screen after Esc = %v, want screenList", m.screen)
	}
}

func TestStartScreenInteractions(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.screen = screenStart

	sendKey := func(k rune, code rune) {
		var msg tea.Msg
		if code != 0 {
			msg = tea.KeyPressMsg{Code: code}
		} else {
			msg = tea.KeyPressMsg{Code: k, Text: string(k)}
		}
		updated, _ := m.Update(msg)
		m = asModel(t, updated)
	}

	// Tab to switch fields
	sendKey(0, tea.KeyTab)

	// Left / Right to cycle flow / dial
	sendKey(0, tea.KeyRight)
	sendKey(0, tea.KeyLeft)

	// 'a' to toggle autopilot
	sendKey('a', 0)

	// Escape to close
	sendKey(0, tea.KeyEsc)
	if m.screen != screenList {
		t.Errorf("screen after Esc = %v, want screenList", m.screen)
	}
}

func TestHitTopLevelRefusalsAndDispatch(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. A window with no geometry at all, or one narrower than the frame
	// can be fit into, points at nothing.
	zero := m
	zero.width, zero.height = 0, 0
	if got := zero.hit(5, 5); got.Kind != TargetNone {
		t.Errorf("hit on a zero-sized window = %+v, want TargetNone", got)
	}
	narrow := m
	narrow.tooNarrow = true
	if got := narrow.hit(5, 5); got.Kind != TargetNone {
		t.Errorf("hit while too narrow = %+v, want TargetNone", got)
	}

	// 2. The bar row while the palette or the menu owns the keyboard names
	// nothing: both are typed into, not clicked on their own row.
	m.palette.open = true
	if got := m.hit(5, m.frame.Bar.Y); got.Kind != TargetNone {
		t.Errorf("hit on the bar with the palette open = %+v, want TargetNone", got)
	}
	m.palette.open = false
	m.menu.open = true
	if got := m.hit(5, m.frame.Bar.Y); got.Kind != TargetNone {
		t.Errorf("hit on the bar with the menu open = %+v, want TargetNone", got)
	}
	m.menu.open = false

	// 3. Status and Band both answer through hitStatus.
	if got := m.hit(5, m.frame.Status.Y); got.Kind != TargetStatusField {
		t.Errorf("hit on the status row = %+v, want TargetStatusField", got)
	}
	if got := m.hit(5, m.frame.Band.Y); got.Kind != TargetStatusField {
		t.Errorf("hit on the band row = %+v, want TargetStatusField", got)
	}

	// 4. In the body: a watched command owns the screen and answers
	// nothing; every other screen dispatches somewhere, which the
	// screen-specific tests below check in detail — here only that none of
	// them panics.
	m.watchUp = true
	if got := m.hit(5, m.frame.Body.Y); got.Kind != TargetNone {
		t.Errorf("hit in the body while a watch is up = %+v, want TargetNone", got)
	}
	m.watchUp = false
	for _, scr := range []screen{screenDetail, screenStart, screenSettings, screenEngines, screenFlows, screenRepos, screenCompose, screenList} {
		m.screen = scr
		_ = m.hit(5, m.frame.Body.Y)
	}
}

func TestHitRowEveryOutcome(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Outside the body entirely.
	if got := m.hitRow(5, 0); got.Kind != TargetNone {
		t.Errorf("hitRow outside the body = %+v, want TargetNone", got)
	}

	// 2. The "…and N more" line, and anything below the rows actually
	// drawn — reachable with a body too short for the fixture's list.
	short := m
	short.frame.Body.H = 2
	if got := short.hitRow(5, short.frame.Body.Y+1); got.Kind != TargetNone {
		t.Errorf("hitRow past the shown rows = %+v, want TargetNone", got)
	}

	// 3. A blank separator row.
	rows := m.rows()
	var blankIdx, headIdx, taskIdx = -1, -1, -1
	for i, r := range rows {
		switch {
		case r.blank && blankIdx < 0:
			blankIdx = i
		case r.head && headIdx < 0:
			headIdx = i
		case !r.head && !r.blank && taskIdx < 0:
			taskIdx = i
		}
	}
	if blankIdx < 0 || headIdx < 0 || taskIdx < 0 {
		t.Fatal("the fixture body is missing a blank, a header or a task row to test against")
	}
	if got := m.hitRow(5, m.frame.Body.Y+blankIdx); got.Kind != TargetNone {
		t.Errorf("hitRow on a blank row = %+v, want TargetNone", got)
	}

	// 4. A band header.
	if got := m.hitRow(5, m.frame.Body.Y+headIdx); got.Kind != TargetBandHeader {
		t.Errorf("hitRow on a band header = %+v, want TargetBandHeader", got)
	}

	// 5. A task row, with the column carried from x.
	got := m.hitRow(gutter+2, m.frame.Body.Y+taskIdx)
	if got.Kind != TargetTask || got.ID != rows[taskIdx].task.ID {
		t.Errorf("hitRow on a task row = %+v, want TargetTask for %q", got, rows[taskIdx].task.ID)
	}
}

func TestHitHeaderEveryField(t *testing.T) {
	m, _ := testModel(t, 150, 30)
	y := m.frame.Header.Y

	if got := m.hitHeader(5, y+50); got.Kind != TargetNone {
		t.Errorf("hitHeader off its own row = %+v, want TargetNone", got)
	}

	tests := []struct {
		x     int
		want  TargetKind
		band  view.Band
		field string
	}{
		{5, TargetHeaderField, 0, "orbit"},
		{15, TargetHeaderQueue, view.ToDo, ""},
		{30, TargetHeaderQueue, view.Running, ""},
		{50, TargetHeaderQueue, view.NeedsYou, ""},
		{70, TargetHeaderQueue, view.Done, ""},
		{85, TargetNone, 0, ""}, // the gap between the queue badges and the right-hand fields
		{100, TargetHeaderField, 0, "repos"},
		{125, TargetHeaderField, 0, "engine"},
		{145, TargetHeaderField, 0, "lang"},
	}
	for _, tt := range tests {
		got := m.hitHeader(tt.x, y)
		if got.Kind != tt.want {
			t.Errorf("hitHeader(%d) = %+v, want kind %v", tt.x, got, tt.want)
		}
		if tt.want == TargetHeaderQueue && got.Band != tt.band {
			t.Errorf("hitHeader(%d) band = %v, want %v", tt.x, got.Band, tt.band)
		}
		if tt.want == TargetHeaderField && got.Field != tt.field {
			t.Errorf("hitHeader(%d) field = %q, want %q", tt.x, got.Field, tt.field)
		}
	}
}

func TestHitBarBranches(t *testing.T) {
	m, _ := testModel(t, 150, 30)
	y := m.frame.Bar.Y

	if got := m.hitBar(10, y+5); got.Kind != TargetNone {
		t.Errorf("hitBar off its own row = %+v, want TargetNone", got)
	}

	m.screen = screenList
	if got := m.hitBar(m.width-5, y); got.Kind != TargetBarHint || got.Key != "c" {
		t.Errorf("hitBar on the cli chip = %+v, want the c hint", got)
	}
	if got := m.hitBar(m.width-45, y); got.Kind != TargetStatusField || got.Field != "autopilot" {
		t.Errorf("hitBar on the autopilot chip (screenList) = %+v, want the autopilot field", got)
	}

	m.screen = screenDetail
	if got := m.hitBar(m.width-10, y); got.Kind != TargetStatusField || got.Field != "autopilot" {
		t.Errorf("hitBar on the autopilot chip (not screenList) = %+v, want the autopilot field", got)
	}

	m.screen = screenList
	_, hints := m.barLayout(m.frame.Bar.W)
	if len(hints) == 0 {
		t.Fatal("the fixture bar has no hints to test against")
	}
	first := hints[0]
	if got := m.hitBar(first.x, y); got.Kind != TargetBarHint || got.Key != first.key {
		t.Errorf("hitBar on the first hint = %+v, want key %q", got, first.key)
	}
	last := hints[len(hints)-1]
	if gapX := last.x + last.w + 1; gapX < m.width-60 {
		if got := m.hitBar(gapX, y); got.Kind != TargetNone {
			t.Errorf("hitBar in the gap past the last hint = %+v, want TargetNone", got)
		}
	}
}
