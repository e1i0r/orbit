package db

// Sentences waiting to be told whether they were rules.
//
// One row per line of the supervisor's thread that looked like somebody
// laying down a rule. It is not knowledge and must not be: a fact is a file
// a person can read, and the model is told every fact there is — so a
// sentence nobody has agreed to yet has to wait somewhere the model cannot
// see it.

import (
	"fmt"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// What a proposal can be. A row moves out of waiting once and never back:
// the thread is append-only and so is what somebody decided about it.
const (
	// Waiting is nobody has looked at it yet.
	Waiting = "waiting"
	// Kept is a person saying yes. The fact exists by then.
	Kept = "kept"
	// Dropped is a person saying it was not a rule.
	Dropped = "dropped"
)

// A Proposal is one sentence and what became of it.
type Proposal struct {
	// SaidAt is when it was said, and it is the name of the row. The thread
	// is append-only and no two turns share an instant, so the moment is
	// what stops the same sentence being proposed twice.
	SaidAt time.Time
	Said   string
	State  string
}

// Propose writes one down, and says nothing when this line already has a row.
//
// Silence rather than an error, because the caller is a screen opening: it
// reads the whole thread every time and most of what it finds it has found
// before.
func (d *DB) Propose(p Proposal) error {
	_, err := d.sql.Exec(insertProposal, record.Stamp(p.SaidAt), p.Said, Waiting)
	if err != nil {
		return fmt.Errorf("write down what you said at %s: %w", p.SaidAt, err)
	}

	return nil
}

// Waiting is every proposal nobody has decided about, oldest first.
func (d *DB) Waiting() ([]Proposal, error) {
	rows, err := d.sql.Query(selectWaiting, Waiting)
	if err != nil {
		return nil, fmt.Errorf("read what is waiting: %w", err)
	}

	defer func() { _ = rows.Close() }() //nolint:errcheck // the read is done

	var out []Proposal

	for rows.Next() {
		var (
			p  Proposal
			at string
		)

		if err := rows.Scan(&at, &p.Said, &p.State); err != nil {
			return nil, fmt.Errorf("read a proposal: %w", err)
		}

		when, err := time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return nil, fmt.Errorf("a proposal is stamped %q, which is not a time: %w", at, err)
		}

		p.SaidAt = when
		out = append(out, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read what is waiting: %w", err)
	}

	return out, nil
}

// Decide records what somebody said about one, and refuses to move a row
// that has already been decided: two answers to one question is not a thing
// a screen should be able to produce by being clicked twice.
func (d *DB) Decide(saidAt time.Time, state string) error {
	out, err := d.sql.Exec(decideProposal, state, record.Stamp(time.Now().UTC()),
		record.Stamp(saidAt), Waiting)
	if err != nil {
		return fmt.Errorf("write down what you decided: %w", err)
	}

	moved, err := out.RowsAffected()
	if err != nil {
		return fmt.Errorf("read what was written down: %w", err)
	}

	if moved == 0 {
		return fmt.Errorf("what you said at %s has already been decided", saidAt.Format(time.RFC3339))
	}

	return nil
}
