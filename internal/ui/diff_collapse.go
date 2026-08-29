package ui

import (
	"fmt"
	"maps"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// collapseFile closes an open file of the diff and opens a closed one.
//
// The map is cloned rather than written in place: a Model is copied by every
// method that returns one and a map is not, so a key written here would
// collapse a file on windows this method never returned.
func (m Model) collapseFile(path string) Model {
	held := maps.Clone(m.collapsedFiles)
	if held == nil {
		held = map[string]bool{}
	}

	if held[path] {
		delete(held, path)
	} else {
		held[path] = true
	}

	m.collapsedFiles = held

	return m.syncPanes()
}

// collapseFileAt is the same for the file a hit test named. A row can only
// carry the file's place in the diff, so that is what it is named by here —
// and turned back into a path at once, because a rebuild moves the rows
// under a file and does not move the file.
func (m Model) collapseFileAt(i int) Model {
	files := parseDiffFiles(strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n"))
	if i < 0 || i >= len(files) {
		return m
	}

	return m.collapseFile(files[i].Path)
}

// toggleCollapseCurrentFile toggles collapse state for the file currently in view.
func (m Model) toggleCollapseCurrentFile() Model {
	raw := strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")

	files := parseDiffFiles(raw)
	if len(files) == 0 {
		return m
	}

	idx := fileIndexAtOffset(files, m.panes[tabDiff].YOffset())
	if idx >= 0 && idx < len(files) {
		path := files[idx].Path
		m = m.collapseFile(path)
		p := m.opts.Words

		status := p.T("diff.file_expanded", "expanded")
		if m.collapsedFiles[path] {
			status = p.T("diff.file_collapsed", "collapsed")
		}

		m = m.say(fmt.Sprintf("%s: %s", path, status))
	}

	return m
}

// toggleCollapseAll toggles collapse state for all changed files.
func (m Model) toggleCollapseAll() Model {
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

	held := maps.Clone(m.collapsedFiles)
	if held == nil {
		held = map[string]bool{}
	}

	for _, f := range files {
		held[f.Path] = target
	}

	m.collapsedFiles = held

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
			m = m.collapseFile(files[m.diffFileCursor].Path)
		}

		return m, nil
	case "a":
		m = m.toggleCollapseAll()
		return m, nil
	}

	return m, nil
}
