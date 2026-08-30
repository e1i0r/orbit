package mcp

// The setting every test in this package runs in: a state root of its own, a
// workspace with real git repositories in it, and a session pointed at that
// workspace rather than at wherever the test binary was started.
//
// HOME is redirected here as well as ORBIT_HOME. That is not tidiness: the
// installer this package ships writes into the home directory's Claude and
// Codex configuration, and the first version of these tests ran it against
// the real one — `go test ./...` rewrote a working orbit entry to point at a
// binary that does not exist. A test in this package must not be able to
// reach the reader's home even by mistake.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// newRoot answers a store beside a workspace, both of them temporary.
func newRoot(t *testing.T) (*store.Store, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("ORBIT_HOME", root)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

	s, err := store.New(root)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	return s, t.TempDir()
}

// gitRepo builds a real repository with one commit, because repo.Open reads
// the current branch and a repository with no commits has none.
func gitRepo(t *testing.T, work, name string) repo.Repo {
	t.Helper()

	dir := filepath.Join(work, name)
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatalf("mkdir %q: %v", dir, err)
	}

	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"commit", "-q", "--allow-empty", "-m", "first"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir

		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	// repo.Open's path and not the one the directory was made at: it
	// resolves symlinks, and a temporary directory is behind one on macOS.
	// A fixture answering the unresolved path files its tasks under a
	// different hash from the one the walk looks in.
	opened, err := repo.Open(dir)
	if err != nil {
		t.Fatalf("open the repository at %q: %v", dir, err)
	}

	return opened
}

// oneRepo is the usual setting: a state root, a workspace, one repository,
// and a session that looks at that workspace.
func oneRepo(t *testing.T) (*store.Store, Session, repo.Repo) {
	t.Helper()
	s, work := newRoot(t)

	return s, Session{Root: work, Version: "test"}, gitRepo(t, work, "payments")
}

// addTask writes a task's directory and its record straight to disk, which
// is how a test states a history that took a run to produce.
func addTask(t *testing.T, s *store.Store, r repo.Repo, id string, events ...record.Event) {
	t.Helper()

	if _, err := s.CreateTaskDir(r.Path, id); err != nil {
		t.Fatalf("create the directory of task %s: %v", id, err)
	}

	path, err := s.TaskFilePath(r.Path, id)
	if err != nil {
		t.Fatalf("task file path of %s: %v", id, err)
	}

	if err := os.WriteFile(path, []byte(id+" was written by a fixture\n"), 0o600); err != nil {
		t.Fatalf("write task %s: %v", id, err)
	}

	appendTo(t, s, r, id, events...)
}

func appendTo(t *testing.T, s *store.Store, r repo.Repo, id string, events ...record.Event) {
	t.Helper()

	path, err := s.EventsPath(r.Path, id)
	if err != nil {
		t.Fatalf("events path of task %s: %v", id, err)
	}

	for _, e := range events {
		if err := record.Append(path, e); err != nil {
			t.Fatalf("append %s to task %s: %v", e.Kind, id, err)
		}
	}
}

// holdTask plants the run marker that makes a task live.
//
// Live is the one column on a board row that does not come from the record:
// it is a pid file this process can answer for, so a test that wants a
// running task writes one naming itself rather than starting an engine.
func holdTask(t *testing.T, s *store.Store, r repo.Repo, id string) {
	t.Helper()

	path, err := s.RunPath(r.Path, id)
	if err != nil {
		t.Fatalf("run marker path of task %s: %v", id, err)
	}

	body := fmt.Sprintf("pid: %d\nstarted: %s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("plant the run marker of task %s: %v", id, err)
	}
}

// damageMarker leaves a run marker nothing can read, which is what a run
// killed mid-write leaves behind. It is the third answer to whether a
// process holds the task: not held, and not free either.
func damageMarker(t *testing.T, s *store.Store, r repo.Repo, id string) {
	t.Helper()

	path, err := s.RunPath(r.Path, id)
	if err != nil {
		t.Fatalf("run marker path of task %s: %v", id, err)
	}

	if err := os.WriteFile(path, []byte("pid: not a number\n"), 0o600); err != nil {
		t.Fatalf("damage the run marker of task %s: %v", id, err)
	}
}

// call runs one tool and decodes the JSON it answered with, failing the test
// if the tool refused. It is what makes the assertions below assertions
// rather than error handling.
func call(t *testing.T, sn Session, name string, args map[string]any) map[string]any {
	t.Helper()

	res := sn.Call(name, args)
	if res.IsError {
		t.Fatalf("%s refused: %s", name, text(t, res))
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(text(t, res)), &decoded); err != nil {
		t.Fatalf("%s answered something that is not a JSON object: %v\n%s", name, err, text(t, res))
	}

	return decoded
}

// refused runs one tool that is expected to say no, and answers what it said.
func refused(t *testing.T, sn Session, name string, args map[string]any) string {
	t.Helper()

	res := sn.Call(name, args)
	if !res.IsError {
		t.Fatalf("%s was expected to refuse and answered: %s", name, text(t, res))
	}

	return text(t, res)
}

// text is a result's one text block.
func text(t *testing.T, res CallToolResult) string {
	t.Helper()

	if len(res.Content) != 1 {
		t.Fatalf("a result carried %d content blocks, want exactly 1", len(res.Content))
	}

	if res.Content[0].Type != "text" {
		t.Fatalf("a result carried a %q block, want text", res.Content[0].Type)
	}

	return res.Content[0].Text
}

// obj, list and str read one field out of a decoded answer, failing the test
// when it is not the shape the tool promised. They exist so that a test
// walking three levels of an answer reads as the assertion it is rather than
// as nine lines of type switching — and so that a field that came back wrong
// stops the test where it went wrong instead of travelling on as a zero
// value into an assertion about something else.
func obj(t *testing.T, v any) map[string]any {
	t.Helper()

	got, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("want an object, got %#v", v)
	}

	return got
}

func list(t *testing.T, v any) []any {
	t.Helper()

	got, ok := v.([]any)
	if !ok {
		t.Fatalf("want an array, got %#v", v)
	}

	return got
}

func str(t *testing.T, v any) string {
	t.Helper()

	got, ok := v.(string)
	if !ok {
		t.Fatalf("want a string, got %#v", v)
	}

	return got
}
