package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/e1i0r/orbit/internal/flow"
)

func wrapPromptText(text string, maxLen int) []string {
	if strings.TrimSpace(text) == "" || maxLen <= 0 {
		return nil
	}

	var lines []string

	// A line the writer ended is a line: these fields hold paragraphs now,
	// and wrapping them as one run of words would join a list of checks
	// into a sentence.
	for _, para := range strings.Split(text, "\n") {
		lines = append(lines, wrapParagraph(para, maxLen)...)
	}

	return lines
}

// wrapOne folds one paragraph, which has no newlines left in it.
func wrapParagraph(text string, maxLen int) []string {
	wordsList := strings.Fields(text)
	if len(wordsList) == 0 {
		return []string{""}
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

		descriptors := m.flows.listed
		curLine := 6

		for i, d := range descriptors {
			fl := m.flows.shown(d.Name).flow
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

	return m.hitBuilder(x, line)
}

// hitBuilder is where a click landed in the designer.
//
// It reads the rows that were drawn rather than a table of line numbers kept
// beside them: see flowsbuilderrows.go for why there is no second opinion
// about the layout any more.
func (m Model) hitBuilder(x, line int) Target {
	lines, start := m.builderView(m.frame.Body.H, m.frame.Body.W)

	at := start + line
	if at < 0 || at >= len(lines) {
		return Target{}
	}

	row := lines[at]

	switch {
	case row.act != "":
		return Target{Kind: TargetFlowItem, Field: row.act}
	case row.strip:
		if at := m.flowTabAt(x); at >= 0 {
			return Target{Kind: TargetFlowItem, Field: "tab", Phase: at}
		}

		return Target{}
	case row.pick != noPick:
		return Target{Kind: TargetFlowItem, Field: "pick", Phase: row.pick}
	case row.phase != noPhase:
		return Target{Kind: TargetFlowItem, Field: "select_phase", Phase: row.phase}
	case row.field == noField:
		return Target{}
	case row.head && row.field == flowFieldPrompt:
		if field := promptPill(m, x); field != "" {
			return Target{Kind: TargetFlowItem, Field: field}
		}
	case row.field == flowFieldAddPhase:
		return Target{Kind: TargetFlowItem, Field: buttonAt(m, x)}
	}

	return Target{Kind: TargetFlowItem, Phase: row.field}
}

// promptPill is which of the instruction row's three pills the pointer is
// over, or nothing when it is over the label to their left.
//
// The ranges are measured off the pills themselves rather than written down,
// because a translation makes every one of them a different width.
func promptPill(m Model, x int) string {
	p := m.opts.Words
	at := 2 + labelWidth + 2

	for _, pill := range []struct {
		text  string
		field string
	}{
		{p.T("flows.btn_paste", "📋 Paste"), "paste_prompt"},
		{p.T("flows.btn_autogen", "✨ Autogenerate"), "autogen_prompt"},
		{p.T("flows.btn_clear", "🗑 Clear"), "clear_prompt"},
	} {
		wide := lipgloss.Width(pill.text) + 2
		if x >= at && x < at+wide {
			return pill.field
		}

		at += wide + 1
	}

	return ""
}

// buttonAt is which of the three buttons under the form the pointer is over,
// measured the same way.
func buttonAt(m Model, x int) string {
	p := m.opts.Words
	at := 4

	for _, btn := range []struct {
		text  string
		field string
	}{
		{p.T("flows.btn_add_phase", "+ Add Phase"), "add_phase"},
		{p.T("flows.btn_del_phase", "🗑 Delete Phase"), "del_phase"},
		{p.T("flows.btn_save_flow", "✔ Save Flow"), "save"},
	} {
		wide := lipgloss.Width(btn.text) + 2
		if x < at+wide {
			return btn.field
		}

		at += wide + 6
	}

	return "save"
}
