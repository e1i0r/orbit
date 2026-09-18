package board

// Following the record by row: what one refresh reads, what a task the
// enumeration has not reached yet does with what was written about it, and
// the one rule the whole design rests on — an event reaches a task once.

import (
	"slices"
	"strings"
	"testing"
)

// TestOneQueryServesEveryTaskOnTheBoard is what the byte offsets used to buy,
// bought again: a refresh reads what was written, for every task at once, and
// the reader stops at the last row it saw rather than at a length per file.
func TestOneQueryServesEveryTaskOnTheBoard(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent())
	addTask(t, s, repoPath, "ACME-2", created("Fix the swagger lint"), startedEvent())

	r := NewReader(s, work)
	refresh(t, r)

	baseline := r.at
	if baseline == 0 {
		t.Fatal("after a refresh over four events the reader has read up to row zero")
	}

	appendTo(t, s, repoPath, "ACME-1", failedEvent())
	appendTo(t, s, repoPath, "ACME-2", finishedEvent())

	b, changed := refresh(t, r)
	if b.Health.EventsRead != 2 {
		t.Errorf("the refresh read %d events, want the 2 that were written", b.Health.EventsRead)
	}

	if !slices.Equal(changed.Tasks, []string{"ACME-1", "ACME-2"}) {
		t.Errorf("Changed.Tasks = %v, want both tasks", changed.Tasks)
	}

	if r.at <= baseline {
		t.Errorf("the reader is still at row %d after two events were written", r.at)
	}

	for _, st := range r.tasks {
		if len(st.events) != 3 {
			t.Errorf("task %s holds %d events, want 3", st.id, len(st.events))
		}
	}

	if _, changed := refresh(t, r); len(changed.Tasks) != 0 {
		t.Errorf("Changed.Tasks = %v on a refresh where nothing was written", changed.Tasks)
	}
}

// TestATaskTheEnumerationHasNotReachedLosesNothing is the half of the design
// the per-task row exists for. The stream is read for every task at once, so
// it carries events about tasks that have no row yet, and those are dropped —
// the reader that finds the task later reads its whole history instead, and
// the history is the record rather than what the stream happened to keep.
func TestATaskTheEnumerationHasNotReachedLosesNothing(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent())

	r := NewReader(s, work)
	refresh(t, r)

	addTask(t, s, repoPath, "ACME-2", created("Fix the swagger lint"), startedEvent(), failedEvent())

	// This refresh reads ACME-2's three events and files them under a task
	// that is not on the board, which is exactly the loss the catch-up undoes.
	if b, _ := refresh(t, r); len(b.Tasks) != 1 {
		t.Fatalf("Refresh found %d tasks: it re-walked the tree, which is the 2 s clock's job", len(b.Tasks))
	}

	if err := r.Rescan(); err != nil {
		t.Fatalf("Rescan: %v", err)
	}

	b, changed := refresh(t, r)
	if len(b.Tasks) != 2 {
		t.Fatalf("after a rescan there are %d rows, want 2", len(b.Tasks))
	}

	if !slices.Equal(changed.Tasks, []string{"ACME-2"}) {
		t.Errorf("Changed.Tasks = %v, want only the task the rescan found", changed.Tasks)
	}

	if got := b.Tasks[1]; got.Title != "Fix the swagger lint" || got.Attempt != 1 {
		t.Errorf("the caught-up task folded to %+v, want its whole history", got)
	}
}

// TestAnEventReachesATaskOnce is the rule that makes the catch-up safe to run
// at any moment. A task the enumeration has just found is behind the stream,
// so one refresh both hands it its history and holds the very same rows in
// what the stream read; taking both would show a run that was attempted twice
// over a record that says once.
func TestAnEventReachesATaskOnce(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent())

	r := NewReader(s, work)
	refresh(t, r)

	behind := r.at

	addTask(t, s, repoPath, "ACME-2", created("Fix the swagger lint"), startedEvent(), failedEvent())

	if err := r.Rescan(); err != nil {
		t.Fatalf("Rescan: %v", err)
	}

	if r.at != behind {
		t.Fatalf("the reader moved to row %d during a rescan, and this test needs it left at %d", r.at, behind)
	}

	b, _ := refresh(t, r)
	if b.Health.EventsRead != 3 {
		t.Errorf("the refresh read %d events over the 3 that were written: the stream and the catch-up overlapped", b.Health.EventsRead)
	}

	if got := b.Tasks[1].Attempt; got != 1 {
		t.Errorf("Attempt = %d, want 1: the catch-up and the stream both delivered the start", got)
	}

	entries, err := r.Log(repoPath, "ACME-2")
	if err != nil {
		t.Fatalf("Log: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("the log folded to %d entries, want the 3 rows the record holds", len(entries))
	}
}

