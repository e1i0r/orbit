package ui

import (
	"fmt"

	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

type costRow struct {
	phase    string
	cost     float64
	duration string
	turns    int
	engine   string
	model    string
}

// figure is what goes in the cost column: money where money was charged,
// and the word for a plan where it was not.
//
// $0.0000 against a phase that ran for twenty minutes reads as "this one was
// free", which is the one thing it certainly was not. Under a subscription
// the money left the account in advance and none of it is attributable to a
// run; the honest column says which kind of number is missing rather than
// printing a zero that looks like one.
func figure(p *words.Printer, cost float64, priced bool) string {
	if !priced {
		return p.T("cost.subscription", "subscription")
	}

	return fmt.Sprintf("$%.4f", cost)
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

		// EntryRetried is a row like the others: an attempt a gate refused
		// ran, and was paid for. Left out, this table stops adding up to
		// the total above it, and what it hides is the spend a reader most
		// wants — what the run cost getting it wrong.
		if e.What() == view.EntryFinished || e.What() == view.EntryFailed ||
			e.What() == view.EntryCancelled || e.What() == view.EntryRetried {
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

	// Whether this engine's use is spoken about in money at all. Asked once
	// for the table rather than per row: every row of it is the same task
	// on the same engine, and asking per row would put a dollar sign on one
	// phase and not the next.
	priced := m.spends(t.Engine)

	for _, r := range rows {
		modStr := r.engine
		if r.model != "" {
			modStr += " (" + r.model + ")"
		}

		out = append(out, "    "+
			Paint(Accent).Render(pad(r.phase, costPhaseCells, false))+" "+
			Paint(OK).Render(pad(figure(p, r.cost, priced), costMoneyCells, false))+" "+
			Paint(Dim).Render(pad(r.duration, costMoneyCells, false))+" "+
			Paint(Dim).Render(modStr))
	}

	out = append(out,
		"",
		"    "+Paint(Dim).Render(pad(p.T("cost.total", "total so far"), costPhaseCells, false))+" "+
			Paint(Accent).Bold(true).Render(figure(p, t.Cost, priced)),
		"",
	)

	if !priced {
		out = append(out,
			"    "+Paint(Dim).Render(p.T("cost.window_is_the_unit",
				"under subscription · what is left is the quota window")),
			"",
		)
	}

	return out
}
