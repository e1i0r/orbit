package ui

import (
	"errors"
	"testing"

	"charm.land/bubbletea/v2"
	"github.com/e1i0r/orbit/internal/view"
)

func TestComposeScreenFullLifecycle(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.board.Tasks = []view.Task{
		{ID: "PAY-1", Repo: "payments"},
	}

	// 1. Open compose
	m = m.openCompose()
	if m.screen != screenCompose {
		t.Fatalf("screen = %v, want screenCompose", m.screen)
	}

	sendKey := func(k rune, code rune, text string) {
		var msg tea.Msg
		switch {
		case text != "":
			msg = tea.KeyPressMsg{Code: code, Text: text}
		case code != 0:
			msg = tea.KeyPressMsg{Code: code}
		default:
			msg = tea.KeyPressMsg{Code: k, Text: string(k)}
		}
		updated, _ := m.Update(msg)
		m = asModel(t, updated)
	}

	// 2. Type repo prefix and autocomplete with composeTab
	sendKey('p', 0, "")
	sendKey('a', 0, "")
	m = m.composeTab(1)
	if m.compose.repo != "payments" {
		t.Errorf("expected autocomplete to 'payments', got %q", m.compose.repo)
	}

	// 3. Move to ID field with composeNext
	updatedModel, _ := m.composeNext()
	m = asModel(t, updatedModel)
	if m.compose.field != composeID {
		t.Errorf("field after composeNext = %d, want composeID", m.compose.field)
	}

	// 4. Type ID and backspace
	sendKey('T', 0, "")
	sendKey('A', 0, "")
	sendKey('S', 0, "")
	sendKey('K', 0, "")
	sendKey('-', 0, "")
	sendKey('9', 0, "")
	sendKey('X', 0, "")
	sendKey(0, tea.KeyBackspace, "")
	if m.compose.id != "TASK-9" {
		t.Errorf("compose.id = %q, want TASK-9", m.compose.id)
	}

	// 5. Move to text field with composeNext
	updatedModel, _ = m.composeNext()
	m = asModel(t, updatedModel)
	if m.compose.field != composeText {
		t.Errorf("field after composeNext = %d, want composeText", m.compose.field)
	}

	// 6. Type task description
	for _, ch := range "Fix payment webhook verification" {
		sendKey(ch, 0, "")
	}

	// 7. Render compose view
	v := m.View()
	if len(v.Content) == 0 {
		t.Error("expected non-empty compose view content")
	}

	// 8. Submit with composeSubmit
	updatedModel, cmd := m.composeSubmit()
	m = asModel(t, updatedModel)
	if cmd == nil {
		t.Error("expected non-nil cmd from composeSubmit")
	}

	// 9. Validation errors on empty fields
	mEmpty := m.openCompose()
	mEmpty.compose.repo = ""
	_, _ = mEmpty.composeSubmit()
	mEmpty.compose.repo = "repo"
	mEmpty.compose.id = ""
	_, _ = mEmpty.composeSubmit()
	mEmpty.compose.id = "TASK-1"
	mEmpty.compose.text = ""
	_, _ = mEmpty.composeSubmit()

	// 10. Escape to abandon compose
	sendKey(0, tea.KeyEsc, "esc")
	if m.screen != screenList {
		t.Errorf("screen after Esc = %v, want screenList", m.screen)
	}
}

// TestComposeKeyTabAndOpen drives composeTab and composeNext through
// composeKey itself — the full lifecycle test above calls both directly —
// so the key.Matches branches for NextTab, PrevTab and Open are exercised
// through Update rather than bypassed.
func TestComposeKeyTabAndOpen(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openCompose()
	// A prefix nothing on the fixture board answers to, so Tab moves the
	// caret instead of autocompleting the repository field.
	m.compose.repo = "zzz-no-match"

	next, _ := m.Update(press("tab"))
	m = asModel(t, next)
	if m.compose.field != composeID {
		t.Fatalf("field after Tab = %d, want composeID", m.compose.field)
	}
	next, _ = m.Update(press("shift+tab"))
	m = asModel(t, next)
	if m.compose.field != composeRepo {
		t.Fatalf("field after Shift-Tab = %d, want composeRepo", m.compose.field)
	}

	// Open on the repo field moves to the id field, same as Tab.
	next, cmd := m.Update(press("enter"))
	m = asModel(t, next)
	if cmd != nil || m.compose.field != composeID {
		t.Fatalf("Open on the repo field = field %d cmd %v, want composeID and no cmd", m.compose.field, cmd)
	}

	// Open on the text field submits.
	m.compose.field, m.compose.repo, m.compose.id, m.compose.text = composeText, "payments", "TASK-1", "do it"
	next, cmd = m.Update(press("enter"))
	m = asModel(t, next)
	if cmd == nil {
		t.Errorf("expected Open on the last field to submit and return a cmd")
	}
}

