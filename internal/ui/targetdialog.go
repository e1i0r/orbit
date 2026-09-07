package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/patch"
	"github.com/e1i0r/orbit/internal/ui/point"
)

// Dialog and subscreen hit detection: task detail, start dialog, settings, repos, compose.

// hitDetail is the task view, one level down: its heading, the tab strip,
// and the pane under them.
func (m Model) hitDetail(x, y int) point.Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return point.Target{}
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
		return point.Target{}
	case line == tabLine:
		return m.hitTabs(x)
	case x >= m.frame.Body.W-1 && line >= paneTop && line < m.frame.Body.H-1 && m.barShows():
		return point.Target{Kind: point.ScrollBar, Pane: line - paneTop}
	case m.tab == tabDiff:
		raw := strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n")

		files := patch.Files(raw)
		if len(files) > 0 {
			if !m.diffFilePicker {
				if line >= bodyStart && line <= bodyStart+2 {
					return point.Target{Kind: point.DiffSelectToggle}
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
					return point.Target{Kind: point.DiffSelectToggle}
				}

				if line >= bodyStart+1 && line < bodyStart+1+numItems {
					return point.Target{Kind: point.DiffFile, Pane: start + (line - (bodyStart + 1))}
				}

				if line >= bodyStart+1+numItems && line <= bodyStart+1+numItems+1 {
					return point.Target{Kind: point.DiffSelectToggle}
				}
			}
		}

		if line < m.frame.Body.H-1 {
			// Under its file bar the diff is a pane like the other ten, and
			// a card of it folds by being pointed at.
			if at, hit := m.hitPaneContent(line - paneTop); hit {
				return at
			}

			return point.Target{Kind: point.PaneBody, Pane: int(m.tab)}
		}
	case line < m.frame.Body.H-1:
		if at, hit := m.hitPaneContent(line - paneTop); hit {
			return at
		}

		return point.Target{Kind: point.PaneBody, Pane: int(m.tab)}
	}

	return point.Target{}
}

// hitPaneContent is what one row inside a pane is, counting from the first
// row that pane drew: a section head, the rule between two attempts, or the
// head of a row that folds.
//
// The row is measured from the top of the pane rather than the top of the
// body, because the diff carries a file bar between the two and every other
// tab carries nothing — which is the same distinction paneBandFor makes when
// it sizes them.
func (m Model) hitPaneContent(row int) (point.Target, bool) {
	if key, ok := m.hitFold(row); ok {
		return point.Target{Kind: point.Fold, Key: key}, true
	}

	if n, ok := m.hitSeam(row); ok {
		return point.Target{Kind: point.Seam, Pane: n}, true
	}

	if i, ok := m.hitPaneRow(row); ok {
		return point.Target{Kind: point.PaneRow, Pane: i}, true
	}

	return point.Target{}, false
}

// hitTabs is which tab of the strip a cell is in.
func (m Model) hitTabs(x int) point.Target {
	for _, t := range m.placeTabs() {
		if x >= t.x && x < t.x+t.w {
			return point.Target{Kind: point.PaneTab, Pane: int(t.tab)}
		}
	}

	return point.Target{}
}

// hitStart is the dialog that decides what a run will be: the flow line, the
// phases it is made of, and the switch under them.
func (m Model) hitStart(x, y int) point.Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return point.Target{}
	}

	p := m.startLayout(m.frame.Body.W)
	switch {
	case line == p.flow:
		return point.Target{Kind: point.DialogSwitch, Field: fieldFlow}
	case line >= p.phases && line < p.phases+p.nPhases:
		return point.Target{Kind: point.DialogPhase, Phase: line - p.phases}
	case line == p.autopilot:
		return point.Target{Kind: point.DialogSwitch, Field: fieldAutopilotOn}
	case line == p.autopilot+1:
		return point.Target{Kind: point.DialogSwitch, Field: fieldAutopilotOff}
	}

	return point.Target{}
}

func (m Model) hitSettings(x, y int) point.Target {
	line, ok := m.frame.BodyRow(y)
	if !ok || line < 4 {
		return point.Target{}
	}

	offset := line - 4
	rowIdx := offset / 3

	rows := m.settingRowsList()
	if rowIdx >= 0 && rowIdx < len(rows) {
		r := rows[rowIdx]

		if x >= 20 {
			curX := 20

			for i, opt := range r.Options {
				pillLen := lipgloss.Width(" "+r.Label(i)+" ") + 1
				if opt == r.Val {
					pillLen = lipgloss.Width(" ● "+r.Label(i)+" ") + 1
				}

				if x >= curX && x < curX+pillLen {
					return point.Target{Kind: point.SettingsRow, Pane: rowIdx, Field: opt}
				}

				curX += pillLen
			}
		}

		return point.Target{Kind: point.SettingsRow, Pane: rowIdx, Field: ""}
	}

	return point.Target{}
}

func (m Model) hitRepos(x, y int) point.Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return point.Target{}
	}

	rowIdx := line - 4

	repos := m.collectRepos()
	if rowIdx >= 0 && rowIdx < len(repos) {
		return point.Target{Kind: point.Repo, ID: repos[rowIdx].Name}
	}

	return point.Target{}
}
