package ui

import (
	"strings"
)

// jumpNextDiffFile scrolls the diff pane to the next file boundary.
func (m Model) jumpNextDiffFile() Model {
	raw := strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")
	files := parseDiffFiles(raw)
	if len(files) == 0 {
		return m
	}
	currentY := m.panes[tabDiff].YOffset()
	for _, f := range files {
		if f.StartLine > currentY+2 {
			m.panes[tabDiff].SetYOffset(f.StartLine)
			return m
		}
	}
	m.panes[tabDiff].SetYOffset(files[0].StartLine)
	return m
}

// jumpPrevDiffFile scrolls the diff pane to the previous file boundary.
func (m Model) jumpPrevDiffFile() Model {
	raw := strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")
	files := parseDiffFiles(raw)
	if len(files) == 0 {
		return m
	}
	currentY := m.panes[tabDiff].YOffset()
	for i := len(files) - 1; i >= 0; i-- {
		if files[i].StartLine < currentY-2 {
			m.panes[tabDiff].SetYOffset(files[i].StartLine)
			return m
		}
	}
	m.panes[tabDiff].SetYOffset(files[len(files)-1].StartLine)
	return m
}

// jumpNextDiffHunk scrolls the diff pane to the next hunk.
func (m Model) jumpNextDiffHunk() Model {
	raw := strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")
	currentY := m.panes[tabDiff].YOffset()
	for i, line := range raw {
		if strings.HasPrefix(line, "@@") && i > currentY+1 {
			m.panes[tabDiff].SetYOffset(i)
			return m
		}
	}
	return m
}

// jumpPrevDiffHunk scrolls the diff pane to the previous hunk.
func (m Model) jumpPrevDiffHunk() Model {
	raw := strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")
	currentY := m.panes[tabDiff].YOffset()
	for i := len(raw) - 1; i >= 0; i-- {
		if strings.HasPrefix(raw[i], "@@") && i < currentY-1 {
			m.panes[tabDiff].SetYOffset(i)
			return m
		}
	}
	return m
}
