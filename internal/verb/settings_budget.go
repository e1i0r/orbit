package verb

// The settings that stop a run, or a queue, spending more.
//
// They are in a file of their own because set.go is at the size ceiling and
// because they are one idea in four lines: what a task may spend, what the
// board may spend, how much of a subscription engine's window has to be
// left — and how long a run may take, which is the same cap counted in the
// one unit that is spent whether or not anybody is charged for it.

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// budgetSettings are the caps, in the order a reader meets them: the task
// first, because that is the one that stops a run rather than a queue.
func budgetSettings() []Rule {
	return []Rule{{
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
		Clear: func(cfg *store.Settings) { cfg.BudgetTask = store.Shipped().BudgetTask },
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
		Clear: func(cfg *store.Settings) { cfg.BudgetWorkspace = store.Shipped().BudgetWorkspace },
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
		Clear: func(cfg *store.Settings) { cfg.QuotaFloor = store.Shipped().QuotaFloor },
	}, {
		Name: "run-timeout",
		About: func(p *words.Printer) string {
			return p.T("setting.run_timeout",
				"how long a run started by the window may take before it is stopped, as 45m or 2h; 0 is no limit")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			v, err := howLong(p, value)
			if err != nil {
				return "", err
			}

			cfg.RunTimeout = v

			return howLongSaid(v), nil
		},
		Value: func(cfg store.Settings) string { return howLongSaid(cfg.RunTimeout) },
		Clear: func(cfg *store.Settings) { cfg.RunTimeout = store.Shipped().RunTimeout },
	}}
}

// howLong reads a length of time as a person writes one, and answers with
// what is written down: the duration, or nothing at all for no limit.
//
// "0" is how a reader turns it off and it has to stay something they can
// type, so a bare zero — in any unit — is the empty setting rather than a
// run stopped the instant it starts. A negative one is refused: it is not a
// length of time, and read as a deadline it is a run that has already run
// out.
func howLong(p *words.Printer, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "0" {
		return "", nil
	}

	d, err := time.ParseDuration(value)
	if err != nil {
		return "", errors.New(p.T("settings.not_a_duration",
			"{val} is not a length of time; write it as 45m, 2h or 90s",
			words.Arg{Name: "val", Value: value}))
	}

	if d < 0 {
		return "", errors.New(p.T("settings.negative_duration",
			"a timeout cannot be negative; 0 is no timeout at all"))
	}

	if d == 0 {
		return "", nil
	}

	// What was typed, and not the duration printed back: "2h" read out as
	// "2h0m0s" is the same length of time said in a way nobody writes, and
	// the settings screen offers 30m, 1h and 2h as a row of pills — a
	// value spelled differently from every one of them is a dial with
	// nothing chosen on it. The same rule the budgets keep: a reader is
	// shown the cap they set.
	return value, nil
}

// howLongSaid is a duration as `orbit set` prints one back, and "0" for
// the setting nobody has turned on: a reader comparing it with what they
// typed sees the word they would type to turn it off again.
func howLongSaid(v string) string {
	if strings.TrimSpace(v) == "" {
		return "0"
	}

	return v
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
