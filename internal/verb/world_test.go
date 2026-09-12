package verb

// A world with somewhere to write and canned answers for everything else.
//
// The same verb has to work for a command line, a server and a tool call,
// so what a verb reaches for is ports — and a test with neither fills them
// here. The store is real: what a verb writes is the thing under test, and
// a fake store would be asserting against the test's own imagination.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// testWorld is internal/verb's ports over one throwaway state root.
type testWorld struct {
	store  *store.Store
	board  board.Board
	log    map[string][]view.Entry
	facts  []knowledge.Fact
	said   []saidLine
	learnt []knowledge.Fact
}

// saidLine is one line handed to the supervisor's thread.
type saidLine struct {
	text string
	by   string
	task string
}

func (w *testWorld) Store() *store.Store { return w.store }

func (w *testWorld) Words() *words.Printer { return words.For("") }

// Find is the task one id in one repository means, the way the command
// line reads it: the repository opens, and the task loads against it.
func (w *testWorld) Find(id, repoPath string) (task.Task, repo.Repo, error) {
	var one repo.Repo

	if repoPath != "" {
		opened, err := repo.Open(repoPath)
		if err != nil {
			return task.Task{}, repo.Repo{}, err
		}

		one = opened
	}

	if id == "" {
		return task.Task{}, one, nil
	}

	t, err := task.Load(w.store, one, id)
	if err != nil {
		return task.Task{}, one, err
	}

	return t, one, nil
}

func (w *testWorld) Unread(string) (int, error) { return 0, nil }

func (w *testWorld) Looked() error { return nil }

func (w *testWorld) Facts() ([]knowledge.Fact, error) { return w.facts, nil }

func (w *testWorld) Board() (board.Board, error) { return w.board, nil }

func (w *testWorld) Log(repoPath, id string) ([]view.Entry, error) {
	if w.log == nil {
		return nil, nil
	}

	return w.log[id], nil
}

func (w *testWorld) Deliver(_ context.Context, t task.Task, verb string) (string, error) {
	return "https://example.test/pr/1", nil
}

func (w *testWorld) Say(text, by, about string) error {
	w.said = append(w.said, saidLine{text: text, by: by, task: about})

	return nil
}

func (w *testWorld) Learn(fact knowledge.Fact) error {
	w.learnt = append(w.learnt, fact)

	return nil
}

func (w *testWorld) Export(into, only string) (string, error) {
	return "written to " + into, nil
}

func (w *testWorld) Take(id, repoPath string) (string, error) {
	return "a terminal in " + repoPath, nil
}

// worldOf is a world over a state root of the test's own.
func worldOf(t *testing.T) *testWorld {
	t.Helper()

	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	t.Cleanup(func() { _ = s.Close() })

	return &testWorld{store: s}
}

// gitRepo is a git repository with one commit in it, because a task is
// written against a checkout and several verbs read the checkout back.
func (w *testWorld) gitRepo(t *testing.T, name string) repo.Repo {
	t.Helper()

	return w.gitRepoIn(t, t.TempDir(), name)
}

// gitRepoIn is the same, in a directory the caller chose — so that two
// repositories can stand beside each other the way a workspace holds them.
func (w *testWorld) gitRepoIn(t *testing.T, ws, name string) repo.Repo {
	t.Helper()

	dir := filepath.Join(ws, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("make the repository directory: %v", err)
	}

	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.name", "t"},
		{"config", "user.email", "t@t"},
		{"commit", "-q", "--allow-empty", "-m", "first"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir

		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}

	one, err := repo.Open(dir)
	if err != nil {
		t.Fatalf("repo.Open: %v", err)
	}

	return one
}

// wrote puts a task on the board through the verb, the way every door
// asks it.
func (w *testWorld) wrote(t *testing.T, id, repoPath, text string) task.Task {
	t.Helper()

	out, err := Run(ctxOf(), w, "board new", In{
		Args: map[string]string{"id": id, "text": text, "repo": repoPath},
		By:   "operator",
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	if out.Of[0] != id {
		t.Fatalf("new answered %v, want it to act on %s", out.Of, id)
	}

	found, _, err := w.Find(id, repoPath)
	if err != nil {
		t.Fatalf("find %s: %v", id, err)
	}

	return found
}

// ctxOf is what a verb that reaches the network is given. A test has no
// deadline of its own, so this is the background and says so rather than
// inventing a timeout.
func ctxOf() context.Context { return context.Background() }

// holdARun lends a task a live process: a sleep with a marker naming it,
// the way a run holds its task while it works. Cancelling is asking that
// process to stop, so without one there is nothing to ask.
func holdARun(t *testing.T, w *testWorld, id string) *exec.Cmd {
	t.Helper()

	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start a background process: %v", err)
	}

	t.Cleanup(func() { _ = cmd.Process.Kill() }) //nolint:errcheck // the run is over either way

	path, err := w.store.RunPath(id)
	if err != nil {
		t.Fatalf("run marker path: %v", err)
	}

	body := fmt.Sprintf("pid: %d\nstarted: %s\n", cmd.Process.Pid, time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write the run marker: %v", err)
	}

	return cmd
}

// mustAsk runs one verb and fails the test on a refusal. Asking and
// refusing are both covered elsewhere; here the verb is meant to work.
func mustAsk(t *testing.T, w *testWorld, name string, in In) Out {
	t.Helper()

	out, err := Run(ctxOf(), w, name, in)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}

	return out
}

// mustRefuse runs one verb that is meant to refuse, and fails the test
// when it does not.
func mustRefuse(t *testing.T, w *testWorld, name string, in In) {
	t.Helper()

	if _, err := Run(ctxOf(), w, name, in); err == nil {
		t.Fatalf("%s was accepted, want a refusal", name)
	}
}

// refuseErr is mustRefuse for the caller that asserts on the refusal
// itself: what it says is the behavior under test.
func refuseErr(t *testing.T, w *testWorld, name string, in In) error {
	t.Helper()

	_, err := Run(ctxOf(), w, name, in)
	if err == nil {
		t.Fatalf("%s was accepted, want a refusal", name)
	}

	return err
}
