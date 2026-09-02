package ui

// Where a cell of the compose form is: the fields a task is written into and
// the rows of pills each one is chosen from.
//
// It is its own file because the form is its own screen — three fields and
// the row of flows they are written against — and reading the task view's
// hit test should not mean reading past all of it.

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
		case line == plan.flow:
			return m.hitComposeFlowPills(x, composeFlow)
		case plan.flowSum != -1 && line >= plan.flowSum && line < plan.flowSum+plan.flowRows:
			return Target{Kind: TargetComposeInspectFlow}
		case line == plan.id:
			return caretAt(composeID, 0, x-composeLabelStart)
		case line == plan.textHeader:
			if x >= 17 && x <= 37 {
				return Target{Kind: TargetComposePaste}
			}

			return Target{Kind: TargetComposeField, Pane: composeText}
		case line >= plan.textBoxTop && line <= plan.textBoxBot:
			return caretAt(composeText, line-plan.textBoxTop, x-composeBoxStart)
		case line >= plan.actions:
			return hitComposeActions(m.opts.Words, x)
		}
	} else {
		switch {
		case line == plan.url:
			if x >= 17 && x <= 37 {
				return Target{Kind: TargetComposePaste}
			}

			return caretAt(composeURL, 0, x-composeLabelStart)
		case line == plan.flow:
			return m.hitComposeFlowPills(x, composeURLFlow)
		case plan.flowSum != -1 && line >= plan.flowSum && line < plan.flowSum+plan.flowRows:
			return Target{Kind: TargetComposeInspectFlow}
		case line >= plan.actions:
			return hitComposeActions(m.opts.Words, x)
		}
	}

	return Target{}
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

// composeBoxStart is the column the text inside the box is drawn at: two
// cells of margin and the border with its space.
const composeBoxStart = 4

// caretAt is a click inside a field, as the place in the value it points
// at. A click to the left of where the value starts is the start of it,
// which is where a reader who lands on the label meant to be.
func caretAt(field, row, col int) Target {
	return Target{Kind: TargetComposeCaret, Pane: field, Phase: row, Caret: max(col, 0)}
}
