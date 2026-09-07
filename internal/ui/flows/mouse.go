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
		rows := s.flowDetailRows(e.Frame.Body.H, e.Frame.Body.W, e)
		if line >= len(rows)-3 {
			if x < 25 {
				return point.Target{Kind: point.FlowItem, Field: "detail_select"}
			} else if x < 50 {
				return point.Target{Kind: point.FlowItem, Field: "edit", ID: s.flowName}
			}

			return point.Target{Kind: point.FlowItem, Field: "detail_back"}
		}

		return point.Target{}
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

	return point.Target{Kind: point.FlowItem, Phase: row.field}
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
