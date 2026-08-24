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

	var rows []costRow
	var started view.Entry
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

	// Table header
	out = append(out, fmt.Sprintf("    %-24s %-12s %-12s %s",
		Paint(Dim).Render("etapa"),
		Paint(Dim).Render("costo"),
		Paint(Dim).Render("duración"),
		Paint(Dim).Render("motor / modelo"),
	))

	for _, r := range rows {
		modStr := r.engine
		if r.model != "" {
			modStr += " (" + r.model + ")"
		}
		out = append(out, fmt.Sprintf("    %-24s %-12s %-12s %s",
			Paint(Accent).Render(r.phase),
			Paint(OK).Render(fmt.Sprintf("$%.4f", r.cost)),
			Paint(Dim).Render(r.duration),
			Paint(Dim).Render(modStr),
		))
	}
	out = append(out, "")

	// Budget and totals box
	out = append(out,
		fmt.Sprintf("    %-24s %s", Paint(Dim).Render("total acumulado"), Paint(Accent).Bold(true).Render(fmt.Sprintf("$%.4f", t.Cost))),
		fmt.Sprintf("    %-24s %s", Paint(Dim).Render("presupuesto tarea"), Paint(Dim).Render("$25.00")),
		fmt.Sprintf("    %-24s %s", Paint(Dim).Render("presupuesto por etapa"), Paint(Dim).Render("$5.00")),
		"",
	)

	return out
}
