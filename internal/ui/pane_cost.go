package ui

import (
	"fmt"

	"github.com/e1i0r/orbit/internal/view"
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
		"  " + Paint(Accent).Render(p.T("cost.heading", "Cost & Resource Breakdown")),
		fmt.Sprintf("  %s %s", Paint(Dim).Render(p.T("cost.total", "Total spent on task:")),
			Paint(Accent).Render(fmt.Sprintf("$%.4f", t.Cost))),
		"",
	}

	var hasEntries bool
	var started view.Entry
	for _, e := range m.entries {
		if e.What() == view.EntryStarted {
			started = e
			continue
		}
		if e.What() == view.EntryFinished || e.What() == view.EntryFailed || e.What() == view.EntryCancelled {
			if e.Cost > 0 || e.Phase != "" {
				hasEntries = true
				engine := started.Engine
				if engine == "" {
					engine = "claude"
				}
				model := started.Model
				if model == "" {
					model = "default"
				}
				line := fmt.Sprintf("    %-20s %-10s %-12s %s",
					Paint(Accent).Render(e.Phase),
					engine,
					model,
					Paint(OK).Render(fmt.Sprintf("$%.4f", e.Cost)))
				if e.Attempt > 0 {
					line += "  " + Paint(Dim).Render(fmt.Sprintf("(attempt %d)", e.Attempt))
				}
				out = append(out, line)
			}
		}
	}

	if !hasEntries && t.Cost == 0 {
		return []string{"", "  " + Paint(Dim).Render(p.T("cost.empty", "no cost recorded for this task"))}
	}

	return out
}
