package flows

import (
	"fmt"
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
	originBadge := theme.Pill(p.T("flows.badge_custom", "Custom"), theme.PillInk, theme.PillCreate)
	if s.isBuiltin {
		originBadge = theme.Pill(p.T("flows.badge_builtin", "Built-in"), theme.PillInk, theme.PillDetails)
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

	return s.detailFramed(out, h, w, e)
}

// detailFramed puts the reading in the room there is: the buttons and the
// keys under them pinned to the floor, and everything above them scrolling
// between the top of the body and that.
//
// A flow of three phases is a diagram, a card each and a footer — more rows
// than a short terminal has. It was drawn whole and then cut, so a reader on
// anything under about thirty rows saw the first phase and nothing else: not
// the rest of the phases, not the buttons, and not the line that says which
// keys do what.
func (s State) detailFramed(body []string, h, w int, e Env) []string {
	foot, _ := s.detailFoot(h, w, e)

	rows := max(h-len(foot), 0)
	if rows <= 0 {
		return cells.Fill(foot, h)
	}

	start := min(max(s.scroll, 0), max(0, len(body)-rows))

	cw := max(w-2, 1)
	track := cells.Track(rows, len(body), start)

	out := make([]string, 0, h)

	for i := range rows {
		at := start + i
		if at >= len(body) {
			break
		}

		row := body[at]
		if track != nil {
			row = cells.PadRight(cells.Fit(row, cw), cw) + track[i]
		}

		out = append(out, row)
	}

	return append(cells.Fill(out, rows), foot...)
}

// detailFoot is the three buttons and the line of keys under them, which
// stay on the floor however far the reading has been scrolled: they are
// what a reader who has read to the end reaches for. It answers the rows
// and which of them carries the buttons, because a click has to find that
// row and counting it twice is how the two readings drift apart.
//
// The blank rows that set it off are the first thing given up: on a window
// with four rows of body they are the difference between two rows of the
// flow and none at all.
func (s State) detailFoot(h, w int, e Env) (rows []string, buttons int) {
	p := e.Words

	line := "  "

	for i, b := range detailButtons(e) {
		if i > 0 {
			line += strings.Repeat(" ", detailButtonGap)
		}

		line += b.pill
	}

	pills := cells.Fit(line, w)
	hints := cells.Fit("  "+theme.Paint(theme.Dim).Render(
		p.T("flows.detail_hints", "[enter] select · [e] edit · [esc] return")), w)

	if h >= detailRoomForBlanks {
		return []string{"", pills, "", hints}, 1
	}

	return []string{pills, hints}, 0
}

// detailRoomForBlanks is the body height at which the footer can afford the
// blank rows that set it off and still leave two rows of the flow.
const detailRoomForBlanks = 6

// detailButton is one of the three: the pill as it is drawn, and what a
// click on it asks for.
type detailButton struct {
	pill  string
	field string
}

// detailButtonGap is the space between two of them.
const detailButtonGap = 3

// detailButtons is the row of them, in the order they are drawn.
//
// One reading, two uses: the drawing lays them out and the hit-test measures
// the same pills. They were two — a drawing that translated its labels and a
// hit-test written against two column numbers — so the zones a click landed
// in had nothing to do with where the buttons were, in either language.
func detailButtons(e Env) []detailButton {
	p := e.Words

	return []detailButton{
		{
			pill: theme.Pill(" ↵ "+p.T("flows.btn_select_return", "Select & Return")+" ",
				theme.PillInk, theme.PillSelect),
			field: "detail_select",
		},
		{
			pill: theme.Pill(" e "+p.T("flows.btn_edit_designer", "Edit in Designer")+" ",
				theme.PillInk, theme.PillDesign),
			field: "edit",
		},
		{
			pill:  theme.Pill(" esc "+p.T("flows.btn_back", "Back")+" ", theme.PillInk, theme.PillBack),
			field: "detail_back",
		},
	}
}

// detailButtonAt is which button a column of the buttons row holds, and
// whether it holds one at all.
func detailButtonAt(x int, e Env) (detailButton, bool) {
	at := 2 // the gutter the row starts in

	for _, b := range detailButtons(e) {
		wide := lipgloss.Width(b.pill)
		if x >= at && x < at+wide {
			return b, true
		}

		at += wide + detailButtonGap
	}

	return detailButton{}, false
}

// badgeIndent is how far in the badges are drawn when they will not fit
// beside the phase's name.
const badgeIndent = 6

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

	head := fmt.Sprintf("    [%s %d: %s]", p.T("flows.phase_label", "Phase"), i+1, ph.Name)

	// One line while there is room for one. A phase that runs on a named
	// model, at an effort, with thinking turned on, that is fed the last
	// output and stops for a human carries five badges: a hundred and
	// twenty-five cells, which wrapped on every terminal narrower than
	// that — and a card that takes an extra row moves every card under it
	// down, on a screen that is read by counting phases.
	out := []string{theme.Paint(theme.Accent).Bold(true).Render(head + " (" + badgeText + ")")}
	if lipgloss.Width(out[0]) > w {
		out = []string{theme.Paint(theme.Accent).Bold(true).Render(cells.Fit(head, w))}
		for _, l := range cells.Lines(badgeText, max(w-badgeIndent, 8)) {
			out = append(out, strings.Repeat(" ", badgeIndent)+theme.Paint(theme.Dim).Render(l))
		}
	}

	if ph.Prompt != "" {
		for _, pl := range cells.WrapKeeping(`"`+ph.Prompt+`"`, w-14) {
			out = append(out, "       "+theme.Paint(theme.Dim).Render(pl))
		}
	}

	return out
}
