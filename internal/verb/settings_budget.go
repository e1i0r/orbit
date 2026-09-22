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
