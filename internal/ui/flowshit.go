package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/e1i0r/orbit/internal/flow"
)

func wrapPromptText(text string, maxLen int) []string {
	if text == "" || maxLen <= 0 {
		return nil
	}
	wordsList := strings.Fields(text)
	if len(wordsList) == 0 {
		return nil
	}
	var lines []string
	curr := ""
	for _, wd := range wordsList {
		if curr == "" {
			curr = wd
		} else if lipgloss.Width(curr)+1+lipgloss.Width(wd) <= maxLen {
			curr += " " + wd
		} else {
			lines = append(lines, curr)
			curr = wd
		}
	}
	if curr != "" {
		lines = append(lines, curr)
	}
	return lines
}

func renderComboPills(options []string, current string) string {
	var views []string
	for _, opt := range options {
		if opt == current {
			views = append(views, Paint(Sel).Render(" "+opt+" "))
		} else {
			views = append(views, Paint(Dim).Render(opt))
		}
	}
	return strings.Join(views, " ")
}

func nextOption(options []string, current string, delta int) string {
	if len(options) == 0 {
		return current
	}
	idx := 0
	for i, opt := range options {
		if opt == current {
			idx = i
			break
		}
	}
	nextIdx := (idx + delta) % len(options)
	if nextIdx < 0 {
		nextIdx += len(options)
	}
	return options[nextIdx]
}

func (m Model) hitFlows(x, y int) Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return Target{}
	}
	if !m.flows.creating {
		if line == 4 {
			return Target{Kind: TargetFlowItem, Field: "create"}
		}
		descriptors := flow.List(m.opts.Flows)
		curLine := 6
		for i, d := range descriptors {
			fl, _ := flow.Resolve(m.opts.Flows, d.Name)
			phaseCount := len(fl.Phases)
			if line >= curLine && line <= curLine+phaseCount {
				m.flows.sel = i
				if d.Origin != flow.OriginBuiltin && x >= 32 {
					return Target{Kind: TargetFlowItem, Field: "delete", ID: d.Name}
				}
				return Target{Kind: TargetFlowItem, Field: "edit", ID: d.Name}
			}
			curLine += 1 + phaseCount + 1
		}
		return Target{}
	}

	st := &m.flows
	// Overview lines
	curL := 5
	for idx, ph := range st.phases {
		startL := curL
		curL++
		if ph.Prompt != "" {
			curL++
		}
		if line >= startL && line < curL {
			return Target{Kind: TargetFlowItem, Field: "select_phase", Phase: idx}
		}
	}
	overviewLines := curL + 1

	pLines := len(wrapPromptText(st.cur().Prompt, 76))
	if pLines < 1 {
		pLines = 1
	}
	if pLines > 4 {
		pLines = 4
	}

	fLine := line - overviewLines
	switch {
	case fLine == 0:
		return Target{Kind: TargetFlowItem, Phase: flowFieldTemplate}
	case fLine == 1:
		return Target{Kind: TargetFlowItem, Phase: flowFieldName}
	case fLine == 2:
		return Target{Kind: TargetFlowItem, Phase: flowFieldPhaseSelect}
	case fLine == 3:
		return Target{Kind: TargetFlowItem, Phase: flowFieldPhaseName}
	case fLine == 4:
		return Target{Kind: TargetFlowItem, Phase: flowFieldEngine}
	case fLine == 5:
		return Target{Kind: TargetFlowItem, Phase: flowFieldModel}
	case fLine == 6:
		return Target{Kind: TargetFlowItem, Phase: flowFieldEffort}
	case fLine == 7:
		return Target{Kind: TargetFlowItem, Phase: flowFieldThinking}
	case fLine == 8:
		return Target{Kind: TargetFlowItem, Phase: flowFieldFeedOutput}
	case fLine == 9:
		return Target{Kind: TargetFlowItem, Phase: flowFieldWait}
	case fLine == 10:
		if x >= 27 && x < 39 {
			return Target{Kind: TargetFlowItem, Field: "paste_prompt"}
		}
		if x >= 39 && x < 57 {
			return Target{Kind: TargetFlowItem, Field: "autogen_prompt"}
		}
		if x >= 57 {
			return Target{Kind: TargetFlowItem, Field: "clear_prompt"}
		}
		return Target{Kind: TargetFlowItem, Phase: flowFieldPrompt}
	case fLine > 10 && fLine <= 10+pLines+2:
		return Target{Kind: TargetFlowItem, Phase: flowFieldPrompt}
	case fLine >= 10+pLines+3:
		if x < 25 {
			return Target{Kind: TargetFlowItem, Field: "add_phase"}
		}
		if x < 45 {
			return Target{Kind: TargetFlowItem, Field: "del_phase"}
		}
		return Target{Kind: TargetFlowItem, Field: "save"}
	}
	return Target{}
}
