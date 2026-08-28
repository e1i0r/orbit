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
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// newRoot roots a store in a temporary directory and points $ORBIT_HOME and
// git's two configuration files at throwaway places. The board runs git
// through repo.Discover, which inherits this process's environment on
// purpose, so it is this process's environment that has to be made safe:
// without this a test would read the developer's real state root and their
// real ~/.gitconfig, and would pass or fail depending on whose machine ran
// it.
//
// It answers the store beside the workspace, and they are two different
// directories on purpose — the record lives under $ORBIT_HOME and the
// checkouts do not. The workspace is what `orbit top` would be pointed at,
// what gitRepo builds repositories in, and what a Reader is opened over.
func newRoot(t *testing.T) (*store.Store, string) {
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

	return s, t.TempDir()
}

// gitRepo builds a real repository with one commit on it, because repo.Open
// reads the current branch and a repository with no commits has none.
//
// It goes under the workspace rather than in a temporary directory of its
// own, because the walk that finds it starts there: a repository somewhere
// else is a repository this board is not of.
func gitRepo(t *testing.T, work, name string) string {
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
	// The path git resolves to, and not the one the directory was made at.
	// Everything else in Orbit reaches a repository through repo.Open, which
	// resolves symlinks — and a temporary directory is behind one on macOS.
	// A fixture that answered the unresolved path would file its tasks under
	// a different hash from the one the walk makes the board look in, which
	// is a disagreement between two halves of the fixture and not a finding
	// about the code.
	opened, err := repo.Open(dir)
	if err != nil {
		t.Fatalf("open the repository at %q: %v", dir, err)
	}

	return opened.Path
}

// oneRepo is the usual setting: a fresh state root, a workspace, and one
// repository in it.
func oneRepo(t *testing.T) (s *store.Store, work, repoPath string) {
	t.Helper()
	s, work = newRoot(t)

	return s, work, gitRepo(t, work, "payments")
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
