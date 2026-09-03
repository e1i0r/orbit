package ui

// The workspace brake: what stops the queue picking up another task when
// the money, or the window the money bought, is nearly gone.

import (
	"fmt"

	"github.com/e1i0r/orbit/internal/words"
)

// brake is one reason the queue is holding, and the chip that says so.
type brake struct {
	key  string // what the header calls it, for a test and for a reader
	text string
}

// workspaceBrake is why nothing new starts, or the zero brake when nothing
// is holding it.
//
// Two numbers and one idea. An engine that charges per token is stopped by
// money, because money is what its runs cost; an engine on a subscription is
// stopped by what is left of its window, because there is no bill to cap and
// the window is the whole of what running out means. They are the same brake
// in the unit that engine is paid in, which is the decision internal/quota
// settled and this only reads.
//
// It holds the queue and nothing else. A run already going is not stopped
// half way — the money for it is spent and the worktree is written — and a
// reader who starts a task by hand is a reader making a decision, which is
// what the whole cockpit is for. What a brake stops is the part that happens
// while nobody is looking.
func (m Model) workspaceBrake(p *words.Printer) brake {
	if m.opts.Settings == nil {
		return brake{}
	}

	if budget := m.opts.Settings.BudgetWorkspace(); budget > 0 {
		if spent := m.spentOnBoard(); spent >= budget {
			return brake{key: "budget", text: p.T("header.budget_brake", "budget spent (${spent})",
				about("spent", fmt.Sprintf("%.2f", spent)))}
		}
	}

	floor := m.opts.Settings.QuotaFloor()
	if floor <= 0 || m.opts.Quota == nil {
		return brake{}
	}

	reading := m.opts.Quota(m.dialEngine(""))
	// An engine paid per token has no window to run out of, and one whose
	// window nobody can read has nothing to compare against a floor. Neither
	// is an engine that is nearly out: a brake that could not tell those
	// apart would hold the queue for every engine Orbit cannot see.
	if reading.Money {
		return brake{}
	}

	for _, w := range reading.Windows {
		if left := 100 - pctUsed(w); left < float64(floor) {
			return brake{key: "quota", text: p.T("header.quota_brake", "quota floor ({left}% left)",
				about("left", fmt.Sprintf("%.0f", left)))}
		}
	}

	return brake{}
}

// brakeField is the header's chip for a queue the money is holding, and
// nothing at all when it is not.
//
// It is drawn beside the unread brake and in the same shape, because they
// are three ways of saying one thing — nothing new is starting — and a
// reader who has learned to look for the warning triangle should not have to
// learn a second place to look for it.
func (m Model) brakeField(p *words.Printer) []headerField {
	b := m.workspaceBrake(p)
	if b.key == "" {
		return nil
	}

	return []headerField{{text: Paint(Warn).Render("⚠️ " + b.text)}}
}

// spentOnBoard is what the tasks on this board have cost between them.
//
// The board and not a session, because the board is what a reader is looking
// at: a number that reset when the window was reopened would be a budget
// somebody could spend twice by restarting.
func (m Model) spentOnBoard() float64 {
	var total float64

	for _, t := range m.board.Tasks {
		total += t.Cost
	}

	return total
}
