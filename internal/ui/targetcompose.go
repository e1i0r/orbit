package ui

// Where a cell of the compose form is: the fields a task is written into and
// the rows of pills each one is chosen from.
//
// It is its own file because the form is its own screen — nine fields and
// six pill rows — and reading the task view's hit test should not mean
// reading past all of it.

import "charm.land/lipgloss/v2"

func (m Model) hitCompose(x, y int) Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return Target{}
	}

	plan := m.composeLayout()

	if line == plan.tabLine {
		if x < 20 {
			return Target{Kind: TargetComposeTab, Pane: composeTabManual}
		}

		return Target{Kind: TargetComposeTab, Pane: composeTabURL}
	}

	if m.compose.tab == composeTabManual {
		switch {
		case line == plan.repo:
			return m.hitComposeRepoPills(x, composeRepo)
		case line == plan.flow:
			return m.hitComposeFlowPills(x, composeFlow)
		case plan.flowSum != -1 && line == plan.flowSum:
			return Target{Kind: TargetComposeInspectFlow}
		case line == plan.engine:
			return m.hitComposeEnginePills(x, composeEngine)
		case line == plan.model:
			return m.hitComposeModelPills(x, composeModel)
		case line == plan.thinking:
			return m.hitComposeThinkingPills(x, composeThinking)
		case line == plan.effort:
			return m.hitComposeEffortPills(x, composeEffort)
		case line == plan.id:
			return Target{Kind: TargetComposeField, Pane: composeID}
		case line == plan.textHeader:
			if x >= 17 && x <= 37 {
				return Target{Kind: TargetComposePaste}
			}

			return Target{Kind: TargetComposeField, Pane: composeText}
		case line >= plan.textBoxTop && line <= plan.textBoxBot:
			return Target{Kind: TargetComposeField, Pane: composeText}
		case line >= plan.actions:
			return hitComposeActions(m.opts.Words, x)
		}
	} else {
		switch {
		case line == plan.url:
			if x >= 17 && x <= 37 {
				return Target{Kind: TargetComposePaste}
			}

			return Target{Kind: TargetComposeField, Pane: composeURL}
		case line == plan.repo:
			return m.hitComposeRepoPills(x, composeURLRepo)
		case line == plan.flow:
			return m.hitComposeFlowPills(x, composeURLFlow)
		case plan.flowSum != -1 && line == plan.flowSum:
			return Target{Kind: TargetComposeInspectFlow}
		case line == plan.engine:
			return m.hitComposeEnginePills(x, composeURLEngine)
		case line == plan.model:
			return m.hitComposeModelPills(x, composeURLModel)
		case line == plan.thinking:
			return m.hitComposeThinkingPills(x, composeURLThinking)
		case line == plan.effort:
			return m.hitComposeEffortPills(x, composeURLEffort)
		case line >= plan.actions:
			return hitComposeActions(m.opts.Words, x)
		}
	}

	return Target{}
}

func (m Model) hitComposeRepoPills(x int, field int) Target {
	curX := composeLabelStart

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
	curX := composeLabelStart

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

	newBtn := Pill(" ➕ "+p.T("compose.new_flow_btn", "New")+" ", "#FFFFFF", "#6366F1")

	newWidth := lipgloss.Width(newBtn)
	if x >= curX && x < curX+newWidth {
		return Target{Kind: TargetComposeNewFlow}
	}

	return Target{Kind: TargetComposeField, Pane: field}
}

func (m Model) hitComposeEnginePills(x int, field int) Target {
	curX := composeLabelStart

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
	curX := composeLabelStart

	for i := range m.compose.models {
		pillWidth := composePillWidth(m.compose.modelLabel(i), i == m.compose.modelIdx)
		if x >= curX && x < curX+pillWidth {
			return Target{Kind: TargetComposeModelChoice, Pane: i}
		}

		curX += pillWidth + 1
	}

	return Target{Kind: TargetComposeField, Pane: field}
}

func (m Model) hitComposeThinkingPills(x int, field int) Target {
	curX := composeLabelStart

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
	curX := composeLabelStart

	for i := range m.compose.efforts {
		pillWidth := composePillWidth(m.compose.effortLabel(i), i == m.compose.effortIdx)
		if x >= curX && x < curX+pillWidth {
			return Target{Kind: TargetComposeEffortChoice, Pane: i}
		}

		curX += pillWidth + 1
	}

	return Target{Kind: TargetComposeField, Pane: field}
}
