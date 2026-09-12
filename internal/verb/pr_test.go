package verb

// The pull request family, asked for the way every way in asks for it.

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

// prWorld is a world with a task, a store and a thread to say into. The
// deliver verbs reach the checkout, the record and the reader's language
// and no further, so a world that answered more would be saying this test
// covers more than it does.
type prWorld struct {
	World

	store *store.Store
	task  task.Task
	said  []string
}

func (w *prWorld) Store() *store.Store   { return w.store }
func (w *prWorld) Words() *words.Printer { return words.For("") }

func (w *prWorld) Find(id, _ string) (task.Task, repo.Repo, error) {
	return w.task, repo.Repo{}, nil
}

func (w *prWorld) Say(text, _, _ string) error {
	w.said = append(w.said, text)

	return nil
}

func workedTask(t *testing.T, s *store.Store, id string) task.Task {
	t.Helper()

	wt, err := s.WorktreeDir("/src/acme", id)
	if err != nil {
		t.Fatalf("worktree dir: %v", err)
	}

	if err := os.MkdirAll(wt, 0o755); err != nil {
		t.Fatalf("make the checkout: %v", err)
	}

	return task.Task{ID: id, Repo: repo.Repo{Path: "/src/acme", Name: "acme"}}
}

func prOf(t *testing.T, id string) *prWorld {
	t.Helper()

	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	t.Cleanup(func() { _ = s.Close() })

	return &prWorld{store: s, task: workedTask(t, s, id)}
}

func askPR(t *testing.T, w *prWorld, name string) (Out, error) {
	t.Helper()

	return Run(context.Background(), w, name, In{Task: w.task.ID, By: "operator", Door: "the command line"})
}

// TestAnErrandGoesToTheSupervisorWithWhereItWorks. The ask carries the
// task, the checkout and the door it came through, under the brief that
// says what the supervisor may not do.
func TestAnErrandGoesToTheSupervisorWithWhereItWorks(t *testing.T) {
	for name, body := range map[string]string{
		"pr update":  "Bring this task's branch up to date",
		"pr checks":  "Make the checks on this task's pull request pass",
		"pr tests":   "Raise this task's tests",
		"pr resolve": "Answer the review comments",
		"pr review":  "Review this task's pull request",
	} {
		w := prOf(t, "ACME-1")

		out, err := askPR(t, w, name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		if len(w.said) != 1 {
			t.Fatalf("%s said %d lines to the thread, want one", name, len(w.said))
		}

		for _, want := range []string{"ACME-1", "the command line", "Never force-push", body} {
			if !strings.Contains(w.said[0], want) {
				t.Errorf("%s did not hand the supervisor %q", name, want)
			}
		}

		if !strings.Contains(out.Said, "ACME-1") {
			t.Errorf("%s answered %q, which names no task", name, out.Said)
		}
	}
}

// TestAnErrandWithNowhereToWorkIsRefused. A task that never ran has no
// checkout, and the supervisor sent there would work in the state root.
func TestAnErrandWithNowhereToWorkIsRefused(t *testing.T) {
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	t.Cleanup(func() { _ = s.Close() })

	w := &prWorld{store: s, task: task.Task{ID: "ACME-2"}}

	if _, err := askPR(t, w, "pr update"); err == nil {
		t.Error("an update with no checkout was accepted")
	} else if !strings.Contains(err.Error(), "ACME-2") {
		t.Errorf("the refusal is %q, which names no task", err)
	}

	if len(w.said) != 0 {
		t.Error("the refused ask still reached the thread")
	}
}

// TestShowReadsWhatOpeningWrote. One row per opening, newest first, with
// what became of each.
func TestShowReadsWhatOpeningWrote(t *testing.T) {
	w := prOf(t, "ACME-1")

	out, err := askPR(t, w, "pr show")
	if err != nil {
		t.Fatalf("pr show: %v", err)
	}

	// Nothing opened yet: an empty reading, not a refusal.
	if !strings.Contains(out.Said, "ACME-1") {
		t.Errorf("an empty show reads %q", out.Said)
	}

	made, err := task.Create(w.store, repo.Repo{Path: "/src/acme", Name: "acme"}, "ACME-1", "Pay the thing", "")
	if err != nil {
		t.Fatalf("write the task down: %v", err)
	}

	if _, err := task.Join(w.store, made, repo.Repo{Path: "/src/acme", Name: "acme"}); err != nil {
		t.Fatalf("join the repository: %v", err)
	}

	if err := w.store.OpenedPR("ACME-1", "/src/acme", "https://github.test/acme/pull/1"); err != nil {
		t.Fatalf("open a pull request: %v", err)
	}

	out, err = askPR(t, w, "pr show")
	if err != nil {
		t.Fatalf("pr show: %v", err)
	}

	for _, want := range []string{"acme", "open", "https://github.test/acme/pull/1"} {
		if !strings.Contains(out.Said, want) {
			t.Errorf("the show does not mention %q:\n%s", want, out.Said)
		}
	}
}