// TestATaskWhoseHistoryWouldNotReadIsAskedForItAgain.
//
// st.err is the previous refresh's verdict. A task whose whole history would
// not read must go back to the history branch rather than onto the
// incremental one: its cursor never left zero, the rows written since are
// already past it, and the row would fold from nothing for ever while the
// error quietly dropped out of the board's own list of them.
func TestATaskWhoseHistoryWouldNotReadIsAskedForItAgain(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"))

	r := NewReader(s, work)
	refresh(t, r)

	st, held := r.index["ACME-1"]
	if !held {
		t.Fatal("the task is not on the board")
	}

	// The state a refresh leaves when the history would not read: seen,
	// remembered, and stuck at row zero with a verdict on it.
	st.err = errReadingHistory
	st.at = 0
	st.events = nil

	appendTo(t, s, repoPath, "ACME-1", finishedEvent())

	refresh(t, r)

	if len(st.events) == 0 {
		t.Fatal("a task whose history failed was moved onto the incremental branch and folds from nothing")
	}

	if st.err != nil {
		t.Errorf("the task still carries %v after its history was read", st.err)
	}

	// The whole history and not only what was written since: the task was
	// created before the failure and that event has to be back.
	var kinds []string
	for _, e := range st.events {
		kinds = append(kinds, e.Kind)
	}

	if !strings.Contains(strings.Join(kinds, " "), "task.created") {
		t.Errorf("it read %v, want the whole history", kinds)
	}
}

// TestAnArrivalExactlyAtTheCursorIsNotReadTwice.
//
// The one rule the whole design rests on, at its boundary: the cursor is the
// last row read, so a row at exactly that number has been read. Reading it
// again would put one event in a task's fold twice — and an event folded
// twice is a cost counted twice, a phase started twice, a task that says it
// did something once that it did once.
func TestAnArrivalExactlyAtTheCursorIsNotReadTwice(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"))

	r := NewReader(s, work)
	refresh(t, r)

	st := r.index["ACME-1"]

	cursor := st.at

	if cursor == 0 {
		t.Fatal("after a refresh the task has read up to row zero")
	}

	// The same rows offered again, as a stream that has not moved would.
	again := map[string][]arrival{
		"ACME-1": {{at: cursor, event: st.events[len(st.events)-1]}},
	}

	fresh, err := r.arrivals(st, again)
	if err != nil {
		t.Fatalf("read the arrivals: %v", err)
	}

	if len(fresh) != 0 {
		t.Errorf("a row at the cursor was read again: %d events", len(fresh))
	}

	if st.at != cursor {
		t.Errorf("the cursor moved to %d from %d over a row it had already read", st.at, cursor)
	}
}

// TestARowPastTheCursorIsRead, which is the other half of the same boundary:
// a cursor that refused everything would be a board that stopped at its
// first refresh.
func TestARowPastTheCursorIsRead(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"))

	r := NewReader(s, work)
	refresh(t, r)

	st := r.index["ACME-1"]

	cursor := st.at

	fresh, err := r.arrivals(st, map[string][]arrival{
		"ACME-1": {{at: cursor + 1, event: finishedEvent()}},
	})
	if err != nil {
		t.Fatalf("read the arrivals: %v", err)
	}

	if len(fresh) != 1 {
		t.Fatalf("a row past the cursor was read %d times, want once", len(fresh))
	}

	if st.at != cursor+1 {
		t.Errorf("the cursor is at %d, want it moved to the row that was read", st.at)
	}
}

// errReadingHistory stands in for the verdict a refresh leaves on a task
// whose whole history would not read.
var errReadingHistory = errHistory{}

type errHistory struct{}

func (errHistory) Error() string { return "the record of this task could not be read" }
