package db

// The supervisor thread: the conversation that belongs to no task.

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// say appends turns to the thread.
func say(t *testing.T, d *DB, events ...record.Event) {
	t.Helper()

	for _, e := range events {
		if err := d.AppendMessage(e); err != nil {
			t.Fatalf("append %s: %v", e.Kind, err)
		}
	}
}

// TestATurnComesBackTheWayItWentIn.
func TestATurnComesBackTheWayItWentIn(t *testing.T) {
	d := open(t)
	tick := clock()

	want := record.Event{
		At:   tick(),
		Kind: record.SupervisorMessage,
		Text: "retry ACME-1, the gate was flaky",
		Data: map[string]string{"by": "elio", "channel": "cli", "task_id": "ACME-1"},
	}

	if err := d.AppendMessage(want); err != nil {
		t.Fatalf("append: %v", err)
	}

	thread, err := d.Messages()
	if err != nil {
		t.Fatalf("read the thread: %v", err)
	}

	if len(thread) != 1 {
		t.Fatalf("the thread holds %d turns, want one", len(thread))
	}

	got := thread[0]
	if !got.At.Equal(want.At) || got.Kind != want.Kind || got.Text != want.Text {
		t.Errorf("read back %+v, want %+v", got, want)
	}

	if got.Data["by"] != "elio" || got.Data["channel"] != "cli" {
		t.Errorf("the data is %v, want the whole map it was given", got.Data)
	}
}

// TestATurnAboutATaskPointsAtIt. Lifting the task out of the data and into a
// column is what makes "everything the supervisor did about ACME-1" a query
// rather than a fold of the whole thread.
func TestATurnAboutATaskPointsAtIt(t *testing.T) {
	d := open(t)
	tick := clock()

	history(t, d, "ACME-1", record.Event{Kind: record.TaskCreated, Text: "Retry the webhook"})

	if err := d.AppendMessage(record.Event{
		At:   tick(),
		Kind: record.SupervisorAction,
		Text: "retried it",
		Data: map[string]string{"by": "supervisor", "channel": "mcp", "task_id": "ACME-1"},
	}); err != nil {
		t.Fatalf("append: %v", err)
	}

	var (
		who, source string
		task        sql.NullInt64
	)

	if err := d.sql.QueryRow(`SELECT who, source, task_id FROM message`).Scan(&who, &source, &task); err != nil {
		t.Fatalf("read the message row: %v", err)
	}

	if who != "supervisor" || source != "mcp" {
		t.Errorf("the turn was taken by %q on %q, want supervisor on mcp", who, source)
	}

	if !task.Valid {
		t.Error("the turn points at no task, want the one it named")
	}
}

// TestATurnAboutATaskNobodyKnowsIsStillATurn. The supervisor talks about
// tasks that were never written down — one it was asked to look for, one
// whose id somebody typed wrong — and a conversation is not a foreign key.
func TestATurnAboutATaskNobodyKnowsIsStillATurn(t *testing.T) {
	d := open(t)
	tick := clock()

	if err := d.AppendMessage(record.Event{
		At:   tick(),
		Kind: record.SupervisorMessage,
		Text: "I cannot find ACME-404",
		Data: map[string]string{"task_id": "ACME-404"},
	}); err != nil {
		t.Fatalf("append: %v", err)
	}

	var task sql.NullInt64
	if err := d.sql.QueryRow(`SELECT task_id FROM message`).Scan(&task); err != nil {
		t.Fatalf("read the message row: %v", err)
	}

	if task.Valid {
		t.Error("the turn points at a task row, want none — the task it named is not in the record")
	}

	thread, err := d.Messages()
	if err != nil {
		t.Fatalf("read the thread: %v", err)
	}

	if len(thread) != 1 || thread[0].Data["task_id"] != "ACME-404" {
		t.Errorf("the thread holds %v, want the turn with the id it named still in it", thread)
	}
}

// TestARetractionLeavesTheTurnWhereItIs. Taking back is writing, not
// erasing: a sentence somebody regretted is still a sentence they said, and
// what changes is whether it goes in front of the model again.
func TestARetractionLeavesTheTurnWhereItIs(t *testing.T) {
	d := open(t)
	tick := clock()

	said := record.Event{At: tick(), Kind: record.SupervisorMessage, Text: "cancel everything"}
	if err := d.AppendMessage(said); err != nil {
		t.Fatalf("append the turn: %v", err)
	}

	if err := d.AppendMessage(record.Event{
		At:   tick(),
		Kind: record.SupervisorRetracted,
		Data: map[string]string{"at": record.Stamp(said.At)},
	}); err != nil {
		t.Fatalf("append the retraction: %v", err)
	}

	thread, err := d.Messages()
	if err != nil {
		t.Fatalf("read the thread: %v", err)
	}

	if len(thread) != 2 || thread[0].Text != "cancel everything" {
		t.Fatalf("the thread holds %d turns, want the sentence and the taking back of it", len(thread))
	}

	// And the reader that folds retractions out of the file gets the same
	// answer here, because the retraction is a turn like any other.
	if gone := record.Retracted(thread); !gone[record.Stamp(said.At)] {
		t.Error("folding the thread does not show the turn as taken back")
	}

	var taken sql.NullString
	if err := d.sql.QueryRow(
		`SELECT retracted_at FROM message WHERE text = 'cancel everything'`,
	).Scan(&taken); err != nil {
		t.Fatalf("read the row: %v", err)
	}

	if !taken.Valid {
		t.Error("the row is not stamped as retracted, want the same answer SQL and the fold agree on")
	}
}

