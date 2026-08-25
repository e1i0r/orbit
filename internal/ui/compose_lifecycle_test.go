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

	// Move to ID field with composeNext
	updatedModel, _ := m.composeNext(false)
	m = asModel(t, updatedModel)
	if m.compose.field != composeID {
		t.Errorf("field after composeNext = %d, want composeID", m.compose.field)
	}

	// Type ID and backspace
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

	// Move to text field with composeNext
	updatedModel, _ = m.composeNext(false)
	m = asModel(t, updatedModel)
	if m.compose.field != composeText {
		t.Errorf("field after composeNext = %d, want composeText", m.compose.field)
	}

	// Type task description
	for _, ch := range "Fix payment webhook verification" {
		sendKey(ch, 0, "")
	}

	v := m.View()
	if len(v.Content) == 0 {
		t.Error("expected non-empty compose view content")
	}

	// Submit with composeSubmit
	updatedModel, cmd := m.composeSubmit(false)
	m = asModel(t, updatedModel)
	if cmd == nil {
		t.Error("expected non-nil cmd from composeSubmit")
	}

	// Validation errors on empty fields
	mEmpty := m.openCompose()
	mEmpty.compose.repo = ""
	mEmpty.compose.repos = nil
	_, _ = mEmpty.composeSubmit(false)
	mEmpty.compose.repo = "repo"
	mEmpty.compose.id = ""
	_, _ = mEmpty.composeSubmit(false)
	mEmpty.compose.id = "TASK-1"
	mEmpty.compose.text = ""
	_, _ = mEmpty.composeSubmit(false)
}

func TestComposeKeyTabAndOpen(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{
		{ID: "ACME-1", Repo: "app"},
	}
	m = m.openCompose()

	// Switching between tabs: [1] Manual, [2] URL
	next, _ := m.Update(press("2"))
	m2 := asModel(t, next)
	if m2.compose.tab != composeTabURL {
		t.Errorf("compose.tab = %d, want composeTabURL (1)", m2.compose.tab)
	}

	next, _ = m2.Update(press("1"))
	m3 := asModel(t, next)
	if m3.compose.tab != composeTabManual {
		t.Errorf("compose.tab = %d, want composeTabManual (0)", m3.compose.tab)
	}

	// Esc abandons
	next, _ = m.Update(press("esc"))
	if asModel(t, next).screen != screenList {
		t.Errorf("screen after esc = %v, want screenList", asModel(t, next).screen)
	}
}

func TestComposeURLAutoParsing(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openCompose()

	// Switch to URL tab
	m.compose.tab = composeTabURL
	m.compose.field = composeURL

	// Type Linear URL
	for _, ch := range "https://linear.app/org/issue/ENG-99/add-tax-rate" {
		updated, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = asModel(t, updated)
	}

	if m.compose.parsedIssue == nil {
		t.Fatal("expected parsedIssue to not be nil")
	}
	if m.compose.id != "ENG-99" {
		t.Errorf("compose.id = %q, want ENG-99", m.compose.id)
	}
	if m.compose.text != "Add tax rate" {
		t.Errorf("compose.text = %q, want 'Add tax rate'", m.compose.text)
	}
}

func TestComposeRepoCycles(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{
		{ID: "APP-1", Repo: "app", RepoPath: "/repos/app"},
		{ID: "PAY-1", Repo: "payments", RepoPath: "/repos/payments"},
		{ID: "AUTH-1", Repo: "auth", RepoPath: "/repos/auth"},
	}
	m = m.openCompose()

	// Left and right arrow keys cycle repo
	m = asModel(t, mustUpdate(m, press("right")))
	if m.compose.repo != "payments" {
		t.Errorf("repo after right arrow = %q, want payments", m.compose.repo)
	}

	m = asModel(t, mustUpdate(m, press("right")))
	if m.compose.repo != "auth" {
		t.Errorf("repo after second right arrow = %q, want auth", m.compose.repo)
	}

	m = asModel(t, mustUpdate(m, press("left")))
	if m.compose.repo != "payments" {
		t.Errorf("repo after left arrow = %q, want payments", m.compose.repo)
	}
}

func TestComposeSubmitValidID(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{{ID: "PAY-1", Repo: "payments", RepoPath: "/repo/payments"}}
	m.compose.repo, m.compose.id, m.compose.text = "payments", "TASK-9", "do the thing"

	m.opts.ValidID = func(id string) error { return errors.New("id taken") }
	m2, cmd := m.composeSubmit(false)
	if cmd != nil {
		t.Fatalf("expected a rejected id to produce no cmd")
	}
	wantBand(t, asModel(t, m2), "id taken")

	m.opts.ValidID = func(id string) error { return nil }
	m3, cmd := m.composeSubmit(false)
	if cmd == nil {
		t.Fatalf("expected an accepted id to submit")
	}
	if asModel(t, m3).screen != screenList {
		t.Errorf("expected a submitted compose to return to the list")
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
	yID := m.frame.Body.Y + 3
	yTask := m.frame.Body.Y + 4
	yRepo := m.frame.Body.Y + 2

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
