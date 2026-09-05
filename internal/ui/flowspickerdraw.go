package ui

// What the picker draws: a title, what has been typed, and the choices.

import "strconv"

// pickerLines is the list as rows, each one carrying the choice it stands
// for so a click lands on that choice and not on the row under it.
func (m Model) pickerLines(h, w int) []builderLine {
	p := m.opts.Words
	ids, labels := m.pickerRows()

	head := map[int]string{
		flowFieldEngine: p.T("flows.pick_engine", "Pick the engine"),
		flowFieldModel:  p.T("flows.pick_model", "Pick the model"),
		flowFieldEffort: p.T("flows.pick_effort", "Pick the effort"),
	}[m.flows.picker.field]

	out := []builderLine{
		plainLine(fit("  "+Paint(Accent).Bold(true).Render(head)+"  "+
			Paint(Dim).Render(p.T("flows.pick_count", "{n} to choose from",
				about("n", strconv.Itoa(len(ids))))), w)),
		plainLine(fit("  "+Paint(Dim).Render(p.T("flows.pick_filter", "type to narrow: "))+
			Paint(Accent).Render(m.flows.picker.filter+"█"), w)),
		plainLine(""),
	}

	// The window follows the cursor, because the list is longer than the
	// screen and the row being chosen is the one that has to be on it.
	rows := max(h-len(out)-2, 1)
	from := max(min(m.flows.picker.sel-rows/2, len(ids)-rows), 0)

	for i := from; i < min(from+rows, len(ids)); i++ {
		mark, ink := "  ", Paint(Dim)
		if i == m.flows.picker.sel {
			mark, ink = Paint(Accent).Bold(true).Render("▸ "), Text(Primary)
		}

		line := "  " + mark + ink.Render(pad(dialLabel(ids, labels, i), 34, false))
		if ids[i] == m.pickedNow(m.flows.picker.field) {
			line += " " + Paint(Live).Render(p.T("flows.pick_current", "· in use"))
		}

		out = append(out, builderLine{text: fit(line, w), field: m.flows.picker.field, phase: noPhase, pick: i})
	}

	return append(out, plainLine(""), plainLine(fit("  "+Paint(Dim).Render(p.T("flows.pick_ways",
		"[↑↓] move · [↵] choose · [esc] back")), w)))
}
