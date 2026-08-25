package ui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

type noteState struct {
	open   bool
	taskID string
	text   string
}

func (m Model) openNote() Model {
	taskID := m.detail
	if taskID == "" {
		if r, ok := m.selected(); ok && !r.head {
			taskID = r.task.ID
		}
	}
	if taskID == "" {
		return m
	}
	m.note = noteState{
		open:   true,
		taskID: taskID,
		text:   "",
	}
	return m
}

func (m Model) closeNote() Model {
	m.note = noteState{}
	return m
}

func (m Model) noteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape || key.Matches(msg, m.keys.Back):
		return m.closeNote(), nil
	case msg.Code == tea.KeyEnter || key.Matches(msg, m.keys.Open):
		return m.submitNote()
	case (msg.Code == 'v' || msg.Code == 'V') && msg.Mod&tea.ModCtrl != 0:
		if clip := readClipboard(); clip != "" {
			m.note.text += clip
		}
		return m, nil
	case msg.Code == tea.KeyBackspace || msg.Code == tea.KeyDelete:
		m.note.text = trimLastRune(m.note.text)
		return m, nil
	}
	if msg.Text != "" {
		m.note.text += msg.Text
	}
	return m, nil
}

func (m Model) submitNote() (tea.Model, tea.Cmd) {
	p := m.opts.Words
	text := strings.TrimSpace(m.note.text)
	if text == "" {
		return m.say(p.T("note.empty", "the note cannot be empty")), nil
	}
	taskID := m.note.taskID
	path := "."
	for _, t := range m.board.Tasks {
		if t.ID == taskID {
			path = t.RepoPath
			if path == "" {
				path = t.Repo
			}
			break
		}
	}
	m.note = noteState{}
	m = m.say(p.T("note.recorded", "note recorded for {id}", about("id", taskID)))
	var cmds []tea.Cmd
	if m.opts.Reader != nil && m.screen == screenDetail {
		cmds = append(cmds, logOf(m.opts.Reader, m.subject()))
	}
	nextM, runCmd := m.runWatched(Command{Name: "note"}, []string{"-repo", path, taskID, "--", text})
	if runCmd != nil {
		cmds = append(cmds, runCmd)
	}
	return nextM, tea.Batch(cmds...)
}