// TestComposeTabBoundariesAndCompletion is composeTab's own clamps and the
// completion path composeComplete takes when nothing on the board matches
// what was typed.
func TestComposeTabBoundariesAndCompletion(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{{ID: "PAY-1", Repo: "payments"}}
	m = m.openCompose()

	// Forward from the last field clamps rather than overflowing.
	m.compose.field = composeText
	m2 := m.composeTab(1)
	if m2.compose.field != composeText {
		t.Errorf("composeTab(1) from the last field = %d, want it to stay put", m2.compose.field)
	}

	// Backward from the first field clamps at zero.
	m.compose.field = composeRepo
	m3 := m.composeTab(-1)
	if m3.compose.field != composeRepo {
		t.Errorf("composeTab(-1) from the first field = %d, want it to stay put", m3.compose.field)
	}

	// A repo prefix nothing on the board answers to just moves the caret.
	m.compose.repo = "nothing-matches-this"
	m4 := m.composeTab(1)
	if m4.compose.repo != "nothing-matches-this" {
		t.Errorf("expected an unmatched prefix left alone, got %q", m4.compose.repo)
	}
	if m4.compose.field != composeID {
		t.Errorf("expected the caret to move on when nothing completes")
	}
}

func TestComposeSubmitValidID(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{{ID: "PAY-1", Repo: "payments", RepoPath: "/repo/payments"}}
	m.compose.repo, m.compose.id, m.compose.text = "payments", "TASK-9", "do the thing"

	m.opts.ValidID = func(id string) error { return errors.New("id taken") }
	m2, cmd := m.composeSubmit()
	if cmd != nil {
		t.Fatalf("expected a rejected id to produce no cmd")
	}
	wantBand(t, asModel(t, m2), "id taken")

	m.opts.ValidID = func(id string) error { return nil }
	m3, cmd := m.composeSubmit()
	if cmd == nil {
		t.Fatalf("expected an accepted id to submit")
	}
	if asModel(t, m3).screen != screenList {
		t.Errorf("expected a submitted compose to return to the list")
	}
}

func TestSelectPending(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// No pending id: a no-op.
	m2 := m.selectPending()
	if m2.pendingID != "" {
		t.Fatalf("expected pendingID to stay empty")
	}

	// A pending id present on the board is found and selected. ACME-2701
	// is in NeedsYou, which is expanded by default — a pending id in a
	// collapsed band would never be on screen to find.
	m.pendingID = "ACME-2701"
	m2 = m.selectPending()
	if m2.pendingID != "" || m2.pendTries != 0 {
		t.Errorf("expected a found id to clear pendingID and pendTries")
	}

	// A pending id absent from the board counts tries, then gives up.
	m.pendingID = "NOT-ON-BOARD"
	m3 := m
	for range 4 {
		m3 = m3.selectPending()
	}
	if m3.pendingID != "" {
		t.Errorf("expected selectPending to give up after a few misses, still pending %q", m3.pendingID)
	}
}

func TestComposeUpDownAndMouseClicks(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openCompose()

	// 1. Arrow down moves between fields
	m = asModel(t, mustUpdate(m, press("down")))
	if m.compose.field != composeID {
		t.Fatalf("field after down arrow = %d, want composeID", m.compose.field)
	}
	m = asModel(t, mustUpdate(m, press("down")))
	if m.compose.field != composeText {
		t.Fatalf("field after second down arrow = %d, want composeText", m.compose.field)
	}

	// 2. Arrow up moves back up
	m = asModel(t, mustUpdate(m, press("up")))
	if m.compose.field != composeID {
		t.Fatalf("field after up arrow = %d, want composeID", m.compose.field)
	}
	m = asModel(t, mustUpdate(m, press("up")))
	if m.compose.field != composeRepo {
		t.Fatalf("field after second up arrow = %d, want composeRepo", m.compose.field)
	}

	// 3. Mouse clicks directly focus fields
	// Line 1 in body is repo, line 2 is id, line 3 is task
	yRepo := m.frame.Body.Y + 1
	yID := m.frame.Body.Y + 2
	yTask := m.frame.Body.Y + 3

	clickField := func(y int) {
		res, _ := m.mouse(tea.MouseClickMsg{X: 10, Y: y, Button: tea.MouseLeft})
		m = asModel(t, res)
		res, _ = m.mouse(tea.MouseReleaseMsg{X: 10, Y: y, Button: tea.MouseLeft})
		m = asModel(t, res)
	}

	clickField(yID)
	if m.compose.field != composeID {
		t.Errorf("field after click on ID = %d, want composeID", m.compose.field)
	}

	clickField(yTask)
	if m.compose.field != composeText {
		t.Errorf("field after click on Task = %d, want composeText", m.compose.field)
	}

	clickField(yRepo)
	if m.compose.field != composeRepo {
		t.Errorf("field after click on Repo = %d, want composeRepo", m.compose.field)
	}
}

func mustUpdate(m Model, msg tea.Msg) tea.Model {
	next, _ := m.Update(msg)
	return next
}
