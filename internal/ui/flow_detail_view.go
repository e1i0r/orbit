package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/flow"
)

// flowDetailRows renders the dedicated visual inspection view for a workflow:
// its purpose description, ASCII pipeline flowchart diagram, and phase cards.
func (m Model) flowDetailRows(h, w int) []string {
	st := &m.flows
	p := m.opts.Words

	var out []string

	out = append(out, "")
	// 1. Header with Flow Name and Origin Badge
	originBadge := Pill(p.T("flows.badge_custom", "Custom"), "#FFFFFF", "#6366F1")
	if st.isBuiltin {
		originBadge = Pill(p.T("flows.badge_builtin", "Built-in"), "#FFFFFF", "#0284C7")
	}

	title := "  " + Paint(Live).Bold(true).Render("⚡ "+p.T("flows.workflow_title", "Workflow")+": ") +
		Paint(Accent).Bold(true).Render(st.flowName) + "  " + originBadge
	out = append(out, title)

	// 2. Purpose / Description Box
	desc := st.description
	if desc == "" {
		desc = p.T("flows.no_description", "No description provided for this workflow.")
	}

	descLines := wrapPromptText(desc, w-8)

	out = append(out, "")

	out = append(out, "  "+Paint(Dim).Bold(true).Render(p.T("flows.purpose_label", "Purpose & When to Use:")))
	for _, dl := range descLines {
		out = append(out, "    "+Paint(OK).Render("↳ ")+Paint(Dim).Render(dl))
	}

	// 3. Visual Pipeline Flowchart Diagram
	out = append(out, "")
	out = append(out, "  "+Paint(Accent).Bold(true).Render(p.T("flows.pipeline_diagram", "Pipeline Flowchart:")))

	diagramLines := renderFlowDiagram(st.phases, w-4)
	for _, dl := range diagramLines {
		out = append(out, "  "+dl)
	}

	// 4. Phase Breakdown Cards
	out = append(out, "")
	out = append(out, "  "+Paint(Live).Bold(true).Render(p.T("flows.phase_breakdown", "Phases Breakdown:")))

	for i, ph := range st.phases {
		if ph.Loop != nil {
			out = append(out, m.loopCard(i, ph, w)...)
			continue
		}

		badgeText := fmt.Sprintf("%s/%s", ph.Engine, orDef(ph.Model, "default"))
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

		cardHdr := fmt.Sprintf("    [%s %d: %s] (%s)",
			p.T("flows.phase_label", "Phase"), i+1, ph.Name, badgeText)
		out = append(out, Paint(Accent).Bold(true).Render(cardHdr))

		if ph.Prompt != "" {
			prmLines := wrapPromptText(`"`+ph.Prompt+`"`, w-14)
			for _, pl := range prmLines {
				out = append(out, "       "+Paint(Dim).Render(pl))
			}
		}
	}

	// 5. Actions Footer
	out = append(out, "")
	selectBtn := Pill(" ↵ "+p.T("flows.btn_select_return", "Select & Return")+" ", "#FFFFFF", "#16A34A")
	editBtn := Pill(" e "+p.T("flows.btn_edit_designer", "Edit in Designer")+" ", "#FFFFFF", "#4F46E5")
	backBtn := Pill(" esc "+p.T("flows.btn_back", "Back")+" ", "#FFFFFF", "#334155")
	out = append(out, "  "+selectBtn+"   "+editBtn+"   "+backBtn, "")

	hints := p.T("flows.detail_hints", "[enter] select · [e] edit · [esc] return")
	out = append(out, fit("  "+Paint(Dim).Render(hints), w))

	return fill(out, h)
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

		line2 := fmt.Sprintf("%s/%s", ph.Engine, orDef(ph.Model, "def"))
		if ph.FeedOutput {
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

		padLine1 := pad(line1, boxW-2, false)
		padLine2 := pad(line2, boxW-2, false)

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
			Paint(Dim).Render(rowTop),
			Paint(Accent).Render(rowMid1),
			Paint(OK).Render(rowMid2),
			Paint(Dim).Render(rowBot),
		}
	}

	// Fallback to vertical stack when horizontal space is limited
	var out []string

	for i, b := range boxes {
		if i > 0 {
			out = append(out, "        │", "        ▼")
		}

		out = append(out,
			Paint(Dim).Render("  "+b.top),
			Paint(Accent).Render("  "+b.mid1),
			Paint(OK).Render("  "+b.mid2),
			Paint(Dim).Render("  "+b.bot),
		)
	}

	return out
}
