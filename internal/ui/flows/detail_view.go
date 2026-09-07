package flows

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// flowDetailRows renders the dedicated visual inspection view for a workflow:
// its purpose description, ASCII pipeline flowchart diagram, and phase cards.
func (s State) flowDetailRows(h, w int, e Env) []string {
	p := e.Words

	var out []string

	out = append(out, "")
	// 1. Header with Flow Name and Origin Badge
	originBadge := theme.Pill(p.T("flows.badge_custom", "Custom"), "#FFFFFF", "#6366F1")
	if s.isBuiltin {
		originBadge = theme.Pill(p.T("flows.badge_builtin", "Built-in"), "#FFFFFF", "#0284C7")
	}

	title := "  " + theme.Paint(theme.Live).Bold(true).Render("⚡ "+p.T("flows.workflow_title", "Workflow")+": ") +
		theme.Paint(theme.Accent).Bold(true).Render(s.flowName) + "  " + originBadge
	out = append(out, title)

	// 2. Purpose / Description Box
	desc := s.description
	if desc == "" {
		desc = p.T("flows.no_description", "No description provided for this workflow.")
	}

	descLines := cells.WrapKeeping(desc, w-8)

	out = append(out, "")

	out = append(out, "  "+theme.Paint(theme.Dim).Bold(true).Render(p.T("flows.purpose_label", "Purpose & When to Use:")))
	for _, dl := range descLines {
		out = append(out, "    "+theme.Paint(theme.OK).Render("↳ ")+theme.Paint(theme.Dim).Render(dl))
	}

	// 3. Visual Pipeline Flowchart Diagram
	out = append(out, "")
	out = append(out, "  "+theme.Paint(theme.Accent).Bold(true).Render(p.T("flows.pipeline_diagram", "Pipeline Flowchart:")))

	diagramLines := renderFlowDiagram(s.phases, w-4)
	for _, dl := range diagramLines {
		out = append(out, "  "+dl)
	}

	// 4. Phase Breakdown Cards
	out = append(out, "")
	out = append(out, "  "+theme.Paint(theme.Live).Bold(true).Render(p.T("flows.phase_breakdown", "Phases Breakdown:")))
	out = append(out, s.phaseCards(s.phases, w, e)...)

	// 5. Actions Footer
	out = append(out, "")
	selectBtn := theme.Pill(" ↵ "+p.T("flows.btn_select_return", "Select & Return")+" ", "#FFFFFF", "#16A34A")
	editBtn := theme.Pill(" e "+p.T("flows.btn_edit_designer", "Edit in Designer")+" ", "#FFFFFF", "#4F46E5")
	backBtn := theme.Pill(" esc "+p.T("flows.btn_back", "Back")+" ", "#FFFFFF", "#334155")
	out = append(out, "  "+selectBtn+"   "+editBtn+"   "+backBtn, "")

	hints := p.T("flows.detail_hints", "[enter] select · [e] edit · [esc] return")
	out = append(out, cells.Fit("  "+theme.Paint(theme.Dim).Render(hints), w))

	return cells.Fill(out, h)
}

