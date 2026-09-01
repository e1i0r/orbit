package board

// What the board says about the directory it is of: how many repositories
// are under it, how many tasks are in each band, which of the finished ones
// nobody has looked at, and what happens to the rest of it when one
// directory under the root will not open. The polling itself is
// refresh_test.go.

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// TestAnEmptyRootIsAnAnswer: a directory with no repository under it is
// empty, not broken.
func TestAnEmptyRootIsAnAnswer(t *testing.T) {
	s, work := newRoot(t)

	b, changed := refresh(t, NewReader(s, work))
	if len(b.Tasks) != 0 || b.Repos != 0 || len(b.Errs) != 0 {
		t.Errorf("an empty root gave %d tasks in %d repos and %d errors, want none of any", len(b.Tasks), b.Repos, len(b.Errs))
	}

	if b.Counts != ([4]int{}) {
		t.Errorf("an empty root counted %v", b.Counts)
	}

	if b.ReadAt.IsZero() {
		t.Error("the board does not say when it was read")
	}

	if len(changed.Tasks) != 0 || len(changed.Entered) != 0 {
		t.Errorf("an empty root changed something: %+v", changed)
	}
}

// TestARepositoryWithNoTasksIsStillARepository is the count in the header
// and where it comes from.
//
// A repository gains a directory under repos/ only when the first task is
// written against it, so a board that counted those would tell somebody who
// has just cloned a project that there are no repositories at all — and
// offer them the one action, clone one, that would change nothing. The count
// is what the walk found and the rows are what the record holds, and a fresh
// checkout is the first without the second.
func TestARepositoryWithNoTasksIsStillARepository(t *testing.T) {
	s, work, _ := oneRepo(t)

	b, _ := refresh(t, NewReader(s, work))
	if b.Repos != 1 {
		t.Errorf("Repos = %d, want 1: a repository nobody has written a task against is still a repository", b.Repos)
	}

	if len(b.Tasks) != 0 {
		t.Errorf("%d tasks were drawn from a repository that has none", len(b.Tasks))
	}

	if len(b.Errs) != 0 {
		t.Errorf("a repository with no tasks was reported as a fault: %v", b.Errs)
	}
}

// TestOnlyTheRepositoriesUnderTheRootAreOnTheBoard is the assertion the
// finding needed: two readers over one state root, opened over two different
// directories, answering two different boards.
//
// The record does not say where a window was pointed, so nothing about one
// board on its own can show that the directory was used at all. Two of them
// can.
func TestOnlyTheRepositoriesUnderTheRootAreOnTheBoard(t *testing.T) {
	s, work, payments := oneRepo(t)
	addTask(t, s, payments, "ACME-1", created("Retry the webhook on 5xx"))
	elsewhere := t.TempDir()
	billing := gitRepo(t, elsewhere, "billing")
	addTask(t, s, billing, "ACME-2", created("Reconcile the ledger nightly"))

	here, _ := refresh(t, NewReader(s, work))
	if here.Repos != 1 {
		t.Errorf("Repos = %d over %q, want 1: the other repository is under another root", here.Repos, work)
	}

	if len(here.Tasks) != 1 || here.Tasks[0].ID != "ACME-1" {
		t.Fatalf("the board over %q drew %+v, want only ACME-1", work, here.Tasks)
	}

	there, _ := refresh(t, NewReader(s, elsewhere))
	if there.Repos != 1 || len(there.Tasks) != 1 || there.Tasks[0].ID != "ACME-2" {
		t.Errorf("the board over %q drew %d repos and %+v, want 1 and only ACME-2", elsewhere, there.Repos, there.Tasks)
	}
}

// TestTheHeaderCountsWhatTheListDraws: Counts is indexed by view.BandOf and
// by nothing else, so the number above a band and the rows inside it cannot
// be two rules that agree by inspection.
func TestTheHeaderCountsWhatTheListDraws(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Index on settlements"))
	addTask(t, s, repoPath, "ACME-2", created("Reconciliation endpoint"), startedEvent())
	addTask(t, s, repoPath, "ACME-3", created("Retry the webhook on 5xx"), startedEvent(), failedEvent())
	addTask(t, s, repoPath, "ACME-4", created("Fix the swagger lint"), startedEvent(), finishedEvent())

	b, _ := refresh(t, NewReader(s, work))

	var drawn [4]int
	for _, task := range b.Tasks {
		drawn[view.BandOf(task)]++
	}

	if b.Counts != drawn {
		t.Errorf("the header counts %v and the list draws %v", b.Counts, drawn)
	}

	if want := ([4]int{view.ToDo: 1, view.NeedsYou: 1, view.Running: 1, view.Done: 1}); b.Counts != want {
		t.Errorf("Counts = %v, want %v", b.Counts, want)
	}
}

// TestRescanFindsWhatRefreshDoesNot is the two clocks made visible. Refresh
// polls the tasks it already knows and never re-walks the tree; a task
// written down after the window opened arrives on the next enumeration, and
// the tasks that were already there keep the row they had read up to.
func TestRescanFindsWhatRefreshDoesNot(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent())
	r := NewReader(s, work)
	refresh(t, r)
	carried := r.tasks[0].at

	addTask(t, s, repoPath, "ACME-2", created("Fix the swagger lint"))

	if b, _ := refresh(t, r); len(b.Tasks) != 1 {
		t.Errorf("Refresh found %d tasks: it re-walked the tree, which is the 2 s clock's job", len(b.Tasks))
	}

	if err := r.Rescan(); err != nil {
		t.Fatalf("Rescan: %v", err)
	}

	if r.tasks[0].at != carried {
		t.Errorf("a task that was already there is at row %d after a rescan, want %d", r.tasks[0].at, carried)
	}

	b, changed := refresh(t, r)
	if len(b.Tasks) != 2 {
		t.Fatalf("after a rescan there are %d tasks, want 2", len(b.Tasks))
	}

	if !slices.Equal(changed.Tasks, []string{"ACME-2"}) {
		t.Errorf("Changed.Tasks = %v, want only the task the rescan found", changed.Tasks)
	}
}

