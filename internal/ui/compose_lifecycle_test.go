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
		{ID: "PAY-1", Repo: "payments", RepoPath: "/checkouts/payments"},
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

	// Move through fields to ID
	m.compose.field = composeID

	// Type ID and backspace
	sendKey('T', 0, "")
	sendKey('A', 0, "")
	sendKey('S', 0, "")
	sendKey('K', 0, "")
	sendKey('-', 0, "")
	sendKey('9', 0, "")
	sendKey('X', 0, "")
	sendKey(0, tea.KeyBackspace, "")

	if m.compose.id.String() != "TASK-9" {
		t.Errorf("compose.id = %q, want TASK-9", m.compose.id)
	}

	// Move to text field
	m.compose.field = composeText

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
	mEmpty.compose.repoPath = ""
	_, _ = mEmpty.composeSubmit(false)
	mEmpty.compose.repoPath = "/checkouts/payments"
	mEmpty.compose.id = newInput("")
	_, _ = mEmpty.composeSubmit(false)
	mEmpty.compose.id = newInput("TASK-1")
	mEmpty.compose.text = newInput("")
	_, _ = mEmpty.composeSubmit(false)
}

func TestComposeKeyTabAndOpen(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{
		{ID: "ACME-1", Repo: "app", RepoPath: "/checkouts/app"},
	}
	m = m.openCompose()

	// Switching between tabs: [1] Manual, [2] URL
	m.compose.field = composeFlow

	m = asModel(t, mustUpdate(m, press("2")))
	if m.compose.tab != composeTabURL {
		t.Errorf("tab = %d, want composeTabURL", m.compose.tab)
	}

	m = asModel(t, mustUpdate(m, press("1")))
	if m.compose.tab != composeTabManual {
		t.Errorf("tab = %d, want composeTabManual", m.compose.tab)
	}

	// Key '+' on flow field opens flow builder
	m.compose.field = composeFlow
	res, _ := m.Update(tea.KeyPressMsg{Text: "+"})

	mFlows := asModel(t, res)
	if mFlows.screen != screenFlows {
		t.Errorf("screen after '+' on flow field = %v, want screenFlows", mFlows.screen)
	}
}

func TestComposeURLAutoParsing(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openCompose()

	// Pasting a Linear URL in manual mode switches to URL tab
	m.compose.tab = composeTabManual
	m.compose.field = composeID
	linearURL := "https://linear.app/acme/issue/ENG-456/fix-auth-flow"
	m = m.paste(linearURL)

	if m.compose.tab != composeTabURL {
		t.Errorf("tab after pasting URL = %d, want composeTabURL", m.compose.tab)
	}

	if m.compose.id.String() != "ENG-456" {
		t.Errorf("compose.id = %q, want ENG-456", m.compose.id)
	}
}

func TestComposePaste(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openCompose()
	m.compose.tab = composeTabManual
	m.compose.field = composeText

	m = m.paste("Pasted task requirement line 1\nLine 2")

	if m.compose.text.String() != "Pasted task requirement line 1\nLine 2" {
		t.Errorf("compose.text = %q, want pasted multiline text", m.compose.text)
	}
}

func TestComposePillsCycle(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{
		{ID: "APP-1", Repo: "frontend"},
		{ID: "API-1", Repo: "backend"},
	}
	m = m.openCompose()

	m.compose.field = composeFlow
	oldFlow := m.compose.chosenFlow()

	m = asModel(t, mustUpdate(m, tea.KeyPressMsg{Code: tea.KeyRight}))
	if len(m.compose.flows) > 1 && m.compose.chosenFlow() == oldFlow {
		t.Errorf("expected flow to cycle from %s", oldFlow)
	}
}

func TestComposeSubmitValidID(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openCompose()
	m.compose.repoPath = "/checkouts/repo"
	m.compose.id = newInput("TASK-10")
	m.compose.text = newInput("Write some tests")

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
	if m.compose.field != composeFlow {
		t.Fatalf("field after second up arrow = %d, want composeFlow", m.compose.field)
	}

	// 3. Mouse clicks directly focus fields
	extra := len(m.flowDetail(m.compose.chosenFlow(), m.width))

	yFlow := m.frame.Body.Y + 2
	yID := m.frame.Body.Y + 3 + extra

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

	clickField(yFlow)

	if m.compose.field != composeFlow {
		t.Errorf("field after click on Flow = %d, want composeFlow", m.compose.field)
	}
}

// TestTheFormPicksWhereTheTaskStartsAndDoesNotAsk. The repository was the
// first field on this screen. What replaced it is not a smaller field or a
// better default — it is nothing at all, and this is the whole of what the
// form does instead.
func TestTheFormPicksWhereTheTaskStartsAndDoesNotAsk(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// The task the cursor is on, because a reader writing a task while
	// looking at another one is usually writing about the same code.
	m = onRow(t, m, "ACME-2701")
	if got := m.openCompose().compose.repoPath; got != "/checkouts/app" {
		t.Errorf("the form starts in %q, want the checkout of the task the cursor was on", got)
	}

	// A checkout Orbit has never listed, which can only name itself by
	// where it is: the session a reader opened by hand somewhere else.
	if got := m.startsIn("/elsewhere/ledger"); got != "/elsewhere/ledger" {
		t.Errorf("a checkout named by path resolved to %q, want the path itself", got)
	}

	// Nothing preferred and nothing to prefer: the first one Orbit knows.
	if got := m.startsIn(""); got == "" {
		t.Error("the form found nowhere to start on a board with four repositories")
	}

	// And a board with none is the one case the form still has to say
	// something about, because there is nothing for it to pick.
	empty, _ := testModel(t, 100, 30)
	empty.board = fixtureBoard(nil, 0)

	if got := empty.startsIn("payments"); got != "" {
		t.Errorf("a board with no repositories offered %q to start in", got)
	}

	said, _ := empty.openCompose().composeSubmit(false)
	wantBand(t, asModel(t, said), "knows no repository")
}

func mustUpdate(m Model, msg tea.Msg) tea.Model {
	next, _ := m.Update(msg)
	return next
}
