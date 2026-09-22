package verb

// The three settings that stop a run, or a queue, spending more.
//
// They are in a file of their own because set.go is at the size ceiling and
// because they are one idea in three lines: what a task may spend, what the
// board may spend, and — for an engine there is no bill to cap — how much of
// its window has to be left.

import (
	"errors"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/env"
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
		Name: "decisions",
		About: func(p *words.Printer) string {
			return p.T("setting.decisions",
				"whether a decision engine answers for the supervisor: off, shadow (it writes down what "+
					"it would have said), or on")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			value = strings.TrimSpace(strings.ToLower(value))

			switch value {
			case store.DecisionsOff, store.DecisionsShadow, store.DecisionsOn:
				cfg.Decisions = value

				return value, nil
			}

			return "", errors.New(p.T("settings.not_a_decision_mode",
				"{val} is not one of off, shadow or on", words.Arg{Name: "val", Value: value}))
		},
		Value: func(cfg store.Settings) string { return cfg.Deciding() },
		Clear: func(cfg *store.Settings) { cfg.Decisions = store.Shipped().Decisions },
	}, {
		Name: "decision-floor",
		About: func(p *words.Printer) string {
			return p.T("setting.decision_floor",
				"how sure the decision engine has to be before Orbit acts on what it says, as a "+
					"percentage; below it the run waits for you")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return "", errors.New(p.T("settings.not_a_number", "{val} is not a whole number",
					words.Arg{Name: "val", Value: value}))
			}

			// Under fifty is a coin toss with an opinion, and a hundred is
			// a floor nothing clears. Both are refused where they are
			// typed rather than found out by a supervisor that acts on
			// everything, or on nothing.
			if n < 50 || n > 99 {
				return "", errors.New(p.T("settings.not_a_bar",
					"a decision floor is a percentage between 50 and 99; below fifty is a coin toss"))
			}

			cfg.DecisionFloor = n

			return value, nil
		},
		Value: func(cfg store.Settings) string { return strconv.Itoa(cfg.DecisionBar()) },
		Clear: func(cfg *store.Settings) { cfg.DecisionFloor = store.Shipped().DecisionFloor },
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

// caveat is what is said after a setting has been written, when writing it
// was not enough to make it work. Empty when there is nothing to add, which
// is every setting but one.
//
// `decisions` is the only setting with a half that does not live in the
// settings file. The key is in the environment on purpose — `orbit
// settings` prints its table to a terminal, to a screen and into a chat,
// and a secret that can be printed is a secret that will be — so the two
// halves get written in different places, on different days, and one of
// them can be forgotten.
//
// Forgetting it is silent. The setting takes the value, the table reads
// back "on", and every gate goes on waiting for a person exactly as it did
// before; a whole task ran that way before this line existed. The moment
// somebody types the half that is a setting is the moment they still have
// the other half in mind, so it is said here as well as in the cockpit's
// status line and in the log the run writes.
func caveat(p *words.Printer, key, value string) string {
	if key != "decisions" || value == store.DecisionsOff || env.Set(env.DecisionKey) {
		return ""
	}

	return p.T("set.decisions_unkeyed",
		"there is no {key} in the environment, so every gate still waits for you",
		words.Arg{Name: "key", Value: env.DecisionKey})
}
