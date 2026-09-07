package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

func TestToggleMarkdownKeyInDetailView(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{
		{ID: "ORBIT-5", Repo: "orbit", RepoPath: ".", Title: "Test task"},
	}
	m, _ = m.openDetail(m.board.Tasks[0])

	if m.rawText {
		t.Error("expected default rawText to be false (formatted by default)")
	}

	// Press 'v' to toggle to raw
	res, _ := m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})

	m = asModel(t, res)
	if !m.rawText {
		t.Error("expected rawText to be true after pressing 'v'")
	}

	// Press 'v' again to toggle back to formatted
	res, _ = m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})

	m = asModel(t, res)
	if m.rawText {
		t.Error("expected rawText to be false after pressing 'v' again")
	}
}
