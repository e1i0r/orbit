package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// toggleCollapseCurrentFile toggles collapse state for the file currently in view.
func (m Model) toggleCollapseCurrentFile() Model {
	if m.collapsedFiles == nil {
		m.collapsedFiles = make(map[string]bool)
	}

	raw := strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")

	files := parseDiffFiles(raw)
	if len(files) == 0 {
		return m
	}

	idx := fileIndexAtOffset(files, m.panes[tabDiff].YOffset())
	if idx >= 0 && idx < len(files) {
		path := files[idx].Path
		m.collapsedFiles[path] = !m.collapsedFiles[path]
		p := m.opts.Words

		status := p.T("diff.file_expanded", "expanded")
		if m.collapsedFiles[path] {
			status = p.T("diff.file_collapsed", "collapsed")
		}

		m = m.syncPanes().say(fmt.Sprintf("%s: %s", path, status))
	}

	return m
}

// toggleCollapseAll toggles collapse state for all changed files.
func (m Model) toggleCollapseAll() Model {
	if m.collapsedFiles == nil {
		m.collapsedFiles = make(map[string]bool)
	}

	raw := strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")

	files := parseDiffFiles(raw)
	if len(files) == 0 {
		return m
	}

	// If any file is not collapsed, collapse all. Otherwise expand all.
	allCollapsed := true

	for _, f := range files {
		if !m.collapsedFiles[f.Path] {
			allCollapsed = false
			break
		}
	}

	target := !allCollapsed
	for _, f := range files {
		m.collapsedFiles[f.Path] = target
	}

	p := m.opts.Words

	msg := p.T("diff.all_expanded", "all files expanded")
	if target {
		msg = p.T("diff.all_collapsed", "all files collapsed")
	}

	return m.syncPanes().say(msg)
}

func fileIndexAtOffset(files []diffFile, offset int) int {
	for i := len(files) - 1; i >= 0; i-- {
		if offset >= files[i].StartLine-2 {
			return i
		}
	}

	return 0
}

// openDiffFilePicker opens the interactive file picker dropdown.
func (m Model) openDiffFilePicker() Model {
	m.diffFilePicker = !m.diffFilePicker
	raw := strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")
	files := parseDiffFiles(raw)
	m.diffFileCursor = fileIndexAtOffset(files, m.panes[tabDiff].YOffset())

	return m
}

// handleDiffFilePickerKey handles keystrokes while the file selector modal is open.
func (m Model) handleDiffFilePickerKey(k fmt.Stringer) (tea.Model, tea.Cmd) {
	raw := strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")
	files := parseDiffFiles(raw)

	switch k.String() {
	case "esc", "q":
		m.diffFilePicker = false
		return m, nil
	case "up", "k":
		if m.diffFileCursor > 0 {
			m.diffFileCursor--
		}

		return m, nil
	case "down", "j":
		if m.diffFileCursor < len(files)-1 {
			m.diffFileCursor++
		}

		return m, nil
	case "enter":
		if m.diffFileCursor >= 0 && m.diffFileCursor < len(files) {
			m.diffFilePicker = false
			m.panes[tabDiff].SetYOffset(files[m.diffFileCursor].StartLine)
		}

		return m, nil
	case " ", "space":
		if m.diffFileCursor >= 0 && m.diffFileCursor < len(files) {
			if m.collapsedFiles == nil {
				m.collapsedFiles = make(map[string]bool)
			}

			path := files[m.diffFileCursor].Path
			m.collapsedFiles[path] = !m.collapsedFiles[path]
			m = m.syncPanes()
		}

		return m, nil
	case "a":
		m = m.toggleCollapseAll()
		return m, nil
	}

	return m, nil
}
