package board

import (
	"errors"
	"os"
	"slices"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// TestTheFirstRefreshRingsNoBell is the rule the whole notification channel
// rests on. Opening the window on tasks that already need you establishes a
// baseline; it does not announce twelve historic failures, which is how a
// channel stops being trusted on its first use.
func TestTheFirstRefreshRingsNoBell(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent(), failedEvent())
	addTask(t, s, repoPath, "ACME-2", created("Fix the swagger lint"), startedEvent(), failedEvent())

	b, changed := refresh(t, NewReader(s, work))
	if len(changed.Entered) != 0 {
		t.Errorf("the first refresh announced %v, and it must announce nothing", changed.Entered)
	}

	if b.Counts[view.NeedsYou] != 2 {
		t.Errorf("Counts[NeedsYou] = %d, want 2 — it found them, it just does not ring", b.Counts[view.NeedsYou])
	}

	if !slices.Equal(changed.Tasks, []string{"ACME-1", "ACME-2"}) {
		t.Errorf("Changed.Tasks = %v, want both tasks: they are new to this reader", changed.Tasks)
	}
}

// TestCrossingIntoNeedsYouIsAnnouncedOnce: Entered is the crossing and not
// the state. The refresh that reads the failure announces it; the one after
// has nothing to say about a task that has not moved.
func TestCrossingIntoNeedsYouIsAnnouncedOnce(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent())

	r := NewReader(s, work)
	if _, changed := refresh(t, r); len(changed.Entered) != 0 {
		t.Fatalf("a running task was announced: %v", changed.Entered)
	}

	appendTo(t, s, repoPath, "ACME-1", failedEvent())

	b, changed := refresh(t, r)
	if !slices.Equal(changed.Entered, []string{"ACME-1"}) {
		t.Errorf("Entered = %v on the refresh that read the failure, want [ACME-1]", changed.Entered)
	}

	if got := view.BandOf(b.Tasks[0]); got != view.NeedsYou {
		t.Errorf("the task is banded %s, want %s", got, view.NeedsYou)
	}

	_, changed = refresh(t, r)
	if len(changed.Entered) != 0 {
		t.Errorf("Entered = %v on a refresh where nothing crossed, want nothing", changed.Entered)
	}

	if len(changed.Tasks) != 0 {
		t.Errorf("Changed.Tasks = %v for a log that did not move", changed.Tasks)
	}
}

// TestTheSecondReadStartsWhereTheFirstStopped is the assertion the polling
// design is worth nothing without. Two refreshes that both give the right
// answer prove nothing — a reader that re-read every log from byte zero
// would pass that. So the offset is asserted directly, and then every byte
// the first refresh already read is overwritten with nonsense: a reader
// that started over would fold damaged lines and lose the title, and a
// reader that starts at its stored offset never looks at them again.
func TestTheSecondReadStartsWhereTheFirstStopped(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent())
	path := eventsPath(t, s, repoPath, "ACME-1")

	r := NewReader(s, work)
	refresh(t, r)

	first := r.tasks[0].offset
	if want := sizeOf(t, path); first != want {
		t.Fatalf("after one refresh the offset is %d and the log is %d bytes long", first, want)
	}

	poison(t, path, first)
	appendTo(t, s, repoPath, "ACME-1", failedEvent())

	b, changed := refresh(t, r)
	if want := sizeOf(t, path); r.tasks[0].offset != want {
		t.Errorf("after the second refresh the offset is %d and the log is %d bytes long", r.tasks[0].offset, want)
	}

	got := b.Tasks[0]
	if got.Damaged != 0 {
		t.Errorf("Damaged = %d: the second refresh re-read bytes it had already read", got.Damaged)
	}

	if got.Title != "Retry the webhook on 5xx" {
		t.Errorf("Title = %q: what the first refresh read was not kept", got.Title)
	}

	if view.BandOf(got) != view.NeedsYou {
		t.Errorf("the task is banded %s: the appended failure was not read at all", view.BandOf(got))
	}

	if !slices.Equal(changed.Tasks, []string{"ACME-1"}) {
		t.Errorf("Changed.Tasks = %v, want [ACME-1]", changed.Tasks)
	}
}

