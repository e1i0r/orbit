package ui

import (
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

func (m Model) hitCompose(x, y int) Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return Target{}
	}
	if line == 0 {
		if x < 20 {
			return Target{Kind: TargetComposeTab, Pane: composeTabManual}
		}
		return Target{Kind: TargetComposeTab, Pane: composeTabURL}
	}

	hasSum := m.flowSummary(m.compose.chosenFlow()) != ""
	extraSum := 0
	if hasSum {
		extraSum = 1
	}

	if m.compose.tab == composeTabManual {
		switch {
		case line == 2:
			return m.hitComposeRepoPills(x, composeRepo)
		case line == 3:
			return m.hitComposeFlowPills(x, composeFlow)
		case line == 4 && hasSum:
			return Target{Kind: TargetComposeInspectFlow}
		case line == 4+extraSum:
			return m.hitComposeEnginePills(x, composeEngine)
		case line == 5+extraSum:
			return m.hitComposeModelPills(x, composeModel)
		case line == 6+extraSum:
			return m.hitComposeThinkingPills(x, composeThinking)
		case line == 7+extraSum:
			return m.hitComposeEffortPills(x, composeEffort)
		case line == 8+extraSum:
			return Target{Kind: TargetComposeField, Pane: composeID}
		case line == 9+extraSum:
			if x >= 17 && x <= 37 {
				return Target{Kind: TargetComposePaste}
			}
			return Target{Kind: TargetComposeField, Pane: composeText}
		case line >= 10+extraSum && line <= 16+extraSum:
			return Target{Kind: TargetComposeField, Pane: composeText}
		case line >= 17+extraSum:
			return hitComposeActions(x)
		}
	} else {
		switch {
		case line == 2:
			if x >= 17 && x <= 37 {
				return Target{Kind: TargetComposePaste}
			}
			return Target{Kind: TargetComposeField, Pane: composeURL}
		case line == 3:
			return m.hitComposeRepoPills(x, composeURLRepo)
		case line == 4:
			return m.hitComposeFlowPills(x, composeURLFlow)
		case line == 5 && hasSum:
			return Target{Kind: TargetComposeInspectFlow}
		case line == 5+extraSum:
			return m.hitComposeEnginePills(x, composeURLEngine)
		case line == 6+extraSum:
			return m.hitComposeModelPills(x, composeURLModel)
		case line == 7+extraSum:
			return m.hitComposeThinkingPills(x, composeURLThinking)
		case line == 8+extraSum:
			return m.hitComposeEffortPills(x, composeURLEffort)
		case line >= 10+extraSum:
			return hitComposeActions(x)
		}
	}
	return Target{}
}

func (m Model) hitComposeRepoPills(x int, field int) Target {
	curX := 17
	for i, r := range m.compose.repos {
		pillWidth := composePillWidth(r.name, i == m.compose.repoIdx)
		if x >= curX && x < curX+pillWidth {
			return Target{Kind: TargetComposeRepoChoice, Pane: i}
		}
		curX += pillWidth + 1
	}
	return Target{Kind: TargetComposeField, Pane: field}
}

func (m Model) hitComposeFlowPills(x int, field int) Target {
	p := m.opts.Words
	curX := 17
	for i, f := range m.compose.flows {
		glyph := "⚡ "
		switch f {
		case "quick":
			glyph = "🚀 "
		case "careful":
			glyph = "🛡️ "
		}
		pillWidth := composePillWidth(glyph+f, i == m.compose.flowIdx)
		if x >= curX && x < curX+pillWidth {
			return Target{Kind: TargetComposeFlowChoice, Pane: i}
		}
		curX += pillWidth + 1
	}

	inspectText := " 👁️ " + p.T("compose.inspect_flow_btn", "inspect") + " "
	inspectWidth := lipgloss.Width(inspectText) + 2
	if x >= curX && x < curX+inspectWidth {
		return Target{Kind: TargetComposeInspectFlow}
	}
	curX += inspectWidth + 1

	newText := " ➕ " + p.T("compose.new_flow_btn", "New") + " "
	newWidth := lipgloss.Width(newText) + 2
	if x >= curX && x < curX+newWidth {
		return Target{Kind: TargetComposeNewFlow}
	}
	return Target{Kind: TargetComposeField, Pane: field}
}

func (m Model) hitComposeEnginePills(x int, field int) Target {
	curX := 17
	for i, eng := range m.compose.engines {
		pillWidth := composePillWidth(eng, i == m.compose.engineIdx)
		if x >= curX && x < curX+pillWidth {
			return Target{Kind: TargetComposeEngineChoice, Pane: i}
		}
		curX += pillWidth + 1
	}
	return Target{Kind: TargetComposeField, Pane: field}
}

func (m Model) hitComposeModelPills(x int, field int) Target {
	curX := 17
	for i, mod := range m.compose.models {
		pillWidth := composePillWidth(mod, i == m.compose.modelIdx)
		if x >= curX && x < curX+pillWidth {
			return Target{Kind: TargetComposeModelChoice, Pane: i}
		}
		curX += pillWidth + 1
	}
	return Target{Kind: TargetComposeField, Pane: field}
}

func (m Model) hitComposeThinkingPills(x int, field int) Target {
	curX := 17
	for i, th := range m.compose.thinkings {
		pillWidth := composePillWidth(th, i == m.compose.thinkingIdx)
		if x >= curX && x < curX+pillWidth {
			return Target{Kind: TargetComposeThinkingChoice, Pane: i}
		}
		curX += pillWidth + 1
	}
	return Target{Kind: TargetComposeField, Pane: field}
}

func (m Model) hitComposeEffortPills(x int, field int) Target {
	curX := 17
	for i, ef := range m.compose.efforts {
		pillWidth := composePillWidth(ef, i == m.compose.effortIdx)
		if x >= curX && x < curX+pillWidth {
			return Target{Kind: TargetComposeEffortChoice, Pane: i}
		}
		curX += pillWidth + 1
	}
	return Target{Kind: TargetComposeField, Pane: field}
}

func hitComposeActions(x int) Target {
	if x < 20 {
		return Target{Kind: TargetComposeAction, Key: "save"}
	} else if x < 50 {
		return Target{Kind: TargetComposeAction, Key: "save_and_run"}
	}
	return Target{Kind: TargetComposeAction, Key: "cancel"}
}
