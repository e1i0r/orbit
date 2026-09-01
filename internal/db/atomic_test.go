package db

// One transaction covers the event and the rows it opens or closes, and
// nothing else. What that buys is this: a write that fails halfway leaves
// the record exactly as it was, so there is never a moment where the event
// exists and the run derived from it does not.
//
// The failures below are made with a trigger that refuses one statement.
// Nothing in Orbit writes a trigger; it is the only way to fail a single
// insert inside a transaction that is otherwise fine, which is the case a
// full disk or a damaged file produces and no test could otherwise reach.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// refuse makes one kind of write on one table fail.
func refuse(t *testing.T, d *DB, verb, table string) {
	t.Helper()

	stmt := fmt.Sprintf(
		`CREATE TRIGGER refuse_%s_%s BEFORE %s ON %s BEGIN SELECT RAISE(ABORT, 'refused'); END`,
		verb, table, verb, table,
	)

	if _, err := d.sql.Exec(stmt); err != nil {
		t.Fatalf("refuse %s on %s: %v", verb, table, err)
	}
}

// count is how many rows a table holds.
func count(t *testing.T, d *DB, table string) int {
	t.Helper()

	var n int
	if err := d.sql.QueryRow(`SELECT count(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}

	return n
}

// TestAFailedWriteLeavesNothingBehind is the property the whole shape rests
// on. task.started opens a run and inserts an event; with the event refused,
// the run must not be there either — a run row nobody can point at is a task
// that looks started with nothing that says it started.
func TestAFailedWriteLeavesNothingBehind(t *testing.T) {
	d := open(t)

	history(t, d, "ACME-1", record.Event{Kind: record.TaskCreated, Text: "Retry the webhook"})
	refuse(t, d, "INSERT", "event")

	err := d.Append("ACME-1", record.Event{Kind: record.TaskStarted})
	if err == nil {
		t.Fatal("a refused insert reported success")
	}

	if !strings.Contains(err.Error(), "task.started") || !strings.Contains(err.Error(), "ACME-1") {
		t.Errorf("the failure reads %q, want the event and the task named", err)
	}

	if n := count(t, d, "run"); n != 0 {
		t.Errorf("%d runs were left behind by a write that failed, want none", n)
	}

	if n := count(t, d, "event"); n != 1 {
		t.Errorf("the record holds %d events, want the one written before the failure", n)
	}

	// And the record still answers afterwards. A rollback that left the
	// connection checked out would hang here rather than fail, which reads
	// as a database that has stopped and is not one.
	events, err := d.Events("ACME-1")
	if err != nil {
		t.Fatalf("read after a failed write: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("the record reads %d events after a failed write, want one", len(events))
	}
}

// TestEveryWriteThatCanFailSaysSo. Each of these is one statement of the
// transaction, and a write that failed and reported success is the one
// outcome that cannot be noticed later.
func TestEveryWriteThatCanFailSaysSo(t *testing.T) {
	join := record.Event{
		Kind: record.RepoJoined,
		Data: map[string]string{"path": "/src/acme", "repo": "acme"},
	}

	for _, c := range []struct {
		what   string
		verb   string
		table  string
		before []record.Event
		then   record.Event
	}{
		{"writing the task down", "INSERT", "task", nil, record.Event{Kind: record.TaskCreated, Text: "Retry"}},
		{"filling in what the task is", "UPDATE", "task", []record.Event{{Kind: record.TaskRead}}, record.Event{Kind: record.TaskCreated, Text: "Retry"}},
		{"opening a run", "INSERT", "run", nil, record.Event{Kind: record.TaskStarted}},
		{"closing a run", "UPDATE", "run", []record.Event{{Kind: record.TaskStarted}}, record.Event{Kind: record.TaskFinished}},
		{"opening a phase", "INSERT", "phase", []record.Event{{Kind: record.TaskStarted}}, phase("implement", "claude", "claude-opus-5")},
		{
			"closing a phase", "UPDATE", "phase",
			[]record.Event{{Kind: record.TaskStarted}, phase("implement", "claude", "claude-opus-5")},
			record.Event{Kind: record.PhaseFinished},
		},
		{"writing down a repository", "INSERT", "repo", nil, join},
		{"joining a repository to the task", "INSERT", "task_repo", nil, join},
	} {
		d := open(t)

		history(t, d, "ACME-1", c.before...)
		refuse(t, d, c.verb, c.table)

		if err := d.Append("ACME-1", c.then); err == nil {
			t.Errorf("%s failed and reported success", c.what)
		}
	}
}

// TestARunClosedWithItsPhaseStillOpenIsOneTransaction. Ending a run closes
// the phase the killed process left running, and that update is inside the
// same transaction as the event: if it cannot happen, none of it does.
func TestARunClosedWithItsPhaseStillOpenIsOneTransaction(t *testing.T) {
	d := open(t)

	history(t, d, "ACME-1",
		record.Event{Kind: record.TaskStarted},
		phase("implement", "claude", "claude-opus-5"),
	)

	refuse(t, d, "UPDATE", "phase")

	if err := d.Append("ACME-1", record.Event{Kind: record.TaskAbandoned}); err == nil {
		t.Fatal("closing the run reported success while the phase could not be closed")
	}

	var ended, outcome any
	if err := d.sql.QueryRow(`SELECT ended_at, outcome FROM run`).Scan(&ended, &outcome); err != nil {
		t.Fatalf("read the run: %v", err)
	}

	if ended != nil || outcome != nil {
		t.Errorf("the run reads as ended %v, %v, want it still open — its transaction did not land", ended, outcome)
	}
}
