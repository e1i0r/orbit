package board

import (
	"os"
	"path/filepath"
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

// TestTheSecondReadStartsAtTheRowTheFirstStopped is the assertion the
// polling design is worth nothing without. Two refreshes that both give the
// right answer prove nothing — a reader that re-read every task's whole
// history would pass that — so what is asserted is the reading itself: the
// second refresh reads the one event that was written and not the three that
// are there.
func TestTheSecondReadStartsAtTheRowTheFirstStopped(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent())

	r := NewReader(s, work)
	b, _ := refresh(t, r)

	first := r.tasks[0].at
	if b.Health.EventsRead != 2 || first == 0 {
		t.Fatalf("the first refresh read %d events and stopped at row %d, want 2 and a row of its own", b.Health.EventsRead, first)
	}

	appendTo(t, s, repoPath, "ACME-1", failedEvent())

	b, changed := refresh(t, r)
	if b.Health.EventsRead != 1 {
		t.Errorf("the second refresh read %d events, want the 1 that was appended", b.Health.EventsRead)
	}

	if r.tasks[0].at <= first {
		t.Errorf("the task is still at row %d after an event was written past %d", r.tasks[0].at, first)
	}

	got := b.Tasks[0]
	if got.Title != "Retry the webhook on 5xx" {
		t.Errorf("Title = %q: what the first refresh read was not kept", got.Title)
	}

	if got.Attempt != 1 {
		t.Errorf("Attempt = %d, want 1: rows the first refresh had read were folded in a second time", got.Attempt)
	}

	if view.BandOf(got) != view.NeedsYou {
		t.Errorf("the task is banded %s: the appended failure was not read at all", view.BandOf(got))
	}

	if !slices.Equal(changed.Tasks, []string{"ACME-1"}) {
		t.Errorf("Changed.Tasks = %v, want [ACME-1]", changed.Tasks)
	}
}

// TestARecordThatWillNotOpenIsARefusalAndNotAnEmptyBoard is where the move
// to one record changed what a failure means. A log per task made an
// unreadable one task's problem: nineteen rows drew and the twentieth said
// so. There is one record now, so a record that will not open is every task's
// problem, and answering an empty board would say there is nothing to do —
// which is a different sentence from "nobody could look".
func TestARecordThatWillNotOpenIsARefusalAndNotAnEmptyBoard(t *testing.T) {
	s, work, _ := oneRepo(t)
	unopenable(t, s)

	b, _, err := NewReader(s, work).Refresh()
	if err == nil {
		t.Fatal("a record nobody can open was refreshed without complaint")
	}

	if len(b.Tasks) != 0 {
		t.Errorf("the failed refresh still drew %d rows", len(b.Tasks))
	}

	if b.Health.Errs != 1 {
		t.Errorf("Health.Errs = %d, want the one failure to be counted", b.Health.Errs)
	}
}

// TestTheEnumerationDoesNotOpenTheRepositoriesItFinds is the cost of the
// two second clock.
//
// Opening a repository is three git subprocesses, and the enumeration ran
// them for every repository under the root every time it re-walked: thirteen
// checkouts came to thirty-nine processes a rescan, for a remote and a
// branch name the board never draws. It asks for the path and the name now,
// which the walk already knows.
//
// The fixture is what proves it: a directory with a .git that git cannot
// open at all. An enumeration that still counts it is one that never asked.
func TestTheEnumerationDoesNotOpenTheRepositoriesItFinds(t *testing.T) {
	s, work := newRoot(t)

	if err := os.MkdirAll(filepath.Join(work, "payments", ".git"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	b, _ := refresh(t, NewReader(s, work))
	if b.Repos != 1 {
		t.Errorf("the board is of %d repositories, want 1 — the walk found it and "+
			"nothing had to run git to say so", b.Repos)
	}
}
