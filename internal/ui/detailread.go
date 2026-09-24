package ui

// The task view's reads: what it asks the record for when it opens, each
// off the event loop. They were at the foot of detailkeys.go, which is the
// view's key map and no place for them.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// filesOf reads what one task's directory holds, off the event loop.
func filesOf(r Reader, t view.Task) tea.Cmd {
	return func() tea.Msg {
		if r == nil {
			return filesMsg{ID: t.ID, Err: errNoRecordPort}
		}

		files, err := r.Files(t.RepoPath, t.ID)
		if err != nil {
			return filesMsg{ID: t.ID, Err: err}
		}

		return filesMsg{ID: t.ID, Files: files}
	}
}

// fileTextOf reads one file of a task's directory, off the event loop.
func fileTextOf(r Reader, t view.Task, name string) tea.Cmd {
	return func() tea.Msg {
		if r == nil {
			return fileTextMsg{ID: t.ID, Name: name, Err: errNoRecordPort}
		}

		text, err := r.FileText(t.RepoPath, t.ID, name)
		if err != nil {
			return fileTextMsg{ID: t.ID, Name: name, Err: err}
		}

		return fileTextMsg{ID: t.ID, Name: name, Text: text}
	}
}

// logOf reads one task's record, off the event loop.
func logOf(r Reader, t view.Task) tea.Cmd {
	return func() tea.Msg {
		if r == nil {
			return logMsg{ID: t.ID, Err: errNoRecordPort}
		}

		entries, err := r.Log(t.RepoPath, t.ID)
		if err != nil {
			return logMsg{ID: t.ID, Err: err}
		}

		return logMsg{ID: t.ID, Entries: entries}
	}
}
