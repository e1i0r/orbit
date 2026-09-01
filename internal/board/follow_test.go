package board

// Following the record by row: what one refresh reads, what a task the
// enumeration has not reached yet does with what was written about it, and
// the one rule the whole design rests on — an event reaches a task once.

import (
	"slices"
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
