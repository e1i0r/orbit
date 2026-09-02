package task

// The task that starts in no repository at all.
//
// It is the same doctrine as the fourth repository joining on the fourth
// phase, taken to the first: a repository is on a task because Orbit opened
// a worktree of it for that task, and a task nobody has opened one for yet
// is a task with no repository. The reader who writes one has not decided
// where the work goes; the run finds out.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// nowhere is a task written against no repository, and the store it is in.
func nowhere(t *testing.T, id, text string) (*store.Store, Task) {
	t.Helper()

	s, _ := fixture(t)

	tk, err := Create(s, repo.Repo{}, id, text, "")
	if err != nil {
		t.Fatalf("Create with no repository: %v", err)
	}

	return s, tk
}

// TestATaskCanBeWrittenAgainstNoRepository. The written sentence is the
// whole of what a task is, and where the work goes is a thing the work
// decides. A form that would not take the sentence without a checkout to
// file it under was asking the reader a question the run answers.
func TestATaskCanBeWrittenAgainstNoRepository(t *testing.T) {
	s, tk := nowhere(t, "ACME-1", "find out which service owns the retry")

	where, err := s.TaskRepos(tk.ID)
	if err != nil {
		t.Fatalf("TaskRepos: %v", err)
	}

	if len(where) != 0 {
		t.Errorf("a task written against nothing is worked in %v", where)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	if len(events) != 1 || events[0].Kind != record.TaskCreated {
		t.Fatalf("the record holds %v, want one task.created", events)
	}

	// Not "repo": "". A reader of the record asks whether the key is there,
	// and an empty one answers yes to a question whose answer is no.
	for _, key := range []string{"repo", "path"} {
		if named, ok := events[0].Data[key]; ok {
			t.Errorf("task.created says %s=%q for a task that has none", key, named)
		}
	}
}

// TestAPhaseOfATaskWithNoRepositoryRunsSomewhere. There is no worktree to
// open, so the run gets a directory of its own under the state root — and
// the prompt says plainly that the task is in no repository rather than
// naming an empty one.
func TestAPhaseOfATaskWithNoRepositoryRunsSomewhere(t *testing.T) {
	s, tk := nowhere(t, "ACME-2", "find out which service owns the retry")

	fake := engine.NewFake("looked")

	f := flow.Flow{Name: "one", Phases: []flow.Phase{{Name: "survey", Engine: "fake"}}}
	if err := Run(context.Background(), s, tk, f, map[string]engine.Engine{"fake": fake}, nil); err != nil {
		t.Fatalf("Run a task with no repository: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("the engine was called %d times, want once", len(fake.Calls))
	}

	dir, err := s.TaskDir(tk.ID)
	if err != nil {
		t.Fatalf("TaskDir: %v", err)
	}

	if want := filepath.Join(dir, "work"); fake.Calls[0].Dir != want {
		t.Errorf("the phase ran in %q, want %q", fake.Calls[0].Dir, want)
	}

	if _, err := os.Stat(fake.Calls[0].Dir); err != nil {
		t.Errorf("the phase was told to run somewhere that is not there: %v", err)
	}

	if said := fake.Calls[0].Prompt; !strings.Contains(said, "not being worked in any repository yet") {
		t.Errorf("the prompt does not say the task is in no repository:\n%s", said)
	}
}

// TestTheFirstRepositoryJoinsTheSameWayTheFourthDoes. This is the whole of
// what the ticket asks for: the task starts nowhere, the work turns out to
// be in the app, and the road the app takes onto the task is the road every
// later repository takes — one repo.joined, written when there is a checkout
// to work in.
func TestTheFirstRepositoryJoinsTheSameWayTheFourthDoes(t *testing.T) {
	s, tk := nowhere(t, "ACME-3", "find out which service owns the retry")

	_, repos := workspaceFixture(t, "app")

	wt, err := Join(s, tk, repos[0])
	if err != nil {
		t.Fatalf("Join the first repository: %v", err)
	}

	if _, err := os.Stat(wt); err != nil {
		t.Errorf("the checkout the join answered is not there: %v", err)
	}

	if got := joins(t, s, tk); len(got) != 1 || got[0] != "app" {
		t.Errorf("the record says %v joined, want app once", got)
	}

	where, err := s.TaskRepos(tk.ID)
	if err != nil {
		t.Fatalf("TaskRepos: %v", err)
	}

	if len(where) != 1 || where[0] != repos[0].Path {
		t.Errorf("the task is worked in %v, want %q", where, repos[0].Path)
	}
}

// TestATaskWithNoRepositoryIsOfferedTheOnesOrbitKnows. The names in the
// prompt are what `orbit join` takes, so the two have to come from one
// place: a phase offered a name that will not resolve is a phase that fails
// on its own instruction.
func TestATaskWithNoRepositoryIsOfferedTheOnesOrbitKnows(t *testing.T) {
	s, tk := nowhere(t, "ACME-4", "find out which service owns the retry")

	_, repos := workspaceFixture(t, "app")

	// Known because a task was run in it, which is the only way the state
	// root learns about a checkout that nothing has been written against.
	if _, err := s.RegisterRepo(repos[0].Path); err != nil {
		t.Fatalf("RegisterRepo: %v", err)
	}

	// The workspace is the fixture's, and this task has no repository to
	// take a parent from; without $ORBIT_WORKSPACE the names come from the
	// record instead.
	t.Setenv(repo.WorkspaceEnv, "")

	names := elsewhere(s, tk)
	if len(names) != 1 || names[0] != "app" {
		t.Fatalf("the phase is offered %v, want app", names)
	}

	found, err := Joinable(s, repo.Repo{}, "app")
	if err != nil {
		t.Fatalf("the name the prompt offered would not resolve: %v", err)
	}

	if found.Path != repos[0].Path {
		t.Errorf("join app answered %q, want %q", found.Path, repos[0].Path)
	}
}
