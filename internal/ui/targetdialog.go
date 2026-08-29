package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Dialog and subscreen hit detection: task detail, start dialog, settings, repos, compose.

// hitDetail is the task view, one level down: its heading, the tab strip,
// and the pane under them.
func (m Model) hitDetail(x, y int) Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return Target{}
	}

	headLen := len(m.detailHeadLines(m.frame.Body.W))

	tabLine := headLen
	if m.frame.Body.H >= headLen+3 {
		tabLine = headLen + 1
	}

	bodyStart := tabLine + 1
	paneTop, _ := m.paneBand()

	switch {
	case line < tabLine:
		return Target{}
	case line == tabLine:
		return m.hitTabs(x)
	case x >= m.frame.Body.W-1 && line >= paneTop && line < m.frame.Body.H-1 && m.barShows():
		return Target{Kind: TargetScrollBar, Pane: line - paneTop}
	case m.tab == tabDiff:
		raw := strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")

		files := parseDiffFiles(raw)
		if len(files) > 0 {
			if !m.diffFilePicker {
				if line >= bodyStart && line <= bodyStart+2 {
					return Target{Kind: TargetDiffSelectToggle}
				}
			} else {
				maxItems := 7

				start := 0
				if m.diffFileCursor >= maxItems {
					start = m.diffFileCursor - maxItems + 1
				}

				end := min(len(files), start+maxItems)
				numItems := end - start

				if line == bodyStart {
					return Target{Kind: TargetDiffSelectToggle}
				}

				if line >= bodyStart+1 && line < bodyStart+1+numItems {
					return Target{Kind: TargetDiffFile, Pane: start + (line - (bodyStart + 1))}
				}

				if line >= bodyStart+1+numItems && line <= bodyStart+1+numItems+1 {
					return Target{Kind: TargetDiffSelectToggle}
				}
			}
		}

		if line < m.frame.Body.H-1 {
			// Under its file bar the diff is a pane like the other ten, and
			// a card of it folds by being pointed at.
			if at, hit := m.hitPaneContent(line - paneTop); hit {
				return at
			}

			return Target{Kind: TargetPaneBody, Pane: int(m.tab)}
		}
	case line < m.frame.Body.H-1:
		if at, hit := m.hitPaneContent(line - paneTop); hit {
			return at
		}

		return Target{Kind: TargetPaneBody, Pane: int(m.tab)}
	}

	return Target{}
}

// hitPaneContent is what one row inside a pane is, counting from the first
// row that pane drew: a section head, the rule between two attempts, or the
// head of a row that folds.
//
// The row is measured from the top of the pane rather than the top of the
// body, because the diff carries a file bar between the two and every other
// tab carries nothing — which is the same distinction paneBandFor makes when
// it sizes them.
func (m Model) hitPaneContent(row int) (Target, bool) {
	if key, ok := m.hitFold(row); ok {
		return Target{Kind: TargetFold, Key: key}, true
	}

	if n, ok := m.hitSeam(row); ok {
		return Target{Kind: TargetSeam, Pane: n}, true
	}

	if i, ok := m.hitPaneRow(row); ok {
		return Target{Kind: TargetPaneRow, Pane: i}, true
	}

	return Target{}, false
}

// hitTabs is which tab of the strip a cell is in.
func (m Model) hitTabs(x int) Target {
	for _, t := range m.placeTabs() {
		if x >= t.x && x < t.x+t.w {
			return Target{Kind: TargetPaneTab, Pane: int(t.tab)}
		}
	}

	return Target{}
}

// hitStart is the dialog that decides what a run will be: the flow line, the
// phases it is made of, and the switch under them.
func (m Model) hitStart(x, y int) Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return Target{}
	}

	p := m.startLayout(m.frame.Body.W)
	switch {
	case line == p.flow:
		return Target{Kind: TargetDialogSwitch, Field: fieldFlow}
	case line >= p.phases && line < p.phases+p.nPhases:
		return Target{Kind: TargetDialogPhase, Phase: line - p.phases}
	case line == p.autopilot:
		return Target{Kind: TargetDialogSwitch, Field: fieldAutopilotOn}
	case line == p.autopilot+1:
		return Target{Kind: TargetDialogSwitch, Field: fieldAutopilotOff}
	}

	return Target{}
}

func (m Model) hitSettings(x, y int) Target {
	line, ok := m.frame.BodyRow(y)
	if !ok || line < 4 {
		return Target{}
	}

	offset := line - 4
	rowIdx := offset / 3

	rows := m.settingRowsList()
	if rowIdx >= 0 && rowIdx < len(rows) {
		r := rows[rowIdx]

		if x >= 20 {
			curX := 20

			for i, opt := range r.options {
				pillLen := lipgloss.Width(" "+r.label(i)+" ") + 1
				if opt == r.val {
					pillLen = lipgloss.Width(" ● "+r.label(i)+" ") + 1
				}

				if x >= curX && x < curX+pillLen {
					return Target{Kind: TargetSettingsRow, Pane: rowIdx, Field: opt}
				}

				curX += pillLen
			}
		}

		return Target{Kind: TargetSettingsRow, Pane: rowIdx, Field: ""}
	}

	return Target{}
}

func (m Model) hitRepos(x, y int) Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return Target{}
	}

	rowIdx := line - 4

	repos := m.collectRepos()
	if rowIdx >= 0 && rowIdx < len(repos) {
		return Target{Kind: TargetRepo, ID: repos[rowIdx].name}
	}

	return Target{}
}
