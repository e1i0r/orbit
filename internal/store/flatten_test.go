package store

// The move to the flat tree, read from the side that matters: what came up,
// what stayed where it was, and what it refuses to decide on its own.

import (
	"os"
	"path/filepath"
	"testing"
)

// filedTask writes a task the old way — under the repository — and answers
// the directory it wrote.
func filedTask(t *testing.T, s *Store, repoPath, id string, files map[string]string) string {
	t.Helper()

	if _, err := s.RegisterRepo(repoPath); err != nil {
		t.Fatalf("RegisterRepo: %v", err)
	}

	repoDir, err := s.RepoDir(repoPath)
	if err != nil {
		t.Fatalf("RepoDir: %v", err)
	}

	dir := filepath.Join(repoDir, "tasks", id)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), fileMode); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	return dir
}

func TestFlattenCopiesATaskUpAndLeavesTheOldOneWhereItIs(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	log := `{"at":"2026-08-01T10:00:00Z","kind":"task.created","text":"retry the webhook on 5xx"}` + "\n"
	was := filedTask(t, s, "/w/app", "ACME-1", map[string]string{
		"task.md":      "retry the webhook on 5xx\n",
		"events.jsonl": log,
		"control":      "pause",
	})

	moved, err := s.Flatten()
	if err != nil {
		t.Fatalf("Flatten: %v", err)
	}

	if len(moved) != 1 || moved[0] != "ACME-1" {
		t.Fatalf("Flatten moved %v, want ACME-1", moved)
	}

	path, err := s.EventsPath("ACME-1")
	if err != nil {
		t.Fatalf("EventsPath: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the log did not arrive: %v", err)
	}

	if string(body) != log {
		t.Errorf("the log came up as %q, want %q", body, log)
	}

	// Nothing was deleted. The old tree is the only other copy of a record
	// there is, and a migration that ate one would be unanswerable.
	if _, err := os.Stat(filepath.Join(was, "events.jsonl")); err != nil {
		t.Errorf("the log where it used to be: %v", err)
	}

	// And the task now says where it is worked, which is what the path used
	// to say.
	joined, err := s.TaskRepos("ACME-1")
	if err != nil {
		t.Fatalf("TaskRepos: %v", err)
	}

	if len(joined) != 1 || joined[0] != "/w/app" {
		t.Errorf("the task names the repositories %v, want /w/app", joined)
	}

	// And it is one of the state root's tasks, which is the listing the
	// migration walks: a flattened task the migration cannot see is a task
	// whose record never reaches the database.
	ids, err := s.TaskIDs()
	if err != nil {
		t.Fatalf("TaskIDs: %v", err)
	}

	if len(ids) != 1 || ids[0] != "ACME-1" {
		t.Errorf("the state root holds %v, want ACME-1", ids)
	}
}

// The second start finds everything in place. It is called before every
// command, so "already done" has to be nothing at all rather than an error
// or a second copy.
func TestFlatteningTwiceMovesNothingTheSecondTime(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	filedTask(t, s, "/w/app", "ACME-1", map[string]string{"task.md": "one\n"})

	if _, err := s.Flatten(); err != nil {
		t.Fatalf("Flatten: %v", err)
	}

	// Somebody keeps working after the move, and the second run must not
	// write over what they wrote.
	path, err := s.TaskFilePath("ACME-1")
	if err != nil {
		t.Fatalf("TaskFilePath: %v", err)
	}

	if err := os.WriteFile(path, []byte("one, and then more\n"), fileMode); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	moved, err := s.Flatten()
	if err != nil {
		t.Fatalf("Flatten again: %v", err)
	}

	if len(moved) != 0 {
		t.Errorf("the second run moved %v, want nothing", moved)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if string(body) != "one, and then more\n" {
		t.Errorf("the task now reads %q, and the second run wrote over it", body)
	}
}

// Two repositories, one name. A flat tree holds one of them, and which one
// is not a question a migration answers while nobody is looking.
func TestFlattenRefusesToChooseBetweenTwoTasksOfTheSameName(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	filedTask(t, s, "/w/app", "ACME-1", map[string]string{"task.md": "the one in app\n"})
	filedTask(t, s, "/w/api", "ACME-1", map[string]string{"task.md": "the one in api\n"})

	moved, err := s.Flatten()
	if err == nil {
		t.Fatal("Flatten said nothing about two tasks named ACME-1")
	}

	if len(moved) != 1 {
		t.Errorf("Flatten moved %v, want the first of the two and no more", moved)
	}

	// The one it would not move is still readable where it always was, and
	// the error names it so somebody can rename it.
	if !filepath.IsAbs(s.TasksDir()) {
		t.Fatal("the tasks directory is not absolute")
	}

	joined, err := s.TaskRepos("ACME-1")
	if err != nil {
		t.Fatalf("TaskRepos: %v", err)
	}

	if len(joined) != 1 {
		t.Errorf("the task that moved names %v, and the collision was folded into it", joined)
	}
}

// The reason the tree is flat: one task, worked in two repositories, and
// both of them list it. The old tree could not hold that at all.
func TestATaskCanBeWorkedInTwoRepositories(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, repoPath := range []string{"/w/app", "/w/api"} {
		if _, err := s.CreateTaskDir(repoPath, "ACME-1"); err != nil {
			t.Fatalf("CreateTaskDir %s: %v", repoPath, err)
		}
	}

	joined, err := s.TaskRepos("ACME-1")
	if err != nil {
		t.Fatalf("TaskRepos: %v", err)
	}

	if len(joined) != 2 || joined[0] != "/w/app" || joined[1] != "/w/api" {
		t.Fatalf("the task names %v, want both repositories in the order they joined", joined)
	}

	// Joining twice is joining once: the marker is a list of who took part,
	// not a count of how often.
	if _, err := s.CreateTaskDir("/w/app", "ACME-1"); err != nil {
		t.Fatalf("CreateTaskDir again: %v", err)
	}

	again, err := s.TaskRepos("ACME-1")
	if err != nil {
		t.Fatalf("TaskRepos: %v", err)
	}

	if len(again) != 2 {
		t.Errorf("after joining twice the task names %v", again)
	}
}
