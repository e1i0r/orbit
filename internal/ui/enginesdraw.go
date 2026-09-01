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

		if note := m.engineQuota(r); note != "" {
			text += "   " + Paint(Dim).Render(note)
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

// engineQuota is what one row of this screen says about the engine it names:
// how much of its windows is gone, or that nobody can tell.
//
// It is here and not only in the header because this is the screen where the
// choice is made. The header carries the engine already running; a reader
// standing on this list is deciding which one to hand the next task to, and
// "claude is at 77% of its week" is the fact that decides it — two lines up
// is far enough away to be worth repeating here.
//
// Only engine rows carry it. A window belongs to the engine and not to the
// model: the proxy reports what each model contributed to the window, which
// is not a limit that model has, and drawn beside opus it would read as a cap
// that does not exist. A row for an engine that is not installed carries
// nothing either — what that row is about is the setup it still needs.
func (m Model) engineQuota(r engineRow) string {
	if m.opts.Quota == nil || r.kind != rowEngine || r.disabled {
		return ""
	}

	p := m.opts.Words

	reading := m.opts.Quota(r.engine)
	if used := m.windowsUsed(reading); used != "" {
		return used
	}

	// Three ways to have no percentage, and a row that drew the same blank
	// for all of them would be the silence this screen was given a quota to
	// end. An engine paid per token has no window to be at the end of; one
	// with a source that has answered nothing is a source to go and look at
	// — a base URL a proxy does not serve /quota on reads exactly like an
	// engine with no proxy, and only one of those is worth fixing; and an
	// engine with nowhere to look at all says so, as it does on the status
	// line.
	switch {
	case reading.Money:
		return p.T("quota.per_token", "billed per token")
	case reading.Sourced:
		return p.T("quota.silent", "source answered nothing")
	default:
		return p.T("status.quota_unread", "no quota source for {engine}",
			about("engine", reading.Engine))
	}
}
