package board

// A task reaches into as many checkouts as the work needed, so the board is
// a list of tasks and not of the pairs it used to be: which repositories one
// row names, which of them put it on this board at all, and what a task that
// joins another between two enumerations says afterwards.

import (
	"slices"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/view"
)

// TestATaskWorkedInTwoRepositoriesIsOneRow. A task reaches into as many
// checkouts as the work needed, and the board is a list of tasks: four
// repositories is one row saying four, not four rows saying the same task.
// Counting the pairs is how a band came to say 4 where there was 1.
func TestATaskWorkedInTwoRepositoriesIsOneRow(t *testing.T) {
	s, work, payments := oneRepo(t)
	ledger := gitRepo(t, work, "ledger")

	addTask(t, s, payments, "ACME-1", created("Move the fee table"))
	joinTo(t, s, "ACME-1", ledger)

	b, _ := refresh(t, NewReader(s, work))
	if len(b.Tasks) != 1 {
		t.Fatalf("a task worked in two repositories drew %d rows, want 1", len(b.Tasks))
	}

	if b.Counts != ([4]int{view.ToDo: 1}) {
		t.Errorf("the bands counted %v, want one task in to do", b.Counts)
	}

	if got := b.Tasks[0].Repos; !slices.Equal(got, []string{"payments", "ledger"}) {
		t.Errorf("the row says it is worked in %v, want payments and then ledger", got)
	}

	if b.Tasks[0].Repo != "payments" {
		t.Errorf("the row is filed under %q, want the repository it was written in", b.Tasks[0].Repo)
	}
}

// TestARepositoryOutsideTheRootIsOnTheRowAndNotOnTheBoard. A task that
// reaches into a checkout this window was not opened over is still a task
// that reached into it, and the row says so — but the repository itself is
// not one of this board's, because the board is of a directory.
func TestARepositoryOutsideTheRootIsOnTheRowAndNotOnTheBoard(t *testing.T) {
	s, work, payments := oneRepo(t)
	elsewhere := gitRepo(t, t.TempDir(), "billing")

	addTask(t, s, payments, "ACME-1", created("Split the fee table"))
	joinTo(t, s, "ACME-1", elsewhere)

	b, _ := refresh(t, NewReader(s, work))
	if b.Repos != 1 {
		t.Errorf("Repos = %d, want the one under the root", b.Repos)
	}

	if len(b.Tasks) != 1 {
		t.Fatalf("%d rows, want one", len(b.Tasks))
	}

	if got := b.Tasks[0].Repos; !slices.Equal(got, []string{"payments", "billing"}) {
		t.Errorf("the row says it is worked in %v, want both repositories it was worked in", got)
	}
}

// TestATaskOfNoRepositoryUnderTheRootIsNotDrawn. The board is of a
// directory, and the walk that asked each repository what it held said so by
// never asking. Asked from the task's end, it has to be said out loud.
func TestATaskOfNoRepositoryUnderTheRootIsNotDrawn(t *testing.T) {
	s, work := newRoot(t)
	elsewhere := gitRepo(t, t.TempDir(), "billing")

	addTask(t, s, elsewhere, "ACME-1", created("Somebody else's board"))

	b, _ := refresh(t, NewReader(s, work))
	if len(b.Tasks) != 0 {
		t.Errorf("a task of a repository outside the root drew %d rows", len(b.Tasks))
	}
}

// TestATaskThatJoinsARepositoryBetweenScansSaysSoOnce. The state a task
// carries across an enumeration is kept for its offset, and the
// repositories it names are the one part of it that has to be dropped: a
// task that joined one more would otherwise name the first ones twice.
func TestATaskThatJoinsARepositoryBetweenScansSaysSoOnce(t *testing.T) {
	s, work, payments := oneRepo(t)
	ledger := gitRepo(t, work, "ledger")

	addTask(t, s, payments, "ACME-1", created("Move the fee table"))

	r := NewReader(s, work)
	refresh(t, r)

	joinTo(t, s, "ACME-1", ledger)

	if err := r.Rescan(); err != nil {
		t.Fatalf("Rescan: %v", err)
	}

	b, _ := refresh(t, r)
	if len(b.Tasks) != 1 {
		t.Fatalf("%d rows after the second repository joined, want one", len(b.Tasks))
	}

	if got := b.Tasks[0].Repos; !slices.Equal(got, []string{"payments", "ledger"}) {
		t.Errorf("the row says it is worked in %v, want each repository once", got)
	}
}

// TestATaskThatHasReachedIntoNothingIsStillARow. The board is of a
// directory, and the test for a task belonging to it is which repositories
// it names — which answers nothing at all for a task that has joined none.
//
// It is kept, and the reason is the difference between the two silences: a
// task naming repositories, none of them under this root, belongs to some
// other root; a task naming none belongs to whichever root holds it, and
// this is that root. A row nobody can see is a run nobody can start, and the
// first phase of such a task is what finds its first checkout.
func TestATaskThatHasReachedIntoNothingIsStillARow(t *testing.T) {
	s, work, payments := oneRepo(t)

	addTask(t, s, payments, "ACME-1", created("Move the fee table"))
	nowhere(t, s, "ACME-2", created("Find out which service owns the retry"))

	b, _ := refresh(t, NewReader(s, work))
	if len(b.Tasks) != 2 {
		t.Fatalf("the board drew %d rows, want the task that is somewhere and the one that is nowhere", len(b.Tasks))
	}

	row := b.Tasks[0]
	if row.ID != "ACME-2" {
		t.Fatalf("the first row is %q, want the task that names no repository", row.ID)
	}

	if row.Repo != "" || row.RepoPath != "" || len(row.Repos) != 0 {
		t.Errorf("the row says it is worked in %q at %q, and in %v", row.Repo, row.RepoPath, row.Repos)
	}

	if row.Title != "Find out which service owns the retry" {
		t.Errorf("the row reads %q, want the task that was written", row.Title)
	}
}

// nowhere writes a task down that is joined to no repository, which is what
// task.Create leaves behind when it is given none.
func nowhere(t *testing.T, s *store.Store, id string, events ...record.Event) {
	t.Helper()

	if _, err := s.CreateTaskDir("", id); err != nil {
		t.Fatalf("create the directory of task %s: %v", id, err)
	}

	appendTo(t, s, "", id, events...)
}
