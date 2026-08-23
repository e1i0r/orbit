package board

// What the board says about the whole state root: how many repositories are
// in it, how many tasks are in each band, and what happens to the rest of it
// when one repository is damaged. The polling itself is refresh_test.go.

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// TestAnEmptyRootIsAnAnswer: a state root nobody has written a task against
// is empty, not broken.
func TestAnEmptyRootIsAnAnswer(t *testing.T) {
	b, changed := refresh(t, NewReader(newRoot(t)))
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

// TestARepositoryWithNoTasksIsStillARepository covers the two shapes a
// directory under repos/ can have without producing a row: one with a
// marker and no tasks, which is counted, and one with no marker at all,
// which is a half-created directory and is skipped in silence.
func TestARepositoryWithNoTasksIsStillARepository(t *testing.T) {
	s, repoPath := oneRepo(t)
	dir, err := s.CreateTaskDir(repoPath, "ACME-1")
	if err != nil {
		t.Fatalf("create the directory of task ACME-1: %v", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("take the task back out: %v", err)
	}
	unmarked := filepath.Join(s.Root(), "repos", "0123456789ab", "tasks", "ACME-9")
	if err := os.MkdirAll(unmarked, 0o700); err != nil {
		t.Fatalf("mkdir %q: %v", unmarked, err)
	}

	b, _ := refresh(t, NewReader(s))
	if b.Repos != 1 {
		t.Errorf("Repos = %d, want 1: the marked repository counts and the unmarked directory does not", b.Repos)
	}
	if len(b.Tasks) != 0 {
		t.Errorf("%d tasks were drawn from repositories that have none", len(b.Tasks))
	}
	if len(b.Errs) != 0 {
		t.Errorf("a directory with no marker was reported as a fault: %v", b.Errs)
	}
}

// TestTheHeaderCountsWhatTheListDraws: Counts is indexed by view.BandOf and
// by nothing else, so the number above a band and the rows inside it cannot
// be two rules that agree by inspection.
func TestTheHeaderCountsWhatTheListDraws(t *testing.T) {
	s, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Index on settlements"))
	addTask(t, s, repoPath, "ACME-2", created("Reconciliation endpoint"), startedEvent())
	addTask(t, s, repoPath, "ACME-3", created("Retry the webhook on 5xx"), startedEvent(), failedEvent())
	addTask(t, s, repoPath, "ACME-4", created("Fix the swagger lint"), startedEvent(), finishedEvent())

	b, _ := refresh(t, NewReader(s))
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
// the tasks that were already there keep the offsets they had.
func TestRescanFindsWhatRefreshDoesNot(t *testing.T) {
	s, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent())
	r := NewReader(s)
	refresh(t, r)
	carried := r.tasks[0].offset

	addTask(t, s, repoPath, "ACME-2", created("Fix the swagger lint"))
	if b, _ := refresh(t, r); len(b.Tasks) != 1 {
		t.Errorf("Refresh found %d tasks: it re-walked the tree, which is the 2 s clock's job", len(b.Tasks))
	}

	if err := r.Rescan(); err != nil {
		t.Fatalf("Rescan: %v", err)
	}
	if r.tasks[0].offset != carried {
		t.Errorf("a task that was already there has offset %d after a rescan, want %d", r.tasks[0].offset, carried)
	}
	b, changed := refresh(t, r)
	if len(b.Tasks) != 2 {
		t.Fatalf("after a rescan there are %d tasks, want 2", len(b.Tasks))
	}
	if !slices.Equal(changed.Tasks, []string{"ACME-2"}) {
		t.Errorf("Changed.Tasks = %v, want only the task the rescan found", changed.Tasks)
	}
}

// TestARepositoryThatIsNotThereStillListsItsTasks: the record lives in the
// state root and the checkout does not, so a repository that has been moved
// or deleted takes nothing on screen with it. It is reported, and its tasks
// fold exactly as before.
func TestARepositoryThatIsNotThereStillListsItsTasks(t *testing.T) {
	s, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent())
	if err := os.RemoveAll(repoPath); err != nil {
		t.Fatalf("remove the repository: %v", err)
	}

	b, _ := refresh(t, NewReader(s))
	if len(b.Tasks) != 1 || b.Tasks[0].Title != "Retry the webhook on 5xx" {
		t.Fatalf("the tasks of a missing repository were lost: %+v", b.Tasks)
	}
	if b.Tasks[0].Repo != "payments" || b.Tasks[0].RepoPath != repoPath {
		t.Errorf("the row says %q at %q, want payments at %q", b.Tasks[0].Repo, b.Tasks[0].RepoPath, repoPath)
	}
	if len(b.Errs) != 1 {
		t.Errorf("Errs = %v, want one error naming the repository", b.Errs)
	}
}

// TestARootThatCannotBeListedIsAnError is the one failure that is not
// isolated to a row: with no enumeration there is no board at all, and
// Refresh says so rather than returning an empty screen that a reader would
// take for an empty root.
func TestARootThatCannotBeListedIsAnError(t *testing.T) {
	s := newRoot(t)
	if err := os.WriteFile(filepath.Join(s.Root(), "repos"), []byte("not a directory\n"), 0o600); err != nil {
		t.Fatalf("write a file where repos/ goes: %v", err)
	}
	if _, _, err := NewReader(s).Refresh(); err == nil {
		t.Error("Refresh accepted a state root whose repositories cannot be listed")
	}
}

// TestRefreshAndRescanMayRunTogether exists to be run under -race. Bubble
// Tea gives every Cmd a goroutine of its own and the window has two clocks,
// so nothing outside the Reader serialises the 500 ms refresh against the
// 2 s enumeration.
func TestRefreshAndRescanMayRunTogether(t *testing.T) {
	s, repoPath := oneRepo(t)
	addTask(t, s, repoPath, "ACME-1", created("Retry the webhook on 5xx"), startedEvent())
	r := NewReader(s)
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
