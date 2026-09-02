package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

const composeLabelWidth = 14

func composeLabel(label string, active bool) string {
	mark := strings.Repeat(" ", gutter)
	if active {
		mark = markGlyph + strings.Repeat(" ", gutter-1)
	}

	padded := pad(label+":", composeLabelWidth, false)

	return mark + Paint(Dim).Render(padded) + " "
}

func (m Model) composeFlowLine(active bool, w int) string {
	p := m.opts.Words
	prefix := composeLabel(p.T("compose.flow", "flow"), active)

	var pills []string

	for i, f := range m.compose.flows {
		selected := i == m.compose.flowIdx
		glyph := "⚡ "

		switch f {
		case "quick":
			glyph = "🚀 "
		case "careful":
			glyph = "🛡️ "
		}

		if selected {
			pills = append(pills, Pill(" ● "+glyph+f+" ", "#000000", "#A855F7"))
		} else {
			pills = append(pills, Pill(" "+glyph+f+" ", "#94A3B8", "#1E293B"))
		}
	}

	newBtn := Pill(" ➕ "+p.T("compose.new_flow_btn", "New")+" ", "#FFFFFF", "#6366F1")
	pills = append(pills, newBtn)

	line := prefix + strings.Join(pills, " ")
	if active {
		line += " " + Paint(Dim).Render(p.T("compose.flow_hint", "(←/→ to cycle, click again/i for details, + new)"))
	}

	return fit(line, w)
}

// flowDetail is what the flow the form is set to will actually do: one row
// per phase, and what the flow says about itself under them.
//
// It was one row for all of it — every phase and the whole description run
// together behind a "·" — which at any terminal width ended in an ellipsis,
// so the sentence a reader was choosing between flows on was the half of it
// that fitted. A phase is a line because a flow is a sequence, and a
// sequence read as a paragraph is a sequence nobody counts.
//
// The description is wrapped to the room left beside the label and cut at
// three rows: it is prose somebody wrote about their own flow, and a form
// that grows without bound underneath the field being edited pushes the task
// box off a short terminal.
func (m Model) flowDetail(name string, w int) []string {
	fl, err := flow.Resolve(m.opts.Flows, name)
	if err != nil {
		return nil
	}

	out := make([]string, 0, len(fl.Phases)+flowAboutRows)

	for i, ph := range fl.Phases {
		out = append(out, m.flowPhaseRow(i+1, ph))
	}

	about := wrapPromptText(fl.Description, max(w-composeLabelStart-2, minFlowAbout))
	if len(about) > flowAboutRows {
		about = append(about[:flowAboutRows-1], about[flowAboutRows-1]+"…")
	}

	return append(out, about...)
}

// flowAboutRows is how many rows of a flow's own description are shown, and
// minFlowAbout the narrowest the wrap is allowed to get — a pane too narrow
// to hold the label and a word would otherwise wrap one letter per row.
const (
	flowAboutRows = 3
	minFlowAbout  = 20
)

// flowPhaseRow is one phase: what it is called, what runs it, and whether it
// stops there.
//
// The pause is a word and not only the glyph it used to be. A flow that
// stops halfway and waits is the one thing about a flow that surprises the
// person who chose it, and ⏸ alone is a symbol a reader has to already know.
func (m Model) flowPhaseRow(n int, ph flow.Phase) string {
	row := fmt.Sprintf("%d %s", n, ph.Name)

	if ph.Engine != "" {
		engine := ph.Engine
		if ph.Model != "" && ph.Model != "default" {
			engine += "/" + ph.Model
		}

		row += " · " + engine
	}

	if ph.Wait {
		return row + " · " + Paint(Warn).Render("⏸ "+m.opts.Words.T("compose.flow_waits", "waits for you"))
	}

	return row
}

const composeLabelStart = gutter + composeLabelWidth + 1

func composePillWidth(name string, selected bool) int {
	if selected {
		return lipgloss.Width(Pill(" ● "+name+" ", "#000000", "#FFFFFF"))
	}

	return lipgloss.Width(Pill(" "+name+" ", "#94A3B8", "#1E293B"))
}
