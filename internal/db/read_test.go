package db

// What reading answers when there is nothing to read, and what it says when
// what is there is damaged.
//
// The damage below is made with SQL against the open file rather than by
// appending, because none of it can be produced by appending: that is the
// point. A record is a file on somebody's disk, and the answer to a stamp
// that will not parse has to be a refusal that names it, not a zero time
// that spreads.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestATaskNobodyHasHeardOfReadsEmpty. That is the answer the file behind
// this gave and the one every caller already handles.
func TestATaskNobodyHasHeardOfReadsEmpty(t *testing.T) {
	d := open(t)

	events, err := d.Events("ACME-404")
	if err != nil {
		t.Fatalf("read the events of a task that is not there: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("an unknown task has %d events, want none", len(events))
	}

	runs, err := d.Runs("ACME-404")
	if err != nil {
		t.Fatalf("read the runs of a task that is not there: %v", err)
	}

	if len(runs) != 0 {
		t.Errorf("an unknown task has %d runs, want none", len(runs))
	}

	ids, err := d.Tasks()
	if err != nil {
		t.Fatalf("read the tasks of an empty record: %v", err)
	}

	if len(ids) != 0 {
		t.Errorf("an empty record holds %v, want nothing", ids)
	}
}

// TestEventsComeBackInTheOrderTheyWereWritten. Not in the order of their
// clocks: two processes writing in the same millisecond have no order
// between their timestamps, and a machine whose clock steps backwards over a
// run would otherwise reorder a history that did not change.
func TestEventsComeBackInTheOrderTheyWereWritten(t *testing.T) {
	d := open(t)
	tick := clock()

	first := record.Event{At: tick(), Kind: record.PhaseToolCall, Text: "first"}
	second := record.Event{At: tick(), Kind: record.PhaseToolCall, Text: "second"}

	// The second one carries the earlier stamp, which is what a clock that
	// stepped back looks like from in here.
	first.At, second.At = second.At, first.At

	for _, e := range []record.Event{first, second} {
		if err := d.Append("ACME-1", e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	events, err := d.Events("ACME-1")
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	if events[0].Text != "first" || events[1].Text != "second" {
		t.Errorf("read back %q then %q, want them in the order they were written", events[0].Text, events[1].Text)
	}
}

// TestTasksComeBackInTheOrderTheyWereFirstSeen.
func TestTasksComeBackInTheOrderTheyWereFirstSeen(t *testing.T) {
	d := open(t)
	tick := clock()

	for _, id := range []string{"ACME-3", "ACME-1", "ACME-2"} {
		if err := d.Append(id, record.Event{At: tick(), Kind: record.TaskCreated, Text: id}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	ids, err := d.Tasks()
	if err != nil {
		t.Fatalf("read the tasks: %v", err)
	}

	if strings.Join(ids, " ") != "ACME-3 ACME-1 ACME-2" {
		t.Errorf("the tasks read back %v, want them in the order they arrived", ids)
	}
}

// TestAStampThatWillNotParseIsRefusedAndNamed. A time nobody can read is
// damage, and answering the zero time for it would put a task in 0001 on a
// board rather than telling anybody the file is wrong.
func TestAStampThatWillNotParseIsRefusedAndNamed(t *testing.T) {
	d := open(t)

	history(t, d, "ACME-1",
		record.Event{Kind: record.TaskStarted},
		phase("implement", "claude", "claude-opus-5"),
		record.Event{Kind: record.PhaseFinished},
		record.Event{Kind: record.TaskFinished},
	)

	for _, damage := range []struct {
		what string
		sql  string
	}{
		{"an event", `UPDATE event SET at = 'yesterday'`},
		{"the start of a run", `UPDATE run SET started_at = 'yesterday'`},
		{"the end of a run", `UPDATE run SET ended_at = 'yesterday'`},
		{"the start of a phase", `UPDATE phase SET started_at = 'yesterday'`},
		{"the end of a phase", `UPDATE phase SET ended_at = 'yesterday'`},
	} {
		d := open(t)

		history(t, d, "ACME-1",
			record.Event{Kind: record.TaskStarted},
			phase("implement", "claude", "claude-opus-5"),
			record.Event{Kind: record.PhaseFinished},
			record.Event{Kind: record.TaskFinished},
		)

		if _, err := d.sql.Exec(damage.sql); err != nil {
			t.Fatalf("damage %s: %v", damage.what, err)
		}

		err := readBoth(d)
		if err == nil {
			t.Errorf("%s with an unreadable time read cleanly, want a refusal", damage.what)
			continue
		}

		if !strings.Contains(err.Error(), "yesterday") {
			t.Errorf("%s failed with %q, want the unreadable value named", damage.what, err)
		}
	}
}

// readBoth asks both readers, and answers the first thing that went wrong.
func readBoth(d *DB) error {
	if _, err := d.Events("ACME-1"); err != nil {
		return err
	}

	if _, err := d.Runs("ACME-1"); err != nil {
		return err
	}

	return nil
}

// TestDataThatWillNotParseIsRefused. The column is written by encode and
// read by decode, so half a JSON object in it is a file somebody edited or a
// write that was torn.
func TestDataThatWillNotParseIsRefused(t *testing.T) {
	d := open(t)

	history(t, d, "ACME-1", record.Event{Kind: record.TaskCreated, Text: "Retry the webhook"})

	if _, err := d.sql.Exec(`UPDATE event SET data = '{"flow":'`); err != nil {
		t.Fatalf("damage the data: %v", err)
	}

	if _, err := d.Events("ACME-1"); err == nil {
		t.Error("an event with half an object in its data read cleanly, want a refusal")
	}
}

// TestAPhaseCannotPointAtARunThatIsNotThere. Every phase hangs off an
// attempt, and the key is enforced rather than assumed: a phase filed under
// nothing would be work that happened during no run.
func TestAPhaseCannotPointAtARunThatIsNotThere(t *testing.T) {
	d := open(t)

	history(t, d, "ACME-1",
		record.Event{Kind: record.TaskStarted},
		phase("implement", "claude", "claude-opus-5"),
	)

	if _, err := d.sql.Exec(
		`INSERT INTO phase(run_id, n, name, started_at) VALUES(999, 1, 'ghost', '2026-08-31T09:00:00Z')`,
	); err == nil {
		t.Error("a phase pointing at a run that is not there was accepted")
	}
}

// TestAnAttemptWithNoPhaseYetIsStillAnAttempt. There is a moment between
// task.started and the first phase.started, and it is exactly the moment
// somebody is watching the board for the task to move.
func TestAnAttemptWithNoPhaseYetIsStillAnAttempt(t *testing.T) {
	d := open(t)

	history(t, d, "ACME-1", record.Event{Kind: record.TaskStarted})

	runs, err := d.Runs("ACME-1")
	if err != nil {
		t.Fatalf("read the runs: %v", err)
	}

	if len(runs) != 1 {
		t.Fatalf("a started task has %d runs, want the one it is on", len(runs))
	}

	if len(runs[0].Phases) != 0 {
		t.Errorf("the attempt has %d phases, want none yet", len(runs[0].Phases))
	}
}

// TestReadingAClosedRecordFails. Every orbit command opens the record and
// dies with it, so a read after the close is a bug in the caller — and it
// has to say so rather than answer nothing, which reads as an empty board.
func TestReadingAClosedRecordFails(t *testing.T) {
	d, err := Open(t.TempDir() + "/orbit.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := d.Append("ACME-1", record.Event{Kind: record.TaskCreated}); err != nil {
		t.Fatalf("append: %v", err)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if _, err := d.Events("ACME-1"); err == nil {
		t.Error("reading the events of a closed record answered cleanly")
	}

	if _, err := d.Runs("ACME-1"); err == nil {
		t.Error("reading the runs of a closed record answered cleanly")
	}

	if _, err := d.Tasks(); err == nil {
		t.Error("reading the tasks of a closed record answered cleanly")
	}

	if err := d.Append("ACME-1", record.Event{Kind: record.TaskRead}); err == nil {
		t.Error("appending to a closed record answered cleanly")
	}
}
