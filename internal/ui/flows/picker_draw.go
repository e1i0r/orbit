package flows

// What the picker draws: a title, what has been typed, and the choices.

import (
	"strconv"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// pickerLines is the list as rows, each one carrying the choice it stands
// for so a click lands on that choice and not on the row under it.
func (s State) pickerLines(h, w int, e Env) []builderLine {
	p := e.Words
	ids, labels := s.pickerRows(e)

	head := map[int]string{
		flowFieldEngine: p.T("flows.pick_engine", "Pick the engine"),
		flowFieldModel:  p.T("flows.pick_model", "Pick the model"),
		flowFieldEffort: p.T("flows.pick_effort", "Pick the effort"),
	}[s.picker.field]

	out := []builderLine{
		plainLine(cells.Fit("  "+theme.Paint(theme.Accent).Bold(true).Render(head)+"  "+
			theme.Paint(theme.Dim).Render(p.T("flows.pick_count", "{n} to choose from",
				about("n", strconv.Itoa(len(ids))))), w)),
		plainLine(cells.Fit("  "+theme.Paint(theme.Dim).Render(p.T("flows.pick_filter", "type to narrow: "))+
			theme.Paint(theme.Accent).Render(s.picker.filter+"█"), w)),
		plainLine(""),
	}

	// The window follows the cursor, because the list is longer than the
	// screen and the row being chosen is the one that has to be on it.
	rows := max(h-len(out)-2, 1)
	from := max(min(s.picker.sel-rows/2, len(ids)-rows), 0)

	for i := from; i < min(from+rows, len(ids)); i++ {
		mark, ink := "  ", theme.Paint(theme.Dim)
		if i == s.picker.sel {
			mark, ink = theme.Paint(theme.Accent).Bold(true).Render("▸ "), theme.Text(theme.Primary)
		}

		line := "  " + mark + ink.Render(cells.Pad(cells.DialLabel(ids, labels, i), 34, false))
		if ids[i] == s.pickedNow(s.picker.field, e) {
			line += " " + theme.Paint(theme.Live).Render(p.T("flows.pick_current", "· in use"))
		}

		out = append(out, builderLine{text: cells.Fit(line, w), field: s.picker.field, phase: noPhase, pick: i})
	}

	return append(out, plainLine(""), plainLine(cells.Fit("  "+theme.Paint(theme.Dim).Render(p.T("flows.pick_ways",
		"[↑↓] move · [↵] choose · [esc] back")), w)))
}
