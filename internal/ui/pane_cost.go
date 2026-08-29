package ui

import (
	"fmt"

	"github.com/e1i0r/orbit/internal/view"
)

type costRow struct {
	phase    string
	cost     float64
	duration string
	turns    int
	engine   string
	model    string
}

// costPhaseCells is the phase column of the cost table, and costMoneyCells
// the figures beside it: wide enough for "$1234.5678" and for the way elapsed
// spells an hour.
const (
	costPhaseCells = 24
	costMoneyCells = 12
)

// costLines renders Pane 4: Cost Breakdown per phase and total.
func (m Model) costLines() []string {
	p := m.opts.Words

	t, ok := m.task(m.detail)
	if !ok {
		return []string{"  " + Paint(Dim).Render(p.T("detail.gone", "this task is no longer on the board"))}
	}

	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(p.T("cost.heading", "Cost & Resource Breakdown")),
		"  " + Paint(Dim).Render(p.T("cost.subtitle", "how much has been spent, stage by stage")),
		"",
	}

	var (
		rows    []costRow
		started view.Entry
	)

	for _, e := range m.entries {
		if e.What() == view.EntryStarted {
			started = e
			continue
		}

		if e.What() == view.EntryFinished || e.What() == view.EntryFailed || e.What() == view.EntryCancelled {
			if e.Cost > 0 || e.Phase != "" {
				eng := started.Engine
				if eng == "" {
					eng = t.Engine
				}

				mod := started.Model
				if mod == "" {
					mod = t.Model
				}

				dur := ""
				if !started.At.IsZero() && !e.At.IsZero() {
					dur = elapsed(e.At, started.At)
				}

				rows = append(rows, costRow{
					phase:    e.Phase,
					cost:     e.Cost,
					duration: dur,
					engine:   eng,
					model:    mod,
					turns:    max(e.Attempt, 1),
				})
			}
		}
	}

	if len(rows) == 0 && t.Cost == 0 {
		out = append(out, "  "+Paint(Dim).Render(p.T("cost.empty", "no cost recorded for this task")))
		return out
	}

	// The columns are padded on the plain text and painted afterwards. A
	// width verb counts the bytes of an escape sequence as characters, so a
	// rendered string padded to a column is not padded at all — which is why
	// this table has never lined up.
	out = append(out, "    "+Paint(Dim).Render(
		pad(p.T("cost.col_phase", "phase"), costPhaseCells, false)+" "+
			pad(p.T("cost.col_cost", "cost"), costMoneyCells, false)+" "+
			pad(p.T("cost.col_duration", "duration"), costMoneyCells, false)+" "+
			p.T("cost.col_engine", "engine / model")))

	for _, r := range rows {
		modStr := r.engine
		if r.model != "" {
			modStr += " (" + r.model + ")"
		}

		out = append(out, "    "+
			Paint(Accent).Render(pad(r.phase, costPhaseCells, false))+" "+
			Paint(OK).Render(pad(fmt.Sprintf("$%.4f", r.cost), costMoneyCells, false))+" "+
			Paint(Dim).Render(pad(r.duration, costMoneyCells, false))+" "+
			Paint(Dim).Render(modStr))
	}

	out = append(out,
		"",
		"    "+Paint(Dim).Render(pad(p.T("cost.total", "total so far"), costPhaseCells, false))+" "+
			Paint(Accent).Bold(true).Render(fmt.Sprintf("$%.4f", t.Cost)),
		"",
	)

	return out
}
