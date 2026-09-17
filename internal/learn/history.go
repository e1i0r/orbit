package learn

// What a rule has been through.
//
// The file a rule lives in says what is true today and travels with the
// checkout. This is the other half, and it stays on this machine: written
// down, reworded, moved, switched off, switched back on. A cycle that cannot
// say "this is the same thing you said differently three weeks ago" is not a
// cycle, and saying it needs both the name the rule keeps and the list of
// what happened under it.
//
// It is written from here because this is the one package that can reach
// both: internal/knowledge holds the rules and is allowed to reach nothing,
// and the command line is allowed to reach the rules but not the record.

import (
	"fmt"
	"sort"
	"time"

	"github.com/e1i0r/orbit/internal/db"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// What can happen to a rule, in the words the record writes them down in.
// They are re-stated here so that a caller which cannot reach the record —
// which is every way in — has names for them.
const (
	Written   = db.RuleKept
	Reworded  = db.RuleReworded
	MovedTo   = db.RuleMoved
	TurnedOff = db.RuleOff
	TurnedOn  = db.RuleOn
	Paused    = db.RulePaused
	Skipped   = db.RuleSkipped
	Failed    = db.RuleFailed
	Forgotten = db.RuleForgotten
)

// A Turn is one thing that happened to one rule.
type Turn struct {
	Rule string
	At   time.Time
	What string
	By   string
	// Was is what it was before, for the two turns where that is the whole
	// point: the old sentence, or the old place.
	Was string
	// Task and Phase are where it happened, for the turns that happen
	// inside a run. What makes a rule worth reconsidering is the pattern,
	// and "I always skip this in the test phase" is a pattern where "I
	// skipped it once" is not.
	Task  string
	Phase string
}

// Happened writes down one turn.
func Happened(s *store.Store, t Turn) error {
	d, err := s.Record()
	if err != nil {
		return err
	}

	return d.Happened(db.RuleTurn{
		Rule: t.Rule, At: t.At, What: t.What, By: t.By, Was: t.Was,
		Task: t.Task, Phase: t.Phase,
	})
}

// History is everything that happened to one rule, oldest first.
//
// Two sources and one story. What somebody did to the rule is written down
// here when they do it; what the rule did is already in the record, because a
// gate that refuses work writes gate.failed against the task with the rule's
// name on it. Copying those into a second table would be a write on every
// failing gate of every run, kept in two places, and wrong in one of them the
// first time something went half way.
func History(s *store.Store, rule string) ([]Turn, error) {
	if rule == "" {
		return nil, nil
	}

	d, err := s.Record()
	if err != nil {
		return nil, err
	}

	rows, err := d.RuleHistory(rule)
	if err != nil {
		return nil, err
	}

	out := make([]Turn, 0, len(rows))
	for _, row := range rows {
		out = append(out, Turn{
			Rule: row.Rule, At: row.At, What: row.What, By: row.By, Was: row.Was,
			Task: row.Task, Phase: row.Phase,
		})
	}

	stopped, err := refusals(s, rule)
	if err != nil {
		return nil, err
	}

	out = append(out, stopped...)

	sort.SliceStable(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })

	return out, nil
}

// refusals is every time this rule refused work, read off the record.
//
// Nobody is named on one: the gate ran on its own, and the task and the phase
// are the whole of where it happened.
func refusals(s *store.Store, rule string) ([]Turn, error) {
	d, err := s.Record()
	if err != nil {
		return nil, err
	}

	ids, err := d.Tasks()
	if err != nil {
		return nil, err
	}

	var out []Turn

	for _, id := range ids {
		events, err := d.Events(id)
		if err != nil {
			return nil, fmt.Errorf("read what rule %s refused in task %s: %w", rule, id, err)
		}

		for _, e := range events {
			if e.Kind != record.GateFailed || e.Data["rule"] != rule {
				continue
			}

			out = append(out, Turn{
				Rule: rule, At: e.At, What: Failed, Task: id, Phase: e.Phase,
			})
		}
	}

	return out, nil
}

// Changed writes down what one edit did to a rule: the sentence, the place,
// whether it is switched on, or any two of those at once.
//
// One row per thing that changed and not one per edit. "Reworded" and "moved
// somewhere else" are two different things to have done, they are asked
// about separately when somebody is deciding whether to keep a rule, and an
// edit that did both is an edit that did both.
//
// A rule with no name writes nothing, and that is not a failure: it is one
// somebody wrote by hand that Orbit has not written since.
//
// where is who did it and, when there was one, the run they were in the
// middle of. It is the caller's because only the caller knows: a pause typed
// at a terminal is about the rule and about no run, and the same pause taken
// while a task sat blocked is the beginning of a pattern.
func Changed(s *store.Store, was, now knowledge.Rule, where Turn) error {
	if now.ID == "" {
		return nil
	}

	where.Rule, where.At = now.ID, time.Now().UTC()

	if was.Phrase != now.Phrase {
		one := where
		one.What, one.Was = db.RuleReworded, was.Phrase

		if err := Happened(s, one); err != nil {
			return err
		}
	}

	if was.Scope != now.Scope {
		one := where
		one.What, one.Was = db.RuleMoved, about(was.Scope)

		if err := Happened(s, one); err != nil {
			return err
		}
	}

	if was.State != now.State {
		one := where
		one.What, one.Was = Stood(now.State), now.Why

		if err := Happened(s, one); err != nil {
			return err
		}
	}

	return nil
}

// Stood is what a rule moving to one state is called in the record.
//
// Moving back to applying is one thing however it got away from it: a rule
// that was paused and one that was switched off both come back the same way,
// and which it was is the row before this one.
func Stood(state knowledge.State) string {
	switch state {
	case knowledge.Paused:
		return db.RulePaused
	case knowledge.Off:
		return db.RuleOff
	default:
		return db.RuleOn
	}
}

// about is a scope in the words the history writes it down in, which are the
// words the rest of Orbit says it in: the place, or how wide it is when it is
// not a place at all.
func about(sc knowledge.Scope) string {
	switch sc.Kind {
	case knowledge.General:
		return "everywhere"
	case knowledge.Language:
		return sc.Lang
	case knowledge.Repo:
		return "the whole checkout"
	case knowledge.Symbol:
		return sc.Path + "#" + sc.Symbol
	default:
		return sc.Path
	}
}
