package supervisor

// What the record says happened, for the supervisor to answer from.
//
// The question this exists for is the one somebody asks coming back from
// lunch: what happened, what did you see, did it all pass. The answer is not
// a digest Orbit writes on its own — it is the supervisor's, asked in words,
// at whatever detail the reader wants. What is here is the ground it stands
// on: the lines the record actually holds, handed to the model with the
// question, so that "the tests passed" is something it read rather than
// something it inferred from a conversation.
//
// The events are handed over as they are, and not folded into a verdict.
// internal/view folds them for the screen and this package may not import
// it — but the deeper reason is that a fold is an opinion about what the
// events mean, and the whole point of this block is that everything in it is
// a fact somebody can go and check.

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// happenedCap is how many lines of the record the model is handed. Enough to
// cover a lunch's worth of runs, and short enough that the question and the
// conversation still fit beside it.
const happenedCap = 60

// happenedWindow is how far back it looks when nobody says.
const happenedWindow = 24 * time.Hour

// Happened is what the record says took place since a moment, newest first,
// one line per event that says something about the work.
//
// A task nobody has touched in the window contributes nothing, which is what
// keeps this short on a board with forty tasks on it and three of them
// moving.
func Happened(s *store.Store, since time.Time) ([]string, error) {
	if s == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}

	d, err := s.Record()
	if err != nil {
		return nil, err
	}

	ids, err := d.Tasks()
	if err != nil {
		return nil, fmt.Errorf("read the tasks: %w", err)
	}

	type line struct {
		at   time.Time
		text string
	}

	var lines []line

	for _, id := range ids {
		events, err := d.Events(id)
		if err != nil {
			return nil, fmt.Errorf("read the events of %q: %w", id, err)
		}

		for _, e := range events {
			if e.At.Before(since) || !worthTelling(e.Kind) {
				continue
			}

			lines = append(lines, line{at: e.At, text: happenedLine(id, e)})
		}
	}

	sort.SliceStable(lines, func(i, j int) bool { return lines[i].at.After(lines[j].at) })

	if len(lines) > happenedCap {
		lines = lines[:happenedCap]
	}

	// Oldest first in what is handed over: a reader — and a model — follows
	// a run forwards, and the cap took the newest.
	out := make([]string, 0, len(lines))
	for i := len(lines) - 1; i >= 0; i-- {
		out = append(out, lines[i].text)
	}

	return out, nil
}

// worthTelling is which kinds say something happened.
//
// The stream of a phase — its thinking, its tool calls — is not here. That
// is what the engine did on its way to the outcome, it is thousands of lines
// of it, and the task view is where somebody reads it. What this block
// answers is what became of the work.
func worthTelling(kind string) bool {
	switch kind {
	case record.TaskStarted, record.TaskFinished, record.TaskFailed,
		record.TaskCancelled, record.TaskRequeued, record.TaskTimedOut,
		record.TaskAbandoned, record.TaskNoted,
		record.PhaseFinished, record.PhaseFailed, record.PhaseWaiting,
		record.PhaseRetried, record.LoopChecked,
		record.GatePassed, record.GateFailed:
		return true
	}

	return false
}

// happenedLine is one event as the model is shown it: when, which task, what
// happened, and the few words that say why.
func happenedLine(id string, e record.Event) string {
	parts := []string{e.At.Format("2006-01-02 15:04"), id, e.Kind}

	if e.Phase != "" {
		parts = append(parts, e.Phase)
	}

	for _, key := range []string{"gate", "check", "exit", "attempt", "turn", "passed", "engine", "model", "cost"} {
		if v := e.Data[key]; v != "" {
			parts = append(parts, key+"="+v)
		}
	}

	if said := firstLineOf(e.Text); said != "" {
		parts = append(parts, `"`+said+`"`)
	}

	return strings.Join(parts, " · ")
}

// firstLineOf is enough of what was printed to say what it was, and no more:
// a phase's whole output is what the task view is for.
func firstLineOf(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")

	runes := []rune(line)
	if len(runes) <= 120 {
		return line
	}

	return string(runes[:120]) + "…"
}
