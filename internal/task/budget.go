package task

// What a task has spent, and the cap that stops it spending more.

import (
	"fmt"
	"strconv"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// overBudget reports whether the task has already spent what it was allowed,
// and how much that was.
//
// It is asked between phases and not inside one. A phase that has started is
// a process that is running and an engine that is already being charged for;
// stopping it there would throw away work that has been paid for and leave a
// worktree half written. Between phases is where a run can stop having
// spent nothing more than it already had.
//
// A budget nobody set is zero, which is no budget: every field of the
// settings file has a working zero, and for a cap the working zero is that
// there is none.
//
// A settings file that will not read is not a budget of zero and not a run
// that stops. Store.Settings answers defaults for a file it cannot parse,
// which is a budget of none — the same answer a fresh install gets, and the
// one that does not stop a run over a file somebody mistyped.
func overBudget(s *store.Store, t Task) (spent, budget float64, over bool) {
	cfg, err := s.Settings()
	if err != nil || cfg.BudgetTask <= 0 {
		return 0, 0, false
	}

	events, err := Events(s, t)
	if err != nil {
		// Best-effort, like the session lookup beside it: a record that
		// will not read is a run that goes on rather than a run that stops
		// over a number nobody can check. What it costs is the cap; what
		// refusing would cost is the work.
		return 0, cfg.BudgetTask, false
	}

	spent = spentOn(events)

	return spent, cfg.BudgetTask, spent >= cfg.BudgetTask
}

// spentOn is what the record says a task has paid, over every attempt.
//
// Every event that ends a phase carries what that phase spent, and all four
// of them count: a phase a gate refused and a phase somebody cancelled were
// charged for exactly like one that finished. Per task and not per run,
// because the budget is the task's — a second attempt is more money on the
// same piece of work.
func spentOn(events []record.Event) float64 {
	var total float64

	for _, e := range events {
		switch e.Kind {
		case record.PhaseFinished, record.PhaseFailed, record.PhaseCancelled, record.PhaseRetried:
			if cost, err := strconv.ParseFloat(e.Data["cost"], 64); err == nil {
				total += cost
			}
		}
	}

	return total
}

// stopSpending ends a run that has spent its budget, and says so in the two
// figures a reader compares.
//
// It is not a failure and not a task that got stuck: nothing was wrong with
// the work, and the run stopped because of a number somebody chose. The
// event says which phase did not run, because that is what a reader deciding
// whether to raise the cap needs — how far it got, not how far it did not.
func stopSpending(s *store.Store, t Task, p flow.Phase, spent, budget float64) error {
	text := fmt.Sprintf("Spent %s of the %s this task was allowed, so phase %q did not run.",
		money(spent), money(budget), p.Name)

	_ = emit(s, t, record.Event{ //nolint:errcheck // best-effort: the run is ending either way
		Kind: record.TaskOverBudget,
		Text: text,
		Data: map[string]string{
			"spent":  money(spent),
			"budget": money(budget),
			"phase":  p.Name,
		},
	})

	return fmt.Errorf("task %s: %s", t.ID, text)
}

// money is a figure as the record spells one: the shortest form that reads
// back as the same number, which is how phase.finished writes a cost.
func money(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
