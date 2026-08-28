package mcp

// Telling Orbit about a repository, asking about one, and taking one off the
// books. handlers_test.go covers the listing; these are the three tools that
// change what is in it, and the one thing they have to get right is that
// forgetting a repository deletes the only account of what its runs did.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// anywhere is a session with no root of its own — the one a desktop
// application spawns, which finds repositories through the state root rather
// than through the directory it happens to have been started in. It is the
// session the repository tools exist for.
func anywhere() Session { return Session{Version: "test"} }

func TestAddRepoMakesARepositoryOrbitKnowsWithoutATask(t *testing.T) {
	s, work := newRoot(t)
	r := gitRepo(t, work, "payments")
	sn := anywhere()

	got := call(t, sn, "orbit_add_repo", map[string]any{"path": r.Path})
	if got["name"] != "payments" || got["path"] != r.Path {
		t.Errorf("orbit_add_repo answered %v, want payments at %q", got, r.Path)
	}
	if got["branch"] != r.Base {
		t.Errorf("branch = %v, want %q", got["branch"], r.Base)
	}
	if got["tasks"] != float64(0) {
		t.Errorf("tasks = %v, want 0: nothing has been run there", got["tasks"])
	}

	refs, err := s.Repos()
	if err != nil {
		t.Fatalf("Repos: %v", err)
	}
	if len(refs) != 1 || refs[0].Path != r.Path {
		t.Fatalf("the state root knows %+v, want the repository that was added", refs)
	}
	repos := list(t, call(t, sn, "orbit_list_repos", nil)["repos"])
	if len(repos) != 1 || obj(t, repos[0])["path"] != r.Path {
		t.Errorf("orbit_list_repos answered %v, want the repository that was added", repos)
	}
}

// A model given a path inside a checkout has given the checkout: it is the
// same repository the walk would have found, and the same key every task
// under it is already filed against.
func TestAddRepoRegistersTheCheckoutAndNotADirectoryInsideIt(t *testing.T) {
	s, work := newRoot(t)
	r := gitRepo(t, work, "payments")
	inside := filepath.Join(r.Path, "internal")
	if err := os.Mkdir(inside, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if got := call(t, anywhere(), "orbit_add_repo", map[string]any{"path": inside}); got["path"] != r.Path {
		t.Errorf("orbit_add_repo recorded %v, want the checkout at %q", got["path"], r.Path)
	}
	refs, err := s.Repos()
	if err != nil {
		t.Fatalf("Repos: %v", err)
	}
	if len(refs) != 1 || refs[0].Path != r.Path {
		t.Errorf("the state root knows %+v, want one record, of the checkout", refs)
	}
}

// Adding a repository twice is what a supervisor does every time it starts,
// and it must not disturb the tasks already recorded there.
func TestAddRepoTwiceLeavesTheTasksAlone(t *testing.T) {
	s, work := newRoot(t)
	r := gitRepo(t, work, "payments")
	addTask(t, s, r, "PAY-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})
	if got := call(t, anywhere(), "orbit_add_repo", map[string]any{"path": r.Path}); got["tasks"] != float64(1) {
		t.Errorf("tasks = %v, want the 1 that was already there", got["tasks"])
	}
	if got := call(t, anywhere(), "orbit_inspect_task", map[string]any{"task_id": "PAY-1"}); got["id"] != "PAY-1" {
		t.Errorf("PAY-1 reads back as %v after its repository was added again", got["id"])
	}
}

func TestAddRepoRefusesWhatIsNotACheckout(t *testing.T) {
	newRoot(t)
	sn := anywhere()
	if said := refused(t, sn, "orbit_add_repo", nil); !strings.Contains(said, "path") {
		t.Errorf("orbit_add_repo with no path said %q, want it to ask for one", said)
	}
	plain := t.TempDir()
	if said := refused(t, sn, "orbit_add_repo", map[string]any{"path": plain}); said == "" {
		t.Error("a directory that is not a repository was accepted")
	}
}

func TestInspectRepoAnswersTheCheckoutAndHowItsTasksStand(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})
	addTask(t, s, r, "PAY-2",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "failed"},
		record.Event{At: at(2), Kind: record.TaskStarted},
		record.Event{At: at(3), Kind: record.TaskFailed, Text: "the build broke"})
	addTask(t, s, r, "PAY-3",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "finished"},
		record.Event{At: at(2), Kind: record.TaskStarted},
		record.Event{At: at(3), Kind: record.TaskFinished})

	got := call(t, sn, "orbit_inspect_repo", map[string]any{"repo": "payments"})
	if got["name"] != r.Name || got["path"] != r.Path || got["branch"] != r.Base {
		t.Errorf("orbit_inspect_repo answered %v, want %s at %q on %q", got, r.Name, r.Path, r.Base)
	}
	if got["tasks"] != float64(3) {
		t.Errorf("tasks = %v, want 3", got["tasks"])
	}
	counts := obj(t, got["counts"])
	for band, want := range map[string]float64{"todo": 1, "needs_you": 1, "done": 1, "running": 0} {
		if counts[band] != want {
			t.Errorf("counts[%q] = %v, want %v — the whole counts map is %v", band, counts[band], want, counts)
		}
	}
	if got["unread"] != float64(1) {
		t.Errorf("unread = %v, want the 1 finished task nobody has read", got["unread"])
	}
	if _, ok := got["spend"]; !ok {
		t.Error("the answer says nothing about what the repository has cost")
	}
}

