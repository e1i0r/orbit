package db

// The table a sentence waits in, and the columns it gained along the way.
//
// Apart from the schema beside it so that adding it to a record already in
// use and creating a fresh one are the same text.

import (
	"database/sql"
	"fmt"
)

// proposals is where a sentence waits between Orbit noticing it and a person
// saying whether it was a rule.
//
// It is apart from the schema above so that adding it to a record already in
// use and creating a fresh one are the same text.
//
// Nothing here is knowledge. A fact lives in a file a person can read and
// edit, and putting a proposal there would mean the model is told about it
// before anybody agreed to it. This table holds exactly the in-between.
//
// said_at is what the line is known by. The thread is append-only and no two
// turns share an instant, so the moment it was said is its name — and that
// is what keeps this from proposing the same sentence twice.
const proposals = `
CREATE TABLE IF NOT EXISTS proposal(
  id         INTEGER PRIMARY KEY,
  said_at    TEXT NOT NULL UNIQUE,
  said       TEXT NOT NULL,
  state      TEXT NOT NULL DEFAULT 'waiting',
  decided    TEXT,
  said_by    TEXT NOT NULL DEFAULT 'operator',
  about_task TEXT NOT NULL DEFAULT '',
  repo       TEXT NOT NULL DEFAULT '',
  path       TEXT NOT NULL DEFAULT '',
  topic      TEXT NOT NULL DEFAULT '',
  habit      TEXT NOT NULL DEFAULT '',
  gate       TEXT NOT NULL DEFAULT ''
);
`

// proposalColumns is the three a proposal gained when a sentence could come
// from somewhere other than the supervisor's thread: who said it, which task
// it came out of, and the checkout it is about.
const proposalColumns = `
ALTER TABLE proposal ADD COLUMN said_by    TEXT NOT NULL DEFAULT 'operator';
ALTER TABLE proposal ADD COLUMN about_task TEXT NOT NULL DEFAULT '';
ALTER TABLE proposal ADD COLUMN repo       TEXT NOT NULL DEFAULT '';
`

// proposalPathColumn is the folder the work was in when the sentence was
// said. It came later than the three above, so a record that has those and
// not this one is a record somebody was already using.
const proposalPathColumn = `
ALTER TABLE proposal ADD COLUMN path TEXT NOT NULL DEFAULT '';
`

// proposalHabitColumns are the two a proposal gained when a rule could be
// drawn from what somebody keeps saying rather than from one sentence: the
// kind of thing it is about, and which habit it was drawn from.
//
// The second is what stops a rule somebody dropped being offered again the
// next time the same sentences are read.
const proposalHabitColumns = `
ALTER TABLE proposal ADD COLUMN topic TEXT NOT NULL DEFAULT '';
ALTER TABLE proposal ADD COLUMN habit TEXT NOT NULL DEFAULT '';
`

// proposalGateColumn is the command a sentence arrived with.
//
// One source can bring it: what the checkout already refuses work over. A
// rule out of a workflow is a command that has been running for years, and
// the sentence is only a way of saying what it does — so the command travels
// with the sentence rather than being invented again by whoever keeps it.
//
// It is called gate and not check because that is what it makes the rule:
// the column holds what the rule would refuse work with, and a proposal is
// not a rule yet.
const proposalGateColumn = `
ALTER TABLE proposal ADD COLUMN gate TEXT NOT NULL DEFAULT '';
`

// hasProposalColumn asks whether the table already has one of them.
const hasProposalColumn = `SELECT count(*) FROM pragma_table_info('proposal') WHERE name = ?`

// widenProposals adds the columns a proposal gained to a table that does not
// have them yet.
//
// One question per group, because they arrived at different times: the three
// when a sentence could come from somewhere other than the supervisor's
// thread, the path when a rule started arriving knowing where the work was,
// the two when a rule could be drawn from a habit, and the gate when a
// sentence could arrive with the command already behind it. A record holding
// any prefix of those is a record somebody was using in between.
func widenProposals(tx *sql.Tx) error {
	if err := widenProposal(tx, "said_by", proposalColumns); err != nil {
		return err
	}

	if err := widenProposal(tx, "path", proposalPathColumn); err != nil {
		return err
	}

	if err := widenProposal(tx, "habit", proposalHabitColumns); err != nil {
		return err
	}

	return widenProposal(tx, "gate", proposalGateColumn)
}

// widenProposal runs one group of columns when the one that names the group
// is not there yet.
//
// It asks the table about its own shape because SQLite has no IF NOT EXISTS
// for a column, and because the version number is not an answer: a 2 written
// by a branch exploring something else means a different record than this 2
// does.
func widenProposal(tx *sql.Tx, named, add string) error {
	var there int

	if err := tx.QueryRow(hasProposalColumn, named).Scan(&there); err != nil {
		return fmt.Errorf("read the shape of the proposal table: %w", err)
	}

	if there > 0 {
		return nil
	}

	if _, err := tx.Exec(add); err != nil {
		return fmt.Errorf("widen the proposal table: %w", err)
	}

	return nil
}

// rules is what happened to a rule, one row per thing that happened.
//
// It is here and not in the file the rule lives in. The file says what is
// true today and travels with the checkout; what a rule has been through is
// of this machine, is append-only, and grows on every run — a history kept
// in the file would put a diff in somebody's repository every time a gate
// ran.
//
// rule_id is the rule's own name and not a row of anything: the rule may be
// in a checkout this record has never seen, and a foreign key to a table
// that cannot hold it would mean the history of a rule somebody cloned in is
// unwritable.
const rules = `
CREATE TABLE IF NOT EXISTS rule(
  id      INTEGER PRIMARY KEY,
  rule_id TEXT NOT NULL,
  at      TEXT NOT NULL,
  what    TEXT NOT NULL,
  said_by TEXT NOT NULL DEFAULT '',
  was     TEXT NOT NULL DEFAULT '',
  task_id TEXT NOT NULL DEFAULT '',
  phase   TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS rule_by_name ON rule(rule_id, id);
`

// ruleWhereColumns are where a turn happened: the task and the phase.
//
// Empty for a turn somebody took from a terminal, which is about the rule and
// not about any run. Filled for the two that happen inside one — a gate
// refusing work, and somebody skipping past it — because what makes a rule
// worth reconsidering is the pattern, and "I always skip this in the test
// phase" is a pattern where "I skipped it once" is not.
const ruleWhereColumns = `
ALTER TABLE rule ADD COLUMN task_id TEXT NOT NULL DEFAULT '';
ALTER TABLE rule ADD COLUMN phase   TEXT NOT NULL DEFAULT '';
`

// hasRuleColumn asks whether the rule table already has one.
const hasRuleColumn = `SELECT count(*) FROM pragma_table_info('rule') WHERE name = ?`

// widenRules adds where a turn happened to a table that does not say.
func widenRules(tx *sql.Tx) error {
	var there int

	if err := tx.QueryRow(hasRuleColumn, "task_id").Scan(&there); err != nil {
		return fmt.Errorf("read the shape of the rule table: %w", err)
	}

	if there > 0 {
		return nil
	}

	if _, err := tx.Exec(ruleWhereColumns); err != nil {
		return fmt.Errorf("widen the rule table: %w", err)
	}

	return nil
}
