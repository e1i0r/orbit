package ui

// What the work in flight has cost so far, and the rule that keeps that
// figure honest.

import (
	"fmt"

	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// runningField is the header's chip for what is on the meter right now.
//
// Not what the board has spent since it was made — that is the status line's
// number, and it is a different question. What a reader deciding whether to
// start another run is asking is what the runs already going are costing
// them, and the answer changes while they look at it.
//
// Nothing at all when nothing is running, and nothing when what is running
// is paid for by subscription: under a plan the money left the account in
// advance, so a figure here would be arithmetic on a charge nobody made.
// What is left of the window is the honest number for those runs, and the
// quota chip beside this one is where it is drawn.
//
// Nothing either while the figure is still zero. The header at a hundred
// cells is contested space — a chip added here is a band count given up —
// and "$0.00 running" is not news: what a reader wants this for is the
// moment the number starts moving.
func (m Model) runningField(p *words.Printer) []headerField {
	spent, charged := m.runningSpend()
	if !charged || spent <= 0 {
		return nil
	}

	text := p.T("header.running_cost", "{cost} running", about("cost", fmt.Sprintf("$%.2f", spent)))

	return []headerField{{name: "running", text: Paint(Live).Render("💸 " + text)}}
}

// runningSpend is what the tasks in flight have cost so far, and whether any
// of them is charged for at all.
//
// The bool is not the total being zero: a board of subscription runs has no
// such figure at any total, and a metered board that has not been charged
// yet has one that is about to move. The caller decides which of those is
// worth a chip.
func (m Model) runningSpend() (float64, bool) {
	var (
		total   float64
		charged bool
	)

	for _, t := range m.board.Tasks {
		if view.BandOf(t) != view.Running || !m.spends(t.Engine) {
			continue
		}

		charged = true
		total += t.Cost
	}

	return total, charged
}