func TestInspectRepoTakesANameOrAPath(t *testing.T) {
	_, sn, r := oneRepo(t)
	call(t, sn, "orbit_add_repo", map[string]any{"path": r.Path})
	byName := call(t, sn, "orbit_inspect_repo", map[string]any{"repo": r.Name})
	byPath := call(t, sn, "orbit_inspect_repo", map[string]any{"repo": r.Path})
	if byName["path"] != byPath["path"] || byName["path"] != r.Path {
		t.Errorf("by name %v and by path %v, want both to be %q", byName["path"], byPath["path"], r.Path)
	}
}

// The state a caller is in when it reaches for orbit_forget_repo: the record
// is there and the checkout is not. A tool that refused because git could not
// be asked would hide the thing being cleaned up.
func TestInspectRepoDescribesARepositoryWhoseCheckoutIsGone(t *testing.T) {
	s, work := newRoot(t)
	r := gitRepo(t, work, "payments")
	addTask(t, s, r, "PAY-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})
	if err := os.RemoveAll(r.Path); err != nil {
		t.Fatalf("remove the checkout: %v", err)
	}

	got := call(t, anywhere(), "orbit_inspect_repo", map[string]any{"repo": "payments"})
	if got["path"] != r.Path {
		t.Errorf("path = %v, want the %q orbit has a record of", got["path"], r.Path)
	}
	if _, ok := got["checkout_error"]; !ok {
		t.Errorf("the answer does not say the checkout could not be read: %v", got)
	}
	if _, ok := got["branch"]; ok {
		t.Errorf("the answer names a branch of a checkout that is gone: %v", got["branch"])
	}
}

func TestRepoToolsRefuseANameNobodyKnows(t *testing.T) {
	_, sn, _ := oneRepo(t)
	for _, name := range []string{"orbit_inspect_repo", "orbit_forget_repo"} {
		if said := refused(t, sn, name, nil); !strings.Contains(said, "repo") {
			t.Errorf("%s with no repository said %q, want it to ask for one", name, said)
		}
		if said := refused(t, sn, name, map[string]any{"repo": "ledger"}); !strings.Contains(said, "ledger") {
			t.Errorf("%s answered %q, want it to name the repository it could not find", name, said)
		}
	}
}

// Forgetting is the one operation that deletes from the append-only record,
// so it refuses while there is anything there to delete.
func TestForgetRepoRefusesWhileItHoldsTasks(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})

	said := refused(t, sn, "orbit_forget_repo", map[string]any{"repo": "payments"})
	for _, want := range []string{"PAY-1", "delete_tasks"} {
		if !strings.Contains(said, want) {
			t.Errorf("the refusal does not mention %q: %s", want, said)
		}
	}
	refs, err := s.Repos()
	if err != nil {
		t.Fatalf("Repos: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("the record went away after a refused forget: %+v", refs)
	}
	if got := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "PAY-1"}); got["id"] != "PAY-1" {
		t.Error("PAY-1 is gone after a refused forget")
	}
}

func TestForgetRepoRemovesTheRecordWhenTold(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})
	dir, err := s.RepoDir(r.Path)
	if err != nil {
		t.Fatalf("RepoDir: %v", err)
	}

	got := call(t, sn, "orbit_forget_repo", map[string]any{"repo": "payments", "delete_tasks": true})
	if got["removed"] != dir || got["tasks_deleted"] != float64(1) {
		t.Errorf("orbit_forget_repo answered %v, want it to have removed %q and 1 task", got, dir)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the record is still at %q: %v", dir, err)
	}
	refs, err := s.Repos()
	if err != nil {
		t.Fatalf("Repos: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("orbit still knows %+v", refs)
	}
	// The checkout is the reader's own and is never Orbit's to remove.
	if _, err := os.Stat(r.Path); err != nil {
		t.Errorf("forgetting the record took the checkout with it: %v", err)
	}
}

func TestForgetRepoTakesAnEmptyRepositoryWithoutBeingTold(t *testing.T) {
	s, work := newRoot(t)
	r := gitRepo(t, work, "payments")
	sn := anywhere()
	call(t, sn, "orbit_add_repo", map[string]any{"path": r.Path})

	if got := call(t, sn, "orbit_forget_repo", map[string]any{"repo": r.Path}); got["tasks_deleted"] != float64(0) {
		t.Errorf("orbit_forget_repo answered %v, want it to have deleted no tasks", got)
	}
	refs, err := s.Repos()
	if err != nil {
		t.Fatalf("Repos: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("orbit still knows %+v", refs)
	}
}

// Two checkouts of the same name is the ordinary state of a machine with a
// fork on it. Acting on whichever came first would be acting on a repository
// the caller did not name — and this is the tool that deletes records.
func TestForgetRepoRefusesTwoRepositoriesOfTheSameName(t *testing.T) {
	_, first := newRoot(t)
	one := gitRepo(t, first, "payments")
	two := gitRepo(t, t.TempDir(), "payments")
	sn := anywhere()
	call(t, sn, "orbit_add_repo", map[string]any{"path": one.Path})
	call(t, sn, "orbit_add_repo", map[string]any{"path": two.Path})

	said := refused(t, sn, "orbit_forget_repo", map[string]any{"repo": "payments"})
	for _, want := range []string{one.Path, two.Path} {
		if !strings.Contains(said, want) {
			t.Errorf("the refusal does not name %q: %s", want, said)
		}
	}
	if got := call(t, sn, "orbit_forget_repo", map[string]any{"repo": two.Path}); got["path"] != two.Path {
		t.Errorf("naming one by path answered %v, want %q", got["path"], two.Path)
	}
}
