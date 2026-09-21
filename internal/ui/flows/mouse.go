package flows

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

func renderComboPills(options []string, current string) string {
	return renderComboPillsLabelled(options, nil, current)
}

// renderComboPillsLabelled is the same row for a dial whose ids and labels
// are not the same string, which is opencode's models: the id it is picked
// by is opencode/claude-opus-5 and what belongs on a pill is the rest.
func renderComboPillsLabelled(ids, labels []string, current string) string {
	var views []string

	for i, id := range ids {
		label := cells.DialLabel(ids, labels, i)
		if id == current {
			views = append(views, theme.Paint(theme.Sel).Render(" "+label+" "))
		} else {
			views = append(views, theme.Paint(theme.Dim).Render(label))
		}
	}

	return strings.Join(views, " ")
}

// Hit is what the designer holds at that cell, so a click can be answered.
func (s State) Hit(x, y int, e Env) point.Target {
	line, ok := e.Frame.BodyRow(y)
	if !ok {
		return point.Target{}
	}

	if s.showingDetail {
		// The buttons are the third row from the floor, and a click is on
		// one of them only where one is drawn.
		//
		// Both halves of that were wrong: the row was found by counting
		// back from the length of what View returned, which is the whole
		// body because it is padded to it — so the strip that answered
		// was the blank floor under a short reading and never the buttons
		// themselves. And the columns were 25 and 50, written against
		// labels in English; the pills are translated and neither number
		// had anything to do with where they end.
		foot, buttons := s.detailFoot(e.Frame.Body.H, e.Frame.Body.W, e)
		if line != e.Frame.Body.H-len(foot)+buttons {
			return point.Target{}
		}

		b, ok := detailButtonAt(x, e)
		if !ok {
			return point.Target{}
		}

		if b.field == "edit" {
			return point.Target{Kind: point.FlowItem, Field: b.field, ID: s.flowName}
		}

		return point.Target{Kind: point.FlowItem, Field: b.field}
	}

	if !s.creating {
		return s.hitList(x, line, e)
	}

	return s.hitBuilder(x, line, e)
}

// hitList is where a click landed in the list of flows.
//
// It reads the rows that were drawn, for the reason hitBuilder does: the
// list scrolls now, and a table of line numbers kept beside the draw would
// put every click one page out the moment it did.
func (s State) hitList(x, line int, e Env) point.Target {
	lines := s.flowsListLines(e.Frame.Body.W, e)
	rows := max(e.Frame.Body.H-1, 0)

	at := s.flowsListStart(lines, rows) + line
	if at < 0 || at >= len(lines) {
		return point.Target{}
	}

	row := lines[at]

	switch {
	case row.create:
		return point.Target{Kind: point.FlowItem, Field: "create"}
	case row.at == noFlow || row.at >= len(s.listed):
		return point.Target{}
	}

	d := s.listed[row.at]
	s.sel = row.at

	if row.head {
		if field := flowPill(d, x, e); field != "" {
			return point.Target{Kind: point.FlowItem, Field: field, ID: d.Name}
		}
	}

	return point.Target{Kind: point.FlowItem, Field: "details", ID: d.Name}
}

// flowPill is which of the row's own pills the pointer is over, measured off
// the pills themselves rather than written down: a translation makes every
// one of them a different width.
func flowPill(d flow.Listed, x int, e Env) string {
	p := e.Words

	at := cells.Gutter + lipgloss.Width(d.Name)
	if origin := OriginSaid(p, d.Origin); origin != "" {
		at += 2 + lipgloss.Width(origin) + 2
	}

	at += 3

	detW := lipgloss.Width("👁 "+p.T("flows.btn_view_details", "Details")) + 4
	editW := lipgloss.Width("✏ "+p.T("flows.btn_edit", "Edit")) + 4

	switch {
	case x >= at+detW && x < at+detW+editW:
		return "edit"
	case d.Origin != flow.OriginBuiltin && x >= at+detW+editW:
		return "delete"
	}

	return ""
}

// hitBuilder is where a click landed in the designer.
//
// It reads the rows that were drawn rather than a table of line numbers kept
// beside them: see flowsbuilderrows.go for why there is no second opinion
// about the layout any more.
func (s State) hitBuilder(x, line int, e Env) point.Target {
	lines, start := s.builderView(e.Frame.Body.H, e.Frame.Body.W, e)

	at := start + line
	if at < 0 || at >= len(lines) {
		return point.Target{}
	}

	row := lines[at]

	switch {
	case row.act != "":
		// A row that acts, acts where it is drawn. The whole width of the
		// window answered before, so a click on the blank forty columns
		// to the right of "✨ Draft it" sent a question to an engine.
		if x >= lipgloss.Width(row.text) {
			return point.Target{}
		}

		// The two dials of the describe tab are pills with a way to see
		// the rest beside them: pointing at one of the pills chooses it,
		// and pointing anywhere else on the row opens the list, which is
		// what the row says pressing ⏎ there does.
		if field, ok := sayDialField(row.act); ok {
			if opts, current, has := s.choices(field, e); has {
				if at, on := choiceAt(x, opts, current); on {
					return point.Target{Kind: point.FlowItem, Field: "dial", Phase: field, Pane: at}
				}
			}
		}

		return point.Target{Kind: point.FlowItem, Field: row.act}
	case row.strip:
		if at := s.flowTabAt(x, e); at >= 0 {
			return point.Target{Kind: point.FlowItem, Field: "tab", Phase: at}
		}

		return point.Target{}
	case row.pick != noPick:
		return point.Target{Kind: point.FlowItem, Field: "pick", Phase: row.pick}
	case row.phase != noPhase:
		return point.Target{Kind: point.FlowItem, Field: "select_phase", Phase: row.phase}
	case row.field == noField:
		return point.Target{}
	case row.head && row.field == flowFieldPrompt:
		if field := promptPill(x, e); field != "" {
			return point.Target{Kind: point.FlowItem, Field: field}
		}
	case row.field == flowFieldAddPhase:
		return point.Target{Kind: point.FlowItem, Field: buttonAt(x, e)}
	}

	// The phase being edited is chosen by pointing at it, the same gesture
	// the pipeline above the form already answers.
	if row.field == flowFieldPhaseSelect {
		if at, on := s.phaseTabAt(x); on {
			return point.Target{Kind: point.FlowItem, Field: "select_phase", Phase: at}
		}
	}

	// A short dial is a row of pills, and a click on one of them is that
	// option: the row's own action steps the dial one along, which is what
	// ⏎ means and not what pointing at a word does.
	if opts, current, ok := s.choices(row.field, e); ok {
		if at, on := choiceAt(x, opts, current); on {
			return point.Target{Kind: point.FlowItem, Field: "dial", Phase: row.field, Pane: at}
		}
	}

	return point.Target{Kind: point.FlowItem, Phase: row.field}
}

// sayDialField is the field one of the describe tab's two dial rows stands
// for, and whether the row is one of them at all.
func sayDialField(act string) (int, bool) {
	switch act {
	case "say_engine":
		return flowFieldSayEngine, true
	case "say_model":
		return flowFieldSayModel, true
	}

	return 0, false
}

// promptPill is which of the instruction row's three pills the pointer is
// over, or nothing when it is over the label to their left.
//
// The ranges are measured off the pills themselves rather than written down,
// because a translation makes every one of them a different width.
func promptPill(x int, e Env) string {
	p := e.Words
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
func buttonAt(x int, e Env) string {
	p := e.Words
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
