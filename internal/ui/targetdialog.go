package ui

import (
	"charm.land/lipgloss/v2"
)

// Dialog and subscreen hit detection: task detail, start dialog, settings, repos.

// hitDetail is the task view, one level down: its heading, the tab strip,
// and the pane under them.
func (m Model) hitDetail(x, y int) Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return Target{}
	}
	switch {
	case line == 0:
		return Target{}
	case line == 1:
		return m.hitTabs(x)
	case line < 2+paneHeight(m.frame.Body.H):
		return Target{Kind: TargetPaneBody, Pane: int(m.tab)}
	}
	return Target{}
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
			for _, opt := range r.options {
				pillLen := lipgloss.Width(" "+opt+" ") + 1
				if opt == r.val {
					pillLen = lipgloss.Width(" ● "+opt+" ") + 1
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
