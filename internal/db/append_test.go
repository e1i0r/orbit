package db

// One event in, one event out — and the rows an event makes on the way past.

import (
	"sync"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// TestAnEventComesBackTheWayItWentIn. Everything else in Orbit is folded
// from these, so a field lost here is a field lost everywhere.
func TestAnEventComesBackTheWayItWentIn(t *testing.T) {
	d := open(t)

	want := record.Event{
		At:    time.Date(2026, 8, 31, 9, 0, 0, 123456789, time.UTC),
		Kind:  record.PhaseToolCall,
		Phase: "implement",
		Text:  "Edit internal/db/append.go",
		Data:  map[string]string{"tool": "Edit", "path": "internal/db/append.go"},
	}

	if err := d.Append("ACME-1", want); err != nil {
		t.Fatalf("append: %v", err)
	}

	events, err := d.Events("ACME-1")
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("read back %d events, want one", len(events))
	}

	got := events[0]

	// Nanoseconds included: the stamp is what tells two lines of one process
	// apart, so a column that rounded would merge them.
	if !got.At.Equal(want.At) {
		t.Errorf("at is %v, want %v", got.At, want.At)
	}

	if got.Kind != want.Kind || got.Phase != want.Phase || got.Text != want.Text {
		t.Errorf("read back %+v, want %+v", got, want)
	}

	if len(got.Data) != 2 || got.Data["tool"] != "Edit" || got.Data["path"] != want.Data["path"] {
		t.Errorf("data is %v, want %v", got.Data, want.Data)
	}
}

// TestAnEventWithNoDataHasNoData. Most events carry none, and a map with
// nothing in it is not the same answer as the four characters "null".
func TestAnEventWithNoDataHasNoData(t *testing.T) {
	d := open(t)

	if err := d.Append("ACME-1", record.Event{Kind: record.TaskRead}); err != nil {
		t.Fatalf("append: %v", err)
	}

	events, err := d.Events("ACME-1")
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	if len(events[0].Data) != 0 {
		t.Errorf("data is %v, want nothing", events[0].Data)
	}
}

// TestAnEventWithNoTimeIsStampedOnArrival. A caller that did not say when is
// saying now, and a record line with no time cannot be ordered against the
// ones beside it.
func TestAnEventWithNoTimeIsStampedOnArrival(t *testing.T) {
	d := open(t)

	before := time.Now()

	if err := d.Append("ACME-1", record.Event{Kind: record.TaskRead}); err != nil {
		t.Fatalf("append: %v", err)
	}

	events, err := d.Events("ACME-1")
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	if at := events[0].At; at.Before(before) || at.After(time.Now()) {
		t.Errorf("the event is stamped %v, want a moment during the append", at)
	}
}

// TestTheTaskRowIsWhatTaskCreatedSaid. task.created is the one event that
// says what a task is, and the row is what a board reads instead of folding
// every event of every task to find a title.
func TestTheTaskRowIsWhatTaskCreatedSaid(t *testing.T) {
	d := open(t)
	tick := clock()

	created := record.Event{
		At:   tick(),
		Kind: record.TaskCreated,
		Text: "Retry the webhook on 5xx",
		Data: map[string]string{"flow": "deliver"},
	}

	if err := d.Append("ACME-1", created); err != nil {
		t.Fatalf("append: %v", err)
	}

	var text, flow string
	if err := d.sql.QueryRow(`SELECT text, flow FROM task WHERE task_id = ?`, "ACME-1").Scan(&text, &flow); err != nil {
		t.Fatalf("read the task row: %v", err)
	}

	if text != created.Text || flow != "deliver" {
		t.Errorf("the task row is %q on flow %q, want %q on deliver", text, flow, created.Text)
	}
}

// TestATaskAppearsBecauseAnEventMentionedIt. The record is what happened,
// and an event about a task nobody wrote down first is still something that
// happened. Refusing it would lose the event to protect a foreign key.
func TestATaskAppearsBecauseAnEventMentionedIt(t *testing.T) {
	d := open(t)
	tick := clock()

	if err := d.Append("ACME-9", record.Event{At: tick(), Kind: record.TaskStarted}); err != nil {
		t.Fatalf("append an event of a task nobody created: %v", err)
	}

	// And the description arriving late fills the row rather than making a
	// second one — a log migrated out of order does exactly this.
	late := record.Event{At: tick(), Kind: record.TaskCreated, Text: "Found later"}
	if err := d.Append("ACME-9", late); err != nil {
		t.Fatalf("append the late task.created: %v", err)
	}

	ids, err := d.Tasks()
	if err != nil {
		t.Fatalf("read the tasks: %v", err)
	}

	if len(ids) != 1 || ids[0] != "ACME-9" {
		t.Fatalf("the record holds %v, want the one task", ids)
	}

	var text string
	if err := d.sql.QueryRow(`SELECT text FROM task WHERE task_id = ?`, "ACME-9").Scan(&text); err != nil {
		t.Fatalf("read the task row: %v", err)
	}

	if text != late.Text {
		t.Errorf("the task row says %q, want the description that arrived late", text)
	}
}

// TestJoiningARepositoryIsObservedRatherThanDeclared. Opening a worktree is
// what joining is, so repo.joined is the event that makes the row — twice
// for one repository still makes one.
func TestJoiningARepositoryIsObservedRatherThanDeclared(t *testing.T) {
	d := open(t)
	tick := clock()

	join := record.Event{
		At:   tick(),
		Kind: record.RepoJoined,
		Data: map[string]string{"path": "/src/acme", "repo": "acme"},
	}

	for range 2 {
		if err := d.Append("ACME-1", join); err != nil {
			t.Fatalf("append repo.joined: %v", err)
		}
	}

	var repos, joins int
	if err := d.sql.QueryRow(`SELECT count(*) FROM repo`).Scan(&repos); err != nil {
		t.Fatalf("count the repositories: %v", err)
	}

	if err := d.sql.QueryRow(`SELECT count(*) FROM task_repo`).Scan(&joins); err != nil {
		t.Fatalf("count the joins: %v", err)
	}

	if repos != 1 || joins != 1 {
		t.Errorf("two identical joins made %d repositories and %d joins, want one of each", repos, joins)
	}

	// A join naming no path is a malformed event, not a repository called
	// "": it is written down like any other event and makes no row.
	if err := d.Append("ACME-1", record.Event{At: tick(), Kind: record.RepoJoined}); err != nil {
		t.Fatalf("append a join with no path: %v", err)
	}

	if err := d.sql.QueryRow(`SELECT count(*) FROM repo`).Scan(&repos); err != nil {
		t.Fatalf("count the repositories: %v", err)
	}

	if repos != 1 {
		t.Errorf("a join naming no path made %d repositories, want the one that was named", repos)
	}
}

// TestEveryWriterLandsItsEvents is the property the whole shape rests on:
// each task is its own process, so the record has as many writers as there
// are runs. What is asserted is that nothing is lost — not the pace, which
// was measured elsewhere and is two orders of magnitude past what Orbit does.
func TestEveryWriterLandsItsEvents(t *testing.T) {
	d := open(t)

	const writers, each = 8, 25

	var wg sync.WaitGroup

	errs := make(chan error, writers*each)

	for range writers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := range each {
				e := record.Event{Kind: record.PhaseToolCall, Text: "Bash", Data: map[string]string{"n": string(rune('a' + i%26))}}
				if err := d.Append("ACME-1", e); err != nil {
					errs <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("a writer failed: %v", err)
	}

	events, err := d.Events("ACME-1")
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	if len(events) != writers*each {
		t.Errorf("%d events landed, want %d", len(events), writers*each)
	}
}
