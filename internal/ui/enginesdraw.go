package ui

import (
	"fmt"
	"strings"
)

func (m Model) enginesRows(h, w int) []string {
	if h <= 0 {
		return nil
	}
	p := m.opts.Words

	if m.engines.showingSetup {
		out := []string{
			"",
			"  " + Paint(Accent).Render(p.T("engines.setup_title", "Setup Steps for {engine}",
				about("engine", m.engines.setupEngine))),
			"",
		}
		rows := m.collectEngineRows()
		for _, r := range rows {
			if r.engine == m.engines.setupEngine && len(r.setup) > 0 {
				for _, step := range r.setup {
					out = append(out, "    "+step)
				}
			}
		}
		out = append(out,
			"",
			"  "+Paint(Dim).Render(p.T("engines.setup_notice", "Orbit verifies setup steps but executes nothing.")),
			"",
			"  "+Paint(Dim).Render(p.T("engines.setup_back", "{back} back", about("back", m.keys.Back.Help().Key))),
		)
		return fill(out, h)
	}

	out := []string{
		"",
		"  " + Paint(Accent).Render(p.T("engines.title", "Engine & Model Knobs")),
		"  " + Paint(Dim).Render(p.T("engines.subtitle", "choose model, effort and thinking for this run")),
		"",
	}

	rows := m.collectEngineRows()
	idxs := m.selectableEngineIndices(rows)
	currentSelectable := -1
	if m.engines.sel >= 0 && m.engines.sel < len(idxs) {
		currentSelectable = idxs[m.engines.sel]
	}

	for i, r := range rows {
		if r.kind == rowHeader {
			out = append(out, "", "  "+Paint(Accent).Render(r.title))
			continue
		}
		mark := strings.Repeat(" ", gutter)
		if i == currentSelectable {
			mark = markGlyph + strings.Repeat(" ", gutter-1)
		}
		text := r.title
		if r.selected {
			text += " " + Paint(OK).Render("●")
		}
		if r.disabled {
			text = Paint(Dim).Render(text)
		}
		line := fmt.Sprintf("%s%s", mark, text)
		out = append(out, fit(line, w))
	}

	waysOut := p.T("engines.ways_out", "{open} select · {up_down} move · {back} back",
		about("open", m.keys.Open.Help().Key),
		about("up_down", m.keys.Up.Help().Key+m.keys.Down.Help().Key),
		about("back", m.keys.Back.Help().Key))
	out = append(out, "", fit("  "+Paint(Dim).Render(waysOut), w))
	return fill(out, h)
}

func (m Model) hitEngines(x, y int) Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return Target{}
	}
	rows := m.collectEngineRows()
	lineIdx := 4
	sIdx := 0
	for _, r := range rows {
		if r.kind == rowHeader {
			lineIdx += 2
			continue
		}
		if line == lineIdx {
			return Target{Kind: TargetEngineRow, Pane: sIdx}
		}
		lineIdx++
		sIdx++
	}
	return Target{}
}
