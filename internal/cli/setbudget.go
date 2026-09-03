package cli

// The three settings that stop a run, or a queue, spending more.
//
// They are in a file of their own because set.go is at the size ceiling and
// because they are one idea in three lines: what a task may spend, what the
// board may spend, and — for an engine there is no bill to cap — how much of
// its window has to be left.

import (
	"errors"
	"strconv"

	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// budgetSettings are the caps, in the order a reader meets them: the task
// first, because that is the one that stops a run rather than a queue.
func budgetSettings() []Setting {
	return []Setting{{
		Name: "budget-task",
		About: func(p *words.Printer) string {
			return p.T("setting.budget_task", "the most one task may spend in dollars; 0 is no budget")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			v, err := dollars(p, value)
			if err != nil {
				return "", err
			}

			cfg.BudgetTask = v

			return value, nil
		},
		Value: func(cfg store.Settings) string { return money(cfg.BudgetTask) },
	}, {
		Name: "budget-workspace",
		About: func(p *words.Printer) string {
			return p.T("setting.budget_workspace",
				"the most the board may have spent before nothing new starts on its own; 0 is no budget")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			v, err := dollars(p, value)
			if err != nil {
				return "", err
			}

			cfg.BudgetWorkspace = v

			return value, nil
		},
		Value: func(cfg store.Settings) string { return money(cfg.BudgetWorkspace) },
	}, {
		Name: "quota-floor",
		About: func(p *words.Printer) string {
			return p.T("setting.quota_floor",
				"how much of a subscription engine's window must be left for the queue to go on, as a percentage; 0 is no floor")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			n, err := strconv.Atoi(value)
			if err != nil {
				return "", errors.New(p.T("settings.not_a_number", "{val} is not a whole number",
					words.Arg{Name: "val", Value: value}))
			}

			// A floor of a hundred would hold the queue for ever, and one
			// below zero is not a share of anything. Both are refused where
			// they are typed rather than found out by a queue that never
			// starts and says only that the floor is on.
			if n < 0 || n >= 100 {
				return "", errors.New(p.T("settings.not_a_share",
					"a quota floor is a percentage between 0 and 99; 0 is no floor at all"))
			}

			cfg.QuotaFloor = n

			return value, nil
		},
		Value: func(cfg store.Settings) string { return strconv.Itoa(cfg.QuotaFloor) },
	}}
}

// dollars reads a figure of money, and refuses what is not one.
//
// Negative is refused and zero is not: zero is how a reader turns a budget
// off, and it has to stay something they can type.
func dollars(p *words.Printer, value string) (float64, error) {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, errors.New(p.T("settings.not_an_amount", "{val} is not an amount of money",
			words.Arg{Name: "val", Value: value}))
	}

	if v < 0 {
		return 0, errors.New(p.T("settings.negative_budget",
			"a budget cannot be negative; zero is no budget at all"))
	}

	return v, nil
}

// money is a budget as `orbit set` prints one back.
func money(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
