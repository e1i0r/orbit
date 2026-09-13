package cli

// The three doors the window's tray answers through: what you said that
// nobody has decided about, keeping one, and saying it was not a rule.
//
// Here rather than in internal/ui for the reason every other port is: the
// window writes nothing. It says what the operator meant and something with
// the state root does it. See the layer table in internal/arch.

import (
	"time"

	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/ui/known"
)

// waitingPort is the tray, oldest first.
//
// A failure is logged and answered as nothing. The tray is a courtesy on a
// screen that is about something else, and a screen that refused to draw
// because a table could not be read would be the worse answer.
func waitingPort(s *store.Store) func() []known.Said {
	return func() []known.Said {
		rows, err := learn.Waiting(s)
		if err != nil {
			logger.Error("cli/rules", "what you said was not read back: %v", err)

			return nil
		}

		out := make([]known.Said, 0, len(rows))
		for _, row := range rows {
			out = append(out, known.Said{At: row.At, Text: row.Text, From: row.From()})
		}

		return out
	}
}

// keepRulePort writes one of them down as a fact of yours, in the place the
// screen was left with.
//
// repo is the checkout the window was opened over, and it is what a sentence
// said to the supervisor has to go on: the supervisor is about the board
// rather than about one task, so the sentence itself knows no repository.
// A rule that came out of a task uses that task's own, which the tray
// carries.
func keepRulePort(
	s *store.Store, repo string,
) func(at time.Time, phrase, check, where string) error {
	return func(at time.Time, phrase, check, where string) error {
		if err := learn.Keep(s, at, phrase, check, learn.Place{Repo: repo, Path: where}); err != nil {
			return err
		}

		logger.Info("cli/rules", "kept what you said at %s: %q", at.Format(time.RFC3339), phrase)

		return nil
	}
}

// dropRulePort says it was not a rule.
func dropRulePort(s *store.Store) func(at time.Time) error {
	return func(at time.Time) error {
		if err := learn.Drop(s, at); err != nil {
			return err
		}

		logger.Info("cli/rules", "what you said at %s was not a rule", at.Format(time.RFC3339))

		return nil
	}
}
