package db

// What happened to a rule.
//
// A rule is a file in a checkout and says what is true today. This is the
// other half: what it has been through — written down, reworded, moved,
// switched off, switched back on. That belongs here rather than in the file,
// because it is of this machine, it only ever grows, and a history kept in
// the file would put a diff in somebody's repository every time it changed.

import (
	"fmt"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// What can happen to a rule. A row is added and never changed: what happened
// happened, and correcting it afterwards is another row.
const (
	// RuleKept is somebody agreeing with it, which is where every rule
	// starts.
	RuleKept = "kept"
	// RuleReworded is the sentence changed. Was carries what it said
	// before, which is the whole reason this row is worth keeping.
	RuleReworded = "reworded"
	// RuleMoved is the same rule about somewhere else. Was carries where it
	// was about before.
	RuleMoved = "moved"
	// RulePaused is somebody stopping it for now, with the reason they
	// gave in Was. RuleOff is them disagreeing with it, and RuleOn is it
	// applying again after either. None of them deletes anything.
	RulePaused = "paused"
	RuleOff    = "off"
	RuleOn     = "on"
	// RuleSkipped is somebody getting past it this once, which leaves it
	// applying. RuleFailed is the rule doing its job: a gate refusing work
	// over it.
	//
	// A gate that passed is not written down. A rule that works is silent
	// and a rule that is in the way is not, so what is worth keeping is the
	// friction — and a row per passing gate per phase per run would bury it
	// under what nobody needs to read.
	RuleSkipped = "skipped"
	RuleFailed  = "failed"
)

// A RuleTurn is one thing that happened to one rule.
type RuleTurn struct {
	// Rule is the rule's own name, the one that survives its sentence
	// changing.
	Rule string
	At   time.Time
	What string
	// By is who did it, in the same words the rest of the record uses:
	// "operator" for a person at any of the controls, a channel's name, a
	// model's.
	By string
	// Was is what it was before, for the two turns where that is the point:
	// the old sentence, or the old place. Empty for the rest.
	Was string
	// Task and Phase are where it happened, and empty for a turn somebody
	// took from a terminal — which is about the rule and not about any run.
	Task  string
	Phase string
}

// Happened writes down one turn, and says nothing about a rule with no name.
//
// Silence rather than an error, because a rule with no name is one somebody
// wrote by hand and Orbit has not written since — which is allowed, and
// which is not a reason to refuse the edit that was actually asked for.
func (d *DB) Happened(t RuleTurn) error {
	if t.Rule == "" {
		return nil
	}

	if t.At.IsZero() {
		t.At = time.Now().UTC()
	}

	_, err := d.sql.Exec(insertRuleTurn, t.Rule, record.Stamp(t.At), t.What, t.By, t.Was,
		t.Task, t.Phase)
	if err != nil {
		return fmt.Errorf("write down what happened to rule %s: %w", t.Rule, err)
	}

	return nil
}

// RuleHistory is everything that happened to one rule, oldest first.
func (d *DB) RuleHistory(rule string) ([]RuleTurn, error) {
	rows, err := d.sql.Query(selectRuleTurns, rule)
	if err != nil {
		return nil, fmt.Errorf("read what happened to rule %s: %w", rule, err)
	}

	defer func() { _ = rows.Close() }() //nolint:errcheck // the read is done

	var out []RuleTurn

	for rows.Next() {
		var (
			t  RuleTurn
			at string
		)

		if err := rows.Scan(&t.Rule, &at, &t.What, &t.By, &t.Was, &t.Task, &t.Phase); err != nil {
			return nil, fmt.Errorf("read a turn of rule %s: %w", rule, err)
		}

		when, err := time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return nil, fmt.Errorf("a turn of rule %s is stamped %q, which is not a time: %w",
				rule, at, err)
		}

		t.At = when
		out = append(out, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read what happened to rule %s: %w", rule, err)
	}

	return out, nil
}
