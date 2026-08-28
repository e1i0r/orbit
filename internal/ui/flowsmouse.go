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
		switch {
		case curr == "":
			curr = wd
		case lipgloss.Width(curr)+1+lipgloss.Width(wd) <= maxLen:
			curr += " " + wd
		default:
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
	return renderComboPillsLabelled(options, nil, current)
}

// renderComboPillsLabelled is the same row for a dial whose ids and labels
// are not the same string, which is opencode's models: the id it is picked
// by is opencode/claude-opus-5 and what belongs on a pill is the rest.
func renderComboPillsLabelled(ids, labels []string, current string) string {
	var views []string

	for i, id := range ids {
		label := dialLabel(ids, labels, i)
		if id == current {
			views = append(views, Paint(Sel).Render(" "+label+" "))
		} else {
			views = append(views, Paint(Dim).Render(label))
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

	if m.flows.showingDetail {
		rows := m.flowDetailRows(m.frame.Body.H, m.frame.Body.W)
		if line >= len(rows)-3 {
			if x < 25 {
				return Target{Kind: TargetFlowItem, Field: "detail_select"}
			} else if x < 50 {
				return Target{Kind: TargetFlowItem, Field: "edit", ID: m.flows.flowName}
			}

			return Target{Kind: TargetFlowItem, Field: "detail_back"}
		}

		return Target{}
	}

	if !m.flows.creating {
		if line == 4 {
			return Target{Kind: TargetFlowItem, Field: "create"}
		}

		descriptors := flow.List(m.opts.Flows)
		curLine := 6

		for i, d := range descriptors {
			//nolint:errcheck // best-effort flow descriptor resolution
			fl, _ := flow.Resolve(m.opts.Flows, d.Name)
			phaseCount := len(fl.Phases)

			extraDesc := 0
			if fl.Description != "" {
				extraDesc = 1
			}

			if line >= curLine && line <= curLine+phaseCount+extraDesc {
				m.flows.sel = i
				if line == curLine {
					originStr := flowOriginString(m.opts.Words, d.Origin)

					offset := gutter + lipgloss.Width(d.Name)
					if originStr != "" {
						offset += 2 + lipgloss.Width(originStr) + 2
					}

					offset += 3

					detW := lipgloss.Width("👁 "+m.opts.Words.T("flows.btn_view_details", "Details")) + 4
					editW := lipgloss.Width("✏ "+m.opts.Words.T("flows.btn_edit", "Edit")) + 4

					if x >= offset+detW && x < offset+detW+editW {
						return Target{Kind: TargetFlowItem, Field: "edit", ID: d.Name}
					}

					if d.Origin != flow.OriginBuiltin && x >= offset+detW+editW {
						return Target{Kind: TargetFlowItem, Field: "delete", ID: d.Name}
					}
				}

				return Target{Kind: TargetFlowItem, Field: "details", ID: d.Name}
			}

			curLine += 1 + extraDesc + phaseCount + 1
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
		return Target{Kind: TargetFlowItem, Phase: flowFieldDescription}
	case fLine == 3:
		return Target{Kind: TargetFlowItem, Phase: flowFieldPhaseSelect}
	case fLine == 4:
		return Target{Kind: TargetFlowItem, Phase: flowFieldPhaseName}
	case fLine == 5:
		return Target{Kind: TargetFlowItem, Phase: flowFieldEngine}
	case fLine == 6:
		return Target{Kind: TargetFlowItem, Phase: flowFieldModel}
	case fLine == 7:
		return Target{Kind: TargetFlowItem, Phase: flowFieldEffort}
	case fLine == 8:
		return Target{Kind: TargetFlowItem, Phase: flowFieldThinking}
	case fLine == 9:
		return Target{Kind: TargetFlowItem, Phase: flowFieldFeedOutput}
	case fLine == 10:
		return Target{Kind: TargetFlowItem, Phase: flowFieldWait}
	case fLine == 11:
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
	case fLine > 11 && fLine <= 11+pLines+2:
		return Target{Kind: TargetFlowItem, Phase: flowFieldPrompt}
	case fLine >= 11+pLines+3:
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