// TestARetractionNamingNothingChangesNothing. A retraction with no line
// named is malformed — it is written down like any other turn and takes
// nothing back.
func TestARetractionNamingNothingChangesNothing(t *testing.T) {
	d := open(t)
	tick := clock()

	say(t, d,
		record.Event{At: tick(), Kind: record.SupervisorMessage, Text: "carry on"},
		record.Event{At: tick(), Kind: record.SupervisorRetracted},
	)

	var stamped int
	if err := d.sql.QueryRow(`SELECT count(*) FROM message WHERE retracted_at IS NOT NULL`).Scan(&stamped); err != nil {
		t.Fatalf("count the retracted turns: %v", err)
	}

	if stamped != 0 {
		t.Errorf("%d turns were taken back, want none", stamped)
	}
}

// TestCountsAreWhereAMigrationGotTo. Both logs are appended to and never
// reordered, so a count is a position in them.
func TestCountsAreWhereAMigrationGotTo(t *testing.T) {
	d := open(t)
	tick := clock()

	history(t, d, "ACME-1",
		record.Event{Kind: record.TaskCreated, Text: "Retry the webhook"},
		record.Event{Kind: record.TaskStarted},
	)

	say(t, d, record.Event{At: tick(), Kind: record.SupervisorMessage, Text: "start it"})

	events, err := d.EventCount("ACME-1")
	if err != nil {
		t.Fatalf("count the events: %v", err)
	}

	if events != 2 {
		t.Errorf("the record holds %d events of the task, want two", events)
	}

	none, err := d.EventCount("ACME-404")
	if err != nil {
		t.Fatalf("count the events of a task that is not there: %v", err)
	}

	if none != 0 {
		t.Errorf("a task nobody has heard of holds %d events, want none", none)
	}

	turns, err := d.MessageCount()
	if err != nil {
		t.Fatalf("count the thread: %v", err)
	}

	if turns != 1 {
		t.Errorf("the thread holds %d turns, want one", turns)
	}
}

// TestATurnWithNoTimeIsStampedOnArrival.
func TestATurnWithNoTimeIsStampedOnArrival(t *testing.T) {
	d := open(t)

	if err := d.AppendMessage(record.Event{Kind: record.SupervisorMessage, Text: "now"}); err != nil {
		t.Fatalf("append: %v", err)
	}

	thread, err := d.Messages()
	if err != nil {
		t.Fatalf("read the thread: %v", err)
	}

	if thread[0].At.IsZero() {
		t.Error("the turn has no time, want the moment it arrived")
	}
}

// TestTheThreadOfAClosedRecordFails, for the same reason the events of one
// do: nothing is not the same answer as a record that is not open.
func TestTheThreadOfAClosedRecordFails(t *testing.T) {
	d, err := Open(t.TempDir() + "/orbit.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if _, err := d.Messages(); err == nil {
		t.Error("reading the thread of a closed record answered cleanly")
	}

	if _, err := d.MessageCount(); err == nil {
		t.Error("counting the thread of a closed record answered cleanly")
	}

	if _, err := d.EventCount("ACME-1"); err == nil {
		t.Error("counting the events of a closed record answered cleanly")
	}

	if err := d.AppendMessage(record.Event{Kind: record.SupervisorMessage, Text: "x"}); err == nil {
		t.Error("appending to a closed record answered cleanly")
	}
}

// TestAFailedTurnLeavesTheThreadAsItWas, the same rule the events follow:
// one transaction covers the turn and the row it takes back, and a write
// that fails halfway leaves neither.
func TestAFailedTurnLeavesTheThreadAsItWas(t *testing.T) {
	d := open(t)
	tick := clock()

	said := record.Event{At: tick(), Kind: record.SupervisorMessage, Text: "cancel everything"}
	say(t, d, said)

	refuse(t, d, "UPDATE", "message")

	err := d.AppendMessage(record.Event{
		At:   tick(),
		Kind: record.SupervisorRetracted,
		Data: map[string]string{"at": record.Stamp(said.At)},
	})
	if err == nil {
		t.Fatal("a retraction that could not be stamped reported success")
	}

	if !strings.Contains(err.Error(), record.SupervisorRetracted) {
		t.Errorf("the failure reads %q, want the kind named", err)
	}

	if n := count(t, d, "message"); n != 1 {
		t.Errorf("the thread holds %d turns, want only the one said before the failure", n)
	}
}

// TestATurnThatCannotBeWrittenSaysSo.
func TestATurnThatCannotBeWrittenSaysSo(t *testing.T) {
	d := open(t)

	refuse(t, d, "INSERT", "message")

	if err := d.AppendMessage(record.Event{Kind: record.SupervisorMessage, Text: "start it"}); err == nil {
		t.Error("a refused turn reported success")
	}
}

// TestAThreadWithAnUnreadableTimeIsRefused. A time nobody can read is
// damage, and the zero time in its place would put a turn in 0001.
func TestAThreadWithAnUnreadableTimeIsRefused(t *testing.T) {
	d := open(t)

	say(t, d, record.Event{Kind: record.SupervisorMessage, Text: "start it"})

	if _, err := d.sql.Exec(`UPDATE message SET at = 'yesterday'`); err != nil {
		t.Fatalf("damage the time: %v", err)
	}

	if _, err := d.Messages(); err == nil {
		t.Error("a turn with an unreadable time read cleanly, want a refusal")
	}

	if _, err := d.sql.Exec(`UPDATE message SET at = '2026-08-31T09:00:00Z', data = '{"by":'`); err != nil {
		t.Fatalf("damage the data: %v", err)
	}

	if _, err := d.Messages(); err == nil {
		t.Error("a turn with half an object in its data read cleanly, want a refusal")
	}
}
