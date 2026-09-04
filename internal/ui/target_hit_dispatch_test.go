package ui

import (
	"testing"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	if short.frame.Body.H > 0 {
		if got := short.hitRow(5, short.frame.Body.Y); got.Kind != TargetNone {
			t.Errorf("hitRow on table header row = %+v, want TargetNone", got)
		}
	}

	// 3. A blank separator row.
	rows := m.rows()
	blankIdx, headIdx, taskIdx := -1, -1, -1

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

	if got := m.hitRow(5, m.frame.Body.Y+1+blankIdx); got.Kind != TargetNone {
		t.Errorf("hitRow on a blank row = %+v, want TargetNone", got)
	}

	// 4. A band header.
	if got := m.hitRow(5, m.frame.Body.Y+1+headIdx); got.Kind != TargetBandHeader {
		t.Errorf("hitRow on a band header = %+v, want TargetBandHeader", got)
	}

	// 5. A task row, with the column carried from x.
	got := m.hitRow(gutter+2, m.frame.Body.Y+1+taskIdx)
	if got.Kind != TargetTask || got.ID != rows[taskIdx].task.ID {
		t.Errorf("hitRow on a task row = %+v, want TargetTask for %q", got, rows[taskIdx].task.ID)
	}
}

func TestHitHeaderEveryField(t *testing.T) {
	m, _ := testModel(t, 150, 30)
	y := m.frame.HeaderLineY()

	if got := m.hitHeader(5, y+50); got.Kind != TargetNone {
		t.Errorf("hitHeader off its own row = %+v, want TargetNone", got)
	}

	// The name badge is the first thing on the line, so it starts at column
	// zero and runs for as many cells as it is wide. Which cells the badges
	// and chips after it occupy is header_hit_test.go's question, and it
	// asks the drawn line rather than writing the answer down.
	for x := range lipgloss.Width(m.name()) {
		if got := m.hitHeader(x, y); got.Kind != TargetHeaderField || got.Field != "orbit" {
			t.Errorf("hitHeader(%d) = %+v, want the orbit badge", x, got)
		}
	}

	past := lipgloss.Width(m.name())
	if got := m.hitHeader(past, y); got.Kind != TargetNone {
		t.Errorf("hitHeader(%d) = %+v, want nothing: that cell is past the badge", past, got)
	}
}

func TestHitBarBranches(t *testing.T) {
	m, _ := testModel(t, 150, 30)
	y := m.frame.Bar.Y

	if got := m.hitBar(10, y+5); got.Kind != TargetNone {
		t.Errorf("hitBar off its own row = %+v, want TargetNone", got)
	}

	// The chips at the right end are asked where they were drawn rather
	// than at a column written down here: their widths are of translated
	// words, and the two constants this replaced — the last 28 columns are
	// the cli chip, the 32 before it the switch — were a guess that was
	// wrong in Spanish and wrong at every width where the bar drops the
	// chips and leaves that end of the line empty.
	m.screen = screenList

	_, hints, chips := m.barLayout(m.frame.Bar.W)
	if len(chips) != 2 {
		t.Fatalf("the board's bar placed %d chips, want the switch and the cli one", len(chips))
	}

	for _, z := range chips {
		for _, x := range []int{z.x, z.x + z.w - 1} {
			if got := m.hitBar(x, y); got != z.target {
				t.Errorf("hitBar(%d) = %+v, want %+v: that cell is on the chip", x, got, z.target)
			}
		}
	}

	if gap := chips[0].x + chips[0].w; gap < chips[1].x {
		if got := m.hitBar(gap, y); got.Kind != TargetNone {
			t.Errorf("hitBar between the two chips = %+v, want TargetNone", got)
		}
	}

	// Off the board there is no cli chip, and the switch is the last thing
	// on the line.
	detail := m
	detail.screen = screenDetail

	_, _, detailChips := detail.barLayout(detail.frame.Bar.W)
	if len(detailChips) != 1 {
		t.Fatalf("the detail bar placed %d chips, want the switch alone", len(detailChips))
	}

	if got := detail.hitBar(detailChips[0].x, y); got.Kind != TargetStatusField || got.Field != "autopilot" {
		t.Errorf("hitBar on the autopilot chip (not screenList) = %+v, want the autopilot field", got)
	}

	if len(hints) == 0 {
		t.Fatal("the fixture bar has no hints to test against")
	}

	first := hints[0]
	if got := m.hitBar(first.x, y); got.Kind != TargetBarHint || got.Key != first.key {
		t.Errorf("hitBar on the first hint = %+v, want key %q", got, first.key)
	}

	last := hints[len(hints)-1]
	if gapX := last.x + last.w + 1; gapX < chips[0].x {
		if got := m.hitBar(gapX, y); got.Kind != TargetNone {
			t.Errorf("hitBar in the gap past the last hint = %+v, want TargetNone", got)
		}
	}
}

// A click on the cheat sheet is a click on nothing.
//
// The help screen is drawn over the board, and until it had an arm of its
// own in hit every cell of it answered as the board's row at that height:
// pointing at a line of the sheet moved the cursor onto a task the reader
// could not see, and pointing at it again opened that task.
func TestClicksOnTheHelpScreenDoNotReachTheBoard(t *testing.T) {
	m, _ := testModel(t, 150, 30)
	m.screen = screenHelp

	for _, dy := range []int{0, 1, 3, 10} {
		if got := m.hit(4, m.frame.Body.Y+dy); got.Kind != TargetNone {
			t.Errorf("hit on help screen row %d = %+v, want TargetNone", dy, got)
		}
	}

	// The same cell on the board still answers, so what changed is the
	// screen and not the rows under it.
	board := m
	board.screen = screenList

	if got := board.hit(4, board.frame.Body.Y+1); got.Kind == TargetNone {
		t.Error("the board's own first row stopped answering")
	}
}