// TestAnUnreadableLogDoesNotBlankTheBoard: one task whose record cannot be
// read is one row that says so and nineteen rows that are unaffected. The
// error names the task, because a window that had to match on the words of
// an error message to find the row would be testing the message.
func TestAnUnreadableLogDoesNotBlankTheBoard(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent())
	addTask(t, s, repoPath, "ACME-2", created("Fix the swagger lint"), startedEvent())
	tooLongLine(t, eventsPath(t, s, repoPath, "ACME-2"))

	r := NewReader(s, work)

	b, _ := refresh(t, r)
	if len(b.Tasks) != 2 {
		t.Fatalf("%d rows, want 2: an unreadable log took the other task with it", len(b.Tasks))
	}

	if b.Tasks[0].Title != "Retry the webhook on 5xx" || view.BandOf(b.Tasks[0]) != view.Running {
		t.Errorf("the readable task folded to %+v", b.Tasks[0])
	}

	var unreadable *TaskError
	if len(b.Errs) != 1 || !errors.As(b.Errs[0], &unreadable) {
		t.Fatalf("Errs = %v, want one TaskError", b.Errs)
	}

	if unreadable.ID != "ACME-2" || unreadable.Repo != "payments" {
		t.Errorf("the error names task %q in %q, want ACME-2 in payments", unreadable.ID, unreadable.Repo)
	}

	// And it keeps saying so. The next refresh finds that log unchanged and
	// skips reading it, which must not be the same as finding nothing wrong
	// with it: a row whose record cannot be read quietly becoming an ordinary
	// row is worse than the failure it hides.
	b, changed := refresh(t, r)
	if len(b.Errs) != 1 || !errors.As(b.Errs[0], &unreadable) {
		t.Errorf("Errs = %v on the second refresh, want the same failure still reported", b.Errs)
	}

	if len(changed.Tasks) != 0 {
		t.Errorf("Changed.Tasks = %v: neither log moved and neither verdict flipped", changed.Tasks)
	}
}

// TestALogThatWasReplacedIsReadAgainFromTheTop: a log shorter than what has
// already been read was replaced rather than appended to. Reading it from the
// top is record.ReadFrom's own doing; what this package has to add is
// forgetting what it folded from the log that is gone, because a task shown
// as attempted twice when its record says once is the reader inventing
// history out of two different logs.
func TestALogThatWasReplacedIsReadAgainFromTheTop(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1",
		created("Retry the webhook on 5xx, and stop pretending it is idempotent"),
		startedEvent(), failedEvent())
	path := eventsPath(t, s, repoPath, "ACME-1")

	r := NewReader(s, work)
	refresh(t, r)
	read := r.tasks[0].offset

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove %q: %v", path, err)
	}

	appendTo(t, s, repoPath, "ACME-1", created("Index on settlements"), startedEvent())

	if size := sizeOf(t, path); size >= read {
		t.Fatalf("the replacement is %d bytes and %d had been read: this test needs a shorter log", size, read)
	}

	b, changed := refresh(t, r)

	got := b.Tasks[0]
	if got.Title != "Index on settlements" {
		t.Errorf("Title = %q, want the replacement's title", got.Title)
	}

	if got.Attempt != 1 {
		t.Errorf("Attempt = %d, want 1: the log that was replaced was folded in with the one that replaced it", got.Attempt)
	}

	if want := sizeOf(t, path); r.tasks[0].offset != want {
		t.Errorf("the offset is %d and the replacement is %d bytes long", r.tasks[0].offset, want)
	}

	if !slices.Equal(changed.Tasks, []string{"ACME-1"}) {
		t.Errorf("Changed.Tasks = %v, want [ACME-1]", changed.Tasks)
	}
}
