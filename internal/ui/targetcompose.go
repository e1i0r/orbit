package ui

// Where a cell of the compose form is: the fields a task is written into and
// the rows of pills each one is chosen from.
//
// It is its own file because the form is its own screen — three fields and
// the row of flows they are written against — and reading the task view's
// hit test should not mean reading past all of it.

import (
	"charm.land/lipgloss/v2"
)

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

	// Which fields those rows are depends on the tab, and where they are
	// does not: both tabs are a flow, what that flow will do, and a box.
	flowField, boxField := composeFlow, composeText
	if m.compose.tab == composeTabURL {
		flowField, boxField = composeURLFlow, composeURL
	}

	switch {
	case line == plan.flow:
		return m.hitComposeFlowPills(x, flowField)
	case plan.flowSum != -1 && line >= plan.flowSum && line < plan.flowSum+plan.flowRows:
		return Target{Kind: TargetComposeInspectFlow}
	case m.compose.tab == composeTabManual && line == plan.id:
		return caretAt(composeID, 0, x-composeLabelStart)
	// The paste button ends the row the label and the top border are on,
	// so that row answers for it before it answers for the field.
	case line == plan.boxTop && m.onComposePaste(x):
		return Target{Kind: TargetComposePaste}
	case line == plan.boxTop, line == plan.boxBot:
		return Target{Kind: TargetComposeField, Pane: boxField}
	case line > plan.boxTop && line < plan.boxBot:
		return caretAt(boxField, line-plan.boxTop-1, x-composeBoxStart)
	case line >= plan.actions:
		return hitComposeActions(m.opts.Words, x)
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

// composeBoxStart is the column the text inside the box is drawn at: the
// box begins where every other value of the form begins, and the border
// with its space takes the two cells after that.
const composeBoxStart = composeLabelStart + 2

// onComposePaste is whether a column is over the paste button, which stands
// one cell past the right edge of the box.
//
// The width comes from the button itself because the words on it are
// translated: a reader whose language spends more cells on "Paste" would
// otherwise have half a button that listens.
func (m Model) onComposePaste(x int) bool {
	from := composeLabelStart + m.composeBoxWidth(m.frame.Body.W) + 1

	return x >= from && x < from+lipgloss.Width(composePasteTab(m.opts.Words))
}

// caretAt is a click inside a field, as the place in the value it points
// at. A click to the left of where the value starts is the start of it,
// which is where a reader who lands on the label meant to be.
func caretAt(field, row, col int) Target {
	return Target{Kind: TargetComposeCaret, Pane: field, Phase: row, Caret: max(col, 0)}
}
