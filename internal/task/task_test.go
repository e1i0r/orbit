package task

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

func fixture(t *testing.T) (*store.Store, repo.Repo) {
	t.Helper()

	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	dir := filepath.Join(t.TempDir(), "src")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	home := t.TempDir()

	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"commit", "-q", "--allow-empty", "-m", "first"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir

		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
			// The developer's real ~/.gitconfig must never leak into the
			// test: a global commit.gpgsign or hooksPath would make this
			// suite pass or fail depending on whose machine runs it.
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
			"HOME="+home,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	r, err := repo.Open(dir)
	if err != nil {
		t.Fatalf("repo.Open: %v", err)
	}

	return s, r
}

func TestCreateWritesTheTaskAndAnEvent(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "retry the webhook on 5xx", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	dir, err := s.TaskDir("ACME-1")
	if err != nil {
		t.Fatalf("TaskDir: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(dir, "task.md"))
	if err != nil {
		t.Fatalf("read task.md: %v", err)
	}

	if string(body) != "retry the webhook on 5xx\n" {
		t.Errorf("task.md = %q", body)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	if len(events) != 1 || events[0].Kind != "task.created" {
		t.Errorf("events = %+v", events)
	}
}

func TestCreateRefusesADuplicate(t *testing.T) {
	s, r := fixture(t)
	if _, err := Create(s, r, "ACME-1", "one", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := Create(s, r, "ACME-1", "two", ""); err == nil {
		t.Error("a second task with the same id was created, overwriting the first")
	}
}

func TestLoadReturnsWhatWasCreated(t *testing.T) {
	s, r := fixture(t)
	if _, err := Create(s, r, "ACME-1", "retry the webhook on 5xx", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	tk, err := Load(s, r, "ACME-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if tk.ID != "ACME-1" || tk.Text != "retry the webhook on 5xx" || tk.Repo.Path != r.Path {
		t.Errorf("Load = %+v", tk)
	}
}

func TestLoadFailsOnATaskThatWasNeverCreated(t *testing.T) {
	s, r := fixture(t)
	if _, err := Load(s, r, "ACME-404"); err == nil {
		t.Error("Load succeeded for a task that was never created")
	}
}

func TestListOfARepositoryWithNoTasksIsEmptyNotAnError(t *testing.T) {
	s, r := fixture(t)

	got, err := List(s, r)
	if err != nil {
		t.Fatalf("List on a repository nobody has written a task against: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("List = %v, want nothing at all", got)
	}
}

func TestListDoesNotInventATaskFromAMistypedID(t *testing.T) {
	s, r := fixture(t)
	if _, err := Create(s, r, "ACME-1", "x", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Someone typed the id wrong. Nothing about looking may leave a mark.
	if _, err := Load(s, r, "ACME-l"); err == nil {
		t.Fatal("Load succeeded for a task that was never created")
	}

	if _, err := Events(s, Task{ID: "ACME-2", Repo: r}); err != nil {
		t.Fatalf("Events: %v", err)
	}

	got, err := List(s, r)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(got) != 1 || got[0] != "ACME-1" {
		t.Errorf("List = %v, want [ACME-1] — reading a task that does not exist minted a directory", got)
	}
}

func TestListReturnsTheTaskIDs(t *testing.T) {
	s, r := fixture(t)
	for _, id := range []string{"ACME-2", "ACME-1"} {
		if _, err := Create(s, r, id, "x", ""); err != nil {
			t.Fatalf("Create %s: %v", id, err)
		}
	}

	got, err := List(s, r)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(got) != 2 || got[0] != "ACME-1" || got[1] != "ACME-2" {
		t.Errorf("List = %v, want [ACME-1 ACME-2] in order", got)
	}
}

// blockTheLog makes the task's events file impossible to write by putting a
// directory where the file has to go. It is the cheapest honest way to make
// a record fail: no permissions to fake, no interface to inject, and it
// behaves the same on every filesystem this runs on.
func blockTheLog(t *testing.T, s *store.Store, r repo.Repo, id string) string {
	t.Helper()

	dir, err := s.CreateTaskDir(r.Path, id)
	if err != nil {
		t.Fatalf("CreateTaskDir: %v", err)
	}

	blocked := filepath.Join(dir, "events.jsonl")
	if err := os.Mkdir(blocked, 0o700); err != nil {
		t.Fatalf("mkdir %q: %v", blocked, err)
	}

	return blocked
}

func TestCreateTakesTheFileBackOutWhenNothingCouldBeRecorded(t *testing.T) {
	s, r := fixture(t)
	blocked := blockTheLog(t, s, r, "ACME-1")

	_, err := Create(s, r, "ACME-1", "one", "")
	if err == nil {
		t.Fatal("Create reported success with nothing in the record")
	}

	if !strings.Contains(err.Error(), "ACME-1") || !strings.Contains(err.Error(), "task.created") {
		t.Errorf("the error does not name both what failed and which task: %v", err)
	}

	path, err := s.TaskFilePath("ACME-1")
	if err != nil {
		t.Fatalf("TaskFilePath: %v", err)
	}

	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("task.md was left behind by a create that failed: %v", statErr)
	}

	// The point of taking it back out: the retry works. Left there, the task
	// would exist to nothing but the duplicate check, which would refuse to
	// write it ever again.
	if err := os.Remove(blocked); err != nil {
		t.Fatalf("unblock: %v", err)
	}

	if _, err := Create(s, r, "ACME-1", "one", ""); err != nil {
		t.Fatalf("the retry was refused after a rolled-back create: %v", err)
	}
}

// TestCreateTakesTheDirectoryBackOutTooSoListForgetsIt covers the half of
// the rollback blockTheLog cannot reach: blockTheLog leaves an events.jsonl
// directory sitting inside the task directory, which is exactly the case
// the non-recursive removal is required to leave alone, so the task
// directory survives that test on purpose. Here the record fails for a
// reason that leaves nothing behind — record.Append refuses a line over
// its MaxLine before it creates anything on disk — so the task directory
// holds only task.md right up until the rollback removes that too, and the
// directory itself comes out empty and removable.
func TestCreateTakesTheDirectoryBackOutTooSoListForgetsIt(t *testing.T) {
	s, r := fixture(t)
	oversized := strings.Repeat("x", 5<<20) // over record.MaxLine (4 MiB)

	if _, err := Create(s, r, "ACME-1", oversized, ""); err == nil {
		t.Fatal("Create reported success with nothing in the record")
	}

	// task.md is gone; if the directory it lived in survived, List would
	// still report a task that was never recorded — `orbit list` naming
	// something `orbit show` has nothing to say about.
	got, err := List(s, r)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("List = %v, want nothing at all — ACME-1 was never recorded", got)
	}

	dir, err := s.TaskDir("ACME-1")
	if err != nil {
		t.Fatalf("TaskDir: %v", err)
	}

	if _, statErr := os.Stat(dir); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("the task directory was left behind by a create that failed: %v", statErr)
	}
}

// TestADuplicateIsTellableFromEveryOtherFailure is why ErrExists is a
// sentinel: "you have used this id before" is a mistake a human fixes by
// picking another id, and a state root that will not take the file is not.
// The command line has to tell them apart without reading the words of an
// error message.
func TestADuplicateIsTellableFromEveryOtherFailure(t *testing.T) {
	s, r := fixture(t)
	if _, err := Create(s, r, "ACME-1", "one", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err := Create(s, r, "ACME-1", "two", "")
	if !errors.Is(err, ErrExists) {
		t.Errorf("a duplicate id did not come back as ErrExists: %v", err)
	}

	blockTheLog(t, s, r, "ACME-2")

	if _, err := Create(s, r, "ACME-2", "one", ""); errors.Is(err, ErrExists) {
		t.Errorf("a record that could not be written came back as ErrExists: %v", err)
	}
}