// renderFlowDiagram builds an ASCII box-and-arrow flowchart diagram for the given phases.
func renderFlowDiagram(phases []flow.Phase, maxW int) []string {
	if len(phases) == 0 {
		return nil
	}

	type box struct {
		top, mid1, mid2, bot string
		width                int
	}

	var boxes []box

	for i, ph := range phases {
		line1 := fmt.Sprintf("%d. %s", i+1, ph.Name)

		// A loop names no engine of its own — what runs is the phase inside
		// it — so the box says how many turns it takes, where a phase says
		// what runs it. It used to say "/def": an empty engine and a model
		// nobody set, which reads as a phase that was never configured.
		line2 := fmt.Sprintf("%s/%s", ph.Engine, cells.OrDef(ph.Model, "def"))
		if ph.Loop != nil {
			line2 = "↻ ×" + strconv.Itoa(ph.Loop.Max)
		}

		if runsIn(ph).FeedOutput {
			line2 += " ➔"
		}

		if ph.Wait {
			line2 += " ⏸"
		}

		w1 := lipgloss.Width(line1)
		w2 := lipgloss.Width(line2)

		boxW := w1
		if w2 > boxW {
			boxW = w2
		}

		boxW += 2
		if boxW < 14 {
			boxW = 14
		}

		padLine1 := cells.Pad(line1, boxW-2, false)
		padLine2 := cells.Pad(line2, boxW-2, false)

		b := box{
			top:   "┌" + strings.Repeat("─", boxW-2) + "┐",
			mid1:  "│" + padLine1 + "│",
			mid2:  "│" + padLine2 + "│",
			bot:   "└" + strings.Repeat("─", boxW-2) + "┘",
			width: boxW,
		}
		boxes = append(boxes, b)
	}

	arrow := " ──▶ "
	arrowPad := "     "

	var rowTop, rowMid1, rowMid2, rowBot string

	for i, b := range boxes {
		if i > 0 {
			rowTop += arrowPad
			rowMid1 += arrow
			rowMid2 += arrowPad
			rowBot += arrowPad
		}

		rowTop += b.top
		rowMid1 += b.mid1
		rowMid2 += b.mid2
		rowBot += b.bot
	}

	if lipgloss.Width(rowTop) <= maxW {
		return []string{
			theme.Paint(theme.Dim).Render(rowTop),
			theme.Paint(theme.Accent).Render(rowMid1),
			theme.Paint(theme.OK).Render(rowMid2),
			theme.Paint(theme.Dim).Render(rowBot),
		}
	}

	// Fallback to vertical stack when horizontal space is limited
	var out []string

	for i, b := range boxes {
		if i > 0 {
			out = append(out, "        │", "        ▼")
		}

		out = append(out,
			theme.Paint(theme.Dim).Render("  "+b.top),
			theme.Paint(theme.Accent).Render("  "+b.mid1),
			theme.Paint(theme.OK).Render("  "+b.mid2),
			theme.Paint(theme.Dim).Render("  "+b.bot),
		)
	}

	return out
}

// phaseCards is one card per phase: what runs it, how it is joined to the
// phase before, and what it is told to do.
//
// It is shared by the flow inspector and the designer's diagram tab, because
// they are two windows onto the same list of phases and a second copy of
// this would be a second answer to "what does this flow do".
func (s State) phaseCards(phases []flow.Phase, w int, e Env) []string {
	var out []string

	for i, ph := range phases {
		out = append(out, s.phaseCard(i, ph, w, e)...)
	}

	return out
}

// phaseCard is one of them, numbered from where it sits in the flow.
func (s State) phaseCard(i int, ph flow.Phase, w int, e Env) []string {
	p := e.Words

	if ph.Loop != nil {
		return s.loopCard(i, ph, w, e)
	}

	badgeText := fmt.Sprintf("%s/%s", ph.Engine, cells.OrDef(ph.Model, "default"))
	if ph.Effort != "" && ph.Effort != "default" {
		badgeText += " · " + p.T("flows.effort_badge", "effort: {v}", about("v", ph.Effort))
	}

	if ph.Thinking != "" && ph.Thinking != "adaptive" {
		badgeText += " · " + p.T("flows.thinking_badge", "thinking: {v}", about("v", ph.Thinking))
	}

	if ph.FeedOutput {
		badgeText += " · " + p.T("flows.feed_badge", "feeds output")
	}

	if ph.Wait {
		badgeText += " · " + p.T("flows.gate_badge", "⏸ human gate")
	}

	out := []string{theme.Paint(theme.Accent).Bold(true).Render(fmt.Sprintf("    [%s %d: %s] (%s)",
		p.T("flows.phase_label", "Phase"), i+1, ph.Name, badgeText))}

	if ph.Prompt != "" {
		for _, pl := range cells.WrapKeeping(`"`+ph.Prompt+`"`, w-14) {
			out = append(out, "       "+theme.Paint(theme.Dim).Render(pl))
		}
	}

	return out
}
