package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

func TestNoteLifecycle(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{
		{ID: "ORBIT-5", Repo: "orbit", RepoPath: ".", Title: "Test task"},
	}
	m.expanded = map[view.Band]bool{view.BandOf(m.board.Tasks[0]): true}
	m.cursor = m.firstTask() // on ORBIT-5

	// 1. Open note from board list using 'a'
	res, _ := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m = asModel(t, res)
	if !m.note.open {
		t.Fatal("expected note dialog to be open")
	}
	if m.note.taskID != "ORBIT-5" {
		t.Errorf("note.taskID = %q, want ORBIT-5", m.note.taskID)
	}

	// 2. Type text into note
	for _, ch := range "Revisar primero el linter" {
		res, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = asModel(t, res)
	}
	if m.note.text != "Revisar primero el linter" {
		t.Errorf("note.text = %q, want 'Revisar primero el linter'", m.note.text)
	}

	// 3. Paste extra text
	res, _ = m.Update(tea.PasteMsg{Content: " y tests"})
	m = asModel(t, res)
	if m.note.text != "Revisar primero el linter y tests" {
		t.Errorf("note.text after paste = %q", m.note.text)
	}

	// 4. Test note rendering
	rows := m.noteRows(15, 80)
	if len(rows) == 0 {
		t.Fatal("expected rendered rows from noteRows")
	}

	// 5. Esc closes note
	res, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = asModel(t, res)
	if m.note.open {
		t.Error("expected note dialog to be closed after esc")
	}
}

func TestNoteInDetailView(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board.Tasks = []view.Task{
		{ID: "ORBIT-5", Repo: "orbit", RepoPath: ".", Title: "Test task"},
	}
	m, _ = m.openDetail(m.board.Tasks[0])

	// Press 'a' in detail view (on tab 0 overview or any other tab)
	res, _ := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m = asModel(t, res)
	if !m.note.open {
		t.Fatal("expected note dialog to be open in detail view")
	}
	if m.note.taskID != "ORBIT-5" {
		t.Errorf("note.taskID in detail = %q, want ORBIT-5", m.note.taskID)
	}

	// Verify detail hints include Ask and CLI
	hints := m.detailHints()
	foundAsk, foundCLI := false, false
	for _, h := range hints {
		if h.key == m.keys.Ask.Help().Key {
			foundAsk = true
		}
		if h.key == m.keys.CLI.Help().Key {
			foundCLI = true
		}
	}
	if !foundAsk {
		t.Error("expected detailHints to contain 'a' (ask/note) hint")
	}
	if !foundCLI {
		t.Error("expected detailHints to contain 'c' (cli) hint")
	}
}
