package board

// The state root a test builds, and the things put into it. It is separate
// from refresh_test.go for the same reason internal/view keeps its case
// tables out of its assertions: a fixture read while checking an assertion
// is a fixture nobody reads.

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// newRoot roots a store in a temporary directory and points $ORBIT_HOME and
// git's two configuration files at throwaway places. The board runs git
// through repo.Open, which inherits this process's environment on purpose,
// so it is this process's environment that has to be made safe: without
// this a test would read the developer's real state root and their real
// ~/.gitconfig, and would pass or fail depending on whose machine ran it.
func newRoot(t *testing.T) *store.Store {
	t.Helper()
	root := t.TempDir()
	t.Setenv("ORBIT_HOME", root)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	s, err := store.New(root)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	return s
}

// gitRepo builds a real repository with one commit on it, because repo.Open
// reads the current branch and a repository with no commits has none.
func gitRepo(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
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
	return dir
}

// oneRepo is the usual setting: a fresh state root and one repository.
func oneRepo(t *testing.T) (*store.Store, string) {
	t.Helper()
	s := newRoot(t)
	return s, gitRepo(t, "payments")
}

// refresh polls once and fails the test if it could not be done. Every test
// here that expects a board gets one this way, so the assertions below are
// assertions and not error handling.
func refresh(t *testing.T, r *Reader) (Board, Changed) {
	t.Helper()
	b, changed, err := r.Refresh()
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	return b, changed
}

func addTask(t *testing.T, s *store.Store, repoPath, id string, events ...record.Event) {
	t.Helper()
	if _, err := s.CreateTaskDir(repoPath, id); err != nil {
		t.Fatalf("create the directory of task %s: %v", id, err)
	}
	appendTo(t, s, repoPath, id, events...)
}

func appendTo(t *testing.T, s *store.Store, repoPath, id string, events ...record.Event) {
	t.Helper()
	for _, e := range events {
		if err := record.Append(eventsPath(t, s, repoPath, id), e); err != nil {
			t.Fatalf("append %s to task %s: %v", e.Kind, id, err)
		}
	}
}

func eventsPath(t *testing.T, s *store.Store, repoPath, id string) string {
	t.Helper()
	path, err := s.EventsPath(repoPath, id)
	if err != nil {
		t.Fatalf("events path of task %s: %v", id, err)
	}
	return path
}

func sizeOf(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %q: %v", path, err)
	}
	return info.Size()
}

// poison overwrites the first n bytes of a log with a character no JSON
// document starts with, leaving the newlines — and therefore the file's
// length — alone.
func poison(t *testing.T, path string, n int64) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	for i := range body[:n] {
		if body[i] != '\n' {
			body[i] = 'x'
		}
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

// tooLongLine appends a line longer than the reader's buffer, which is the
// one damage record.ReadFrom reports as an error rather than folding into a
// record.unreadable event. record.Append refuses to write one, so it is
// written here by hand.
func tooLongLine(t *testing.T, path string) {
	t.Helper()
	line := make([]byte, record.MaxLine+1)
	for i := range line {
		line[i] = 'x'
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open %q: %v", path, err)
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		t.Fatalf("write a line nobody can read back: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close %q: %v", path, err)
	}
}

func at(n int) time.Time { return time.Date(2026, 8, 23, 9, 0, n, 0, time.UTC) }

func created(title string) record.Event {
	return record.Event{At: at(0), Kind: "task.created", Text: title}
}

func startedEvent() record.Event {
	return record.Event{At: at(1), Kind: "task.started", Data: map[string]string{"flow": "task"}}
}

func failedEvent() record.Event { return record.Event{At: at(2), Kind: "task.failed"} }

func finishedEvent() record.Event { return record.Event{At: at(3), Kind: "task.finished"} }
