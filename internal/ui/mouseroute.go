package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/e1i0r/orbit/internal/view"
)

// sendKey puts a keystroke through the same map a pressed key goes through,
// which is how a clicked hint reaches the verb it names.
func (m Model) sendKey(k keystroke) (tea.Model, tea.Cmd) {
	switch {
	case m.filtering:
		return m, nil
	case m.screen == screenStart:
		return m.startKey(k)
	case m.screen == screenDetail:
		return m.detailKey(k)
	}

	return m.listKey(k)
}

// flip is one of the start dialog's switches, clicked.
func (m Model) flip(field string) (tea.Model, tea.Cmd) {
	on := m.autopilotOn()
	switch {
	case field == fieldFlow:
		return m.cycleFlow(), nil
	case field == fieldAutopilotOn && !on, field == fieldAutopilotOff && on:
		return m.autopilot()
	}

	return m, nil
}

// rightClick opens the menu for what was pointed at.
func (m Model) rightClick(t Target) (tea.Model, tea.Cmd) {
	if t.Kind == TargetPaneBody {
		if s := m.subject(); s.ID != "" {
			return m.openMenu(s.ID), nil
		}

		return m, nil
	}

	i, ok := m.rowOf(t)
	if !ok {
		return m, nil
	}

	next := m.moveTo(i)
	if t.Kind == TargetTask {
		return next.openMenu(t.ID), nil
	}

	return next, nil
}

func (m Model) jumpToBand(b view.Band) (tea.Model, tea.Cmd) {
	m = m.expand(b)

	all := m.rows()
	for i, r := range all {
		if r.band == b && !r.blank {
			return m.moveTo(i).clampCursor(), nil
		}
	}

	return m, nil
}

func (m Model) handleComposeClick(t Target) (tea.Model, tea.Cmd) {
	switch t.Kind {
	case TargetComposeTab:
		m.compose.tab = t.Pane
		m.compose.field = 0

		return m, nil
	case TargetComposeRepoChoice:
		if t.Pane >= 0 && t.Pane < len(m.compose.repos) {
			m.compose.repoIdx = t.Pane
			m.compose.repo = m.compose.repos[t.Pane].name
		}

		return m, nil
	case TargetComposeFlowChoice:
		if t.Pane >= 0 && t.Pane < len(m.compose.flows) {
			if m.compose.flowIdx == t.Pane {
				return m.openFlowPreview(m.compose.flows[t.Pane]), nil
			}

			m.compose.flowIdx = t.Pane
		}

		return m, nil
	case TargetComposeNewFlow:
		return m.openFlows(), nil
	case TargetComposeInspectFlow:
		return m.openFlowPreview(m.compose.chosenFlow()), nil
	case TargetComposeField:
		m.compose.field = t.Pane
		return m, nil
	case TargetComposeCaret:
		return m.composeAim(t), nil
	case TargetComposeAction:
		switch t.Key {
		case "save":
			return m.composeSubmit(false)
		case "save_and_run":
			return m.composeSubmit(true)
		case "cancel":
			return m.abandonCompose(), nil
		}
	case TargetComposePaste:
		if clip := readClipboard(); clip != "" {
			return m.paste(clip), nil
		}
	}

	return m, nil
}
