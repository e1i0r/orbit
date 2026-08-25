package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

func TestTabMenuInDetailView(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{
		{ID: "ORBIT-5", Repo: "orbit", RepoPath: ".", Title: "Test task"},
	}
	m, _ = m.openDetail(m.board.Tasks[0])

	// 1. Press 'm' in detail view to open tab menu
	res, _ := m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	m = asModel(t, res)
	if !m.menu.open {
		t.Fatal("expected menu to be open after pressing 'm' in detail view")
	}

	entries := m.menuEntries()
	if len(entries) != int(tabCount) {
		t.Fatalf("menuEntries in detail = %d, want %d (all tabs)", len(entries), int(tabCount))
	}

	// Verify each entry has title and description
	if entries[0].title != "overview" || entries[0].detail == "" {
		t.Errorf("entry 0 = %+v, expected overview with description", entries[0])
	}

	// 2. Select diff tab (index 9) using down arrows and press Enter
	for i := 0; i < 9; i++ {
		res, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m = asModel(t, res)
	}
	res, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = asModel(t, res)
	if m.menu.open {
		t.Error("expected menu to close after selection")
	}
	if m.tab != tabDiff {
		t.Errorf("active tab = %v, want tabDiff", m.tab)
	}

	// 3. Reopen menu and press direct shortcut '6' (timeline)
	res, _ = m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	m = asModel(t, res)
	res, _ = m.Update(tea.KeyPressMsg{Code: '6', Text: "6"})
	m = asModel(t, res)
	if m.menu.open {
		t.Error("expected menu to close after direct key shortcut")
	}
	if m.tab != tabTimeline {
		t.Errorf("active tab after '6' = %v, want tabTimeline", m.tab)
	}
}