// TestARepositoryTakenOutOfTheRootTakesItsRowsWithIt is the cost of counting
// what the walk found rather than what the record holds, stated here rather
// than left to be found.
//
// The tasks of a repository that has been moved or deleted are not listed,
// under the name its path ended in or under any other, even though the
// record lives in the state root and outlives the checkout: the repositories
// are the ones under the root, and a checkout that is not there is not one
// of them. Nothing is lost — the record is untouched, `orbit show` and
// `orbit list` read it exactly as before — and the rows come back the moment
// the checkout does.
func TestARepositoryTakenOutOfTheRootTakesItsRowsWithIt(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent())

	if err := os.RemoveAll(repoPath); err != nil {
		t.Fatalf("remove the repository: %v", err)
	}

	b, _ := refresh(t, NewReader(s, work))
	if b.Repos != 0 || len(b.Tasks) != 0 {
		t.Errorf("a checkout that is no longer under the root gave %d repos and %d tasks, want none of either", b.Repos, len(b.Tasks))
	}

	if restored := gitRepo(t, work, "payments"); restored != repoPath {
		t.Fatalf("the repository was rebuilt at %q, want %q", restored, repoPath)
	}

	back, _ := refresh(t, NewReader(s, work))
	if len(back.Tasks) != 1 || back.Tasks[0].Title != "Retry the webhook on 5xx" {
		t.Errorf("the record did not come back with the checkout: %+v", back.Tasks)
	}
}

// TestADirectoryThatWillNotOpenIsNotFatal: something under the root that
// looks like a repository and is not — a .git that is neither a directory
// nor a gitfile git will take — is one directory left out of the listing,
// and not the board gone. It is the only fault the walk can meet that would
// otherwise be indistinguishable from an empty root, and it does not clear
// up on its own: getting it wrong draws an empty window for as long as the
// directory is there.
func TestADirectoryThatWillNotOpenIsNotFatal(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"))

	broken := filepath.Join(work, "not-really")
	if err := os.Mkdir(broken, 0o700); err != nil {
		t.Fatalf("mkdir %q: %v", broken, err)
	}

	if err := os.WriteFile(filepath.Join(broken, ".git"), []byte("this is not a gitfile\n"), 0o600); err != nil {
		t.Fatalf("write a .git nobody can open: %v", err)
	}

	b, _, err := NewReader(s, work).Refresh()
	if err != nil {
		t.Fatalf("Refresh: %v — one directory that will not open must never be fatal", err)
	}

	if b.Repos != 1 || len(b.Tasks) != 1 {
		t.Errorf("a directory that will not open gave %d repos and %d tasks, want 1 of each", b.Repos, len(b.Tasks))
	}
}

// TestARootThatCannotBeWalkedIsAnError is the one failure that is not
// isolated to a row: with no enumeration there is no board at all, and
// Refresh says so rather than answering an empty screen that a reader would
// take for a directory with nothing in it.
func TestARootThatCannotBeWalkedIsAnError(t *testing.T) {
	s, work := newRoot(t)
	if _, _, err := NewReader(s, filepath.Join(work, "nowhere")).Refresh(); err == nil {
		t.Error("Refresh accepted a root that is not there")
	}
}

// TestRefreshAndRescanMayRunTogether exists to be run under -race. Bubble
// Tea gives every Cmd a goroutine of its own and the window has two clocks,
// so nothing outside the Reader serialises the 500 ms refresh against the
// 2 s enumeration.
func TestRefreshAndRescanMayRunTogether(t *testing.T) {
	s, work, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent())
	r := NewReader(s, work)

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(2)
		go func() {
			defer wg.Done()

			if _, _, err := r.Refresh(); err != nil {
				t.Errorf("Refresh: %v", err)
			}
		}()
		go func() {
			defer wg.Done()

			if err := r.Rescan(); err != nil {
				t.Errorf("Rescan: %v", err)
			}
		}()
	}

	wg.Wait()
}

// TestUnreadCountsOnlyFinishedWorkNobodyHasLookedAt walks the four bands and
// the two exemptions. The rows are built by hand, which is exactly what
// view.BandOf's exported field is for.
func TestUnreadCountsOnlyFinishedWorkNobodyHasLookedAt(t *testing.T) {
	for _, tc := range []struct {
		name  string
		tasks []view.Task
		want  int
	}{
		{"an empty board", nil, 0},
		{"one finished task nobody read", []view.Task{{Band: view.Done}}, 1},
		{"one finished task somebody read", []view.Task{{Band: view.Done, Read: true}}, 0},
		{
			"a cancelled run is not homework",
			[]view.Task{{Band: view.Done, Reason: view.Reason{Key: view.ReasonCancelled}}},
			0,
		},
		{
			"a failure is already in front of the reader",
			[]view.Task{{Band: view.NeedsYou, Reason: view.Reason{Key: view.ReasonFailed}}},
			0,
		},
		{"nothing has run yet", []view.Task{{Band: view.ToDo}, {Band: view.Running}}, 0},
		{
			"three finished and one of them read",
			[]view.Task{
				{Band: view.Done},
				{Band: view.Done, Read: true},
				{Band: view.Done},
				{Band: view.Done, Reason: view.Reason{Key: view.ReasonCancelled}},
			},
			2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Unread(Board{Tasks: tc.tasks}); got != tc.want {
				t.Errorf("Unread is %d, want %d", got, tc.want)
			}
		})
	}
}
