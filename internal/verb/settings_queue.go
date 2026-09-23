package verb

// The settings of the queue: how many runs go at once, and how full memory
// may be for another to start.

import (
	"errors"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// queueSettings are the two limits the queue keeps.
func queueSettings() []Rule {
	return []Rule{{
		Name: "max-running",
		About: func(p *words.Printer) string {
			return p.T("setting.max_running",
				"how many runs go at once; the rest wait in the queue and start as one finishes")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			n, err := wholeNumber(p, value)
			if err != nil {
				return "", err
			}

			if n < 1 {
				return "", errors.New(p.T("settings.not_a_slot",
					"at least one run has to be able to go, or nothing ever starts"))
			}

			cfg.MaxRunning = n

			return strconv.Itoa(n), nil
		},
		Value: func(cfg store.Settings) string { return strconv.Itoa(cfg.Running()) },
		Clear: func(cfg *store.Settings) { cfg.MaxRunning = store.Shipped().MaxRunning },
	}, {
		Name: "memory-ceiling",
		About: func(p *words.Printer) string {
			return p.T("setting.memory_ceiling",
				"how full the machine's memory may be, as a percentage, for the queue to "+
					"start another run")
		},
		Set: func(p *words.Printer, cfg *store.Settings, value string) (string, error) {
			n, err := wholeNumber(p, value)
			if err != nil {
				return "", err
			}

			// Under half is a machine that would never start anything with
			// a browser open, and a hundred is no ceiling at all.
			if n < 50 || n > 99 {
				return "", errors.New(p.T("settings.not_a_ceiling",
					"a memory ceiling is a percentage between 50 and 99"))
			}

			cfg.MemoryCeiling = n

			return strconv.Itoa(n), nil
		},
		Value: func(cfg store.Settings) string { return strconv.Itoa(cfg.MemoryBar()) },
		Clear: func(cfg *store.Settings) { cfg.MemoryCeiling = store.Shipped().MemoryCeiling },
	}}
}

// wholeNumber is value read as one, or the sentence saying it is not.
func wholeNumber(p *words.Printer, value string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, errors.New(p.T("settings.not_a_number", "{val} is not a whole number",
			words.Arg{Name: "val", Value: value}))
	}

	return n, nil
}
