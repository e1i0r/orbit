package task

// join_test.go is a repository joining a task by being worked in: the
// checkout, the record of it, and what a task that reaches into three of
// them looks like afterwards.

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// workspaceFixture is a state root and a directory holding one repository
// per name, side by side, which is the arrangement repo.Workspace reads.
func workspaceFixture(t *testing.T, names ...string) (*store.Store, []repo.Repo) {
	t.Helper()

	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	ws, home := t.TempDir(), t.TempDir()

	repos := make([]repo.Repo, 0, len(names))

	for _, name := range names {
		dir := filepath.Join(ws, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", dir, err)
		}

		for _, args := range [][]string{
			{"init", "-q", "-b", "main"},
			{"commit", "-q", "--allow-empty", "-m", "first"},
		} {
			cmd := exec.Command("git", args...)
			cmd.Dir = dir
			cmd.Env = append(cmd.Environ(),
				"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
				"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
				"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
				"HOME="+home,
			)

			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git %v in %q: %v\n%s", args, dir, err, out)
			}
		}

		r, err := repo.Open(dir)
		if err != nil {
			t.Fatalf("repo.Open %q: %v", dir, err)
		}

		repos = append(repos, r)
	}

	return s, repos
}

// joins is which repositories the record says a task reaches into, in the
// order they joined.
func joins(t *testing.T, s *store.Store, tk Task) []string {
	t.Helper()

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	var names []string

	for _, e := range events {
		if e.Kind == record.RepoJoined {
			names = append(names, e.Data["repo"])
		}
	}

	return names
}

// TestATaskThatWorksInThreeRepositoriesHasThree, and nobody named one of
// them. The set was not declared at the start and no phase confirmed it: it
// is what it is because three checkouts were opened, which is the whole of
// how a task's scope is decided.
func TestATaskThatWorksInThreeRepositoriesHasThree(t *testing.T) {
	s, repos := workspaceFixture(t, "api", "app", "scripts")

	tk, err := Create(s, repos[0], "ACME-1", "retry the webhook on 5xx", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	for _, r := range repos {
		wt, err := Join(s, tk, r)
		if err != nil {
			t.Fatalf("Join %q: %v", r.Name, err)
		}

		if _, err := os.Stat(filepath.Join(wt, ".git")); err != nil {
			t.Errorf("joining %q left no checkout at %q: %v", r.Name, wt, err)
		}
	}

	joined, err := s.TaskRepos("ACME-1")
	if err != nil {
		t.Fatalf("TaskRepos: %v", err)
	}

	if len(joined) != 3 {
		t.Errorf("the task reaches into %d repositories, want 3: %v", len(joined), joined)
	}

	// api joined by being written against and the other two by being worked
	// in, and the record does not tell them apart — task.created carries the
	// first, repo.joined the rest, and both make the same link.
	if got := joins(t, s, tk); len(got) != 2 || got[0] != "app" || got[1] != "scripts" {
		t.Errorf("the record's joins are %v, want app then scripts", got)
	}
}

// TestARepositoryThatJoinsLateJoinsTheSameWay. A task discovers on its
// fourth phase that the API needs a change too, and what happens then has to
// be what would have happened on the first: the same checkout, the same
// event, the same listing afterwards.
func TestARepositoryThatJoinsLateJoinsTheSameWay(t *testing.T) {
	s, repos := workspaceFixture(t, "app", "api")

	tk, err := Create(s, repos[0], "ACME-2", "add the retry", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Three phases' worth of work in the repository it started in.
	for range 3 {
		if _, err := prepare(s, tk); err != nil {
			t.Fatalf("prepare: %v", err)
		}
	}

	late, err := Join(s, tk, repos[1])
	if err != nil {
		t.Fatalf("Join api: %v", err)
	}

	// The worktree is filed under the repository it is of, which is what
	// makes a fourth-phase join need no new place to live: the layout was
	// already keyed by both repository and task.
	want, err := s.WorktreeDir(repos[1].Path, "ACME-2")
	if err != nil {
		t.Fatalf("WorktreeDir: %v", err)
	}

	if late != want {
		t.Errorf("the late checkout is at %q, want %q", late, want)
	}

	if _, err := os.Stat(filepath.Join(late, ".git")); err != nil {
		t.Errorf("the late join left no checkout at %q: %v", late, err)
	}

	if got := joins(t, s, tk); len(got) != 1 || got[0] != "api" {
		t.Errorf("the record's joins are %v, want api once", got)
	}
}

// TestJoiningARepositoryTwiceIsOneJoin. Every phase of a run opens the
// worktree it works in, so the second phase would say "app joined" again on
// a task whose scope has not changed. The checkout is reused and the record
// is left alone.
func TestJoiningARepositoryTwiceIsOneJoin(t *testing.T) {
	s, repos := workspaceFixture(t, "app")

	tk, err := Create(s, repos[0], "ACME-3", "add the retry", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	first, err := Join(s, tk, repos[0])
	if err != nil {
		t.Fatalf("Join: %v", err)
	}

	second, err := Join(s, tk, repos[0])
	if err != nil {
		t.Fatalf("Join again: %v", err)
	}

	if first != second {
		t.Errorf("joining twice gave two checkouts, %q and %q", first, second)
	}

	if got := joins(t, s, tk); len(got) != 0 {
		t.Errorf("the record says %v joined, want nothing said twice", got)
	}
}

// TestANameNoRepositoryAnswersToIsRefusedWithTheOnesThatWould. A model that
// asks for "backend" in a workspace of api and app is told what is there
// rather than given the nearest match: work committed in a repository that
// was not part of the task is a mistake nobody finds by reading the record.
func TestANameNoRepositoryAnswersToIsRefusedWithTheOnesThatWould(t *testing.T) {
	_, repos := workspaceFixture(t, "api", "app")

	found, err := Joinable(repos[0], "app")
	if err != nil {
		t.Fatalf("Joinable app: %v", err)
	}

	if found.Path != repos[1].Path {
		t.Errorf("Joinable(app) = %q, want %q", found.Path, repos[1].Path)
	}

	_, err = Joinable(repos[0], "backend")
	if err == nil {
		t.Fatal("Joinable answered a name no repository has")
	}

	for _, want := range []string{"backend", "api", "app"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal %q does not name %q", err, want)
		}
	}
}

// TestARunTellsItsEngineWhichTaskItIsRunning, so that the command that joins
// a repository takes one word rather than an id the model has to copy off
// the prompt correctly.
func TestARunTellsItsEngineWhichTaskItIsRunning(t *testing.T) {
	_, repos := workspaceFixture(t, "app")

	env := childEnv(Task{ID: "ACME-4", Repo: repos[0]})

	for _, want := range []string{
		IDEnv + "=ACME-4",
		repo.WorkspaceEnv + "=" + filepath.Dir(repos[0].Path),
	} {
		if !slices.Contains(env, want) {
			t.Errorf("the phase's environment %v does not carry %q", env, want)
		}
	}
}

// TestAWorkspaceWithNothingInItSaysThatRatherThanListingNothing. The refusal
// is read by a model deciding what to do next, and "no repository called api
// in /x, which holds []" reads as a listing that failed rather than a
// workspace that is empty.
func TestAWorkspaceWithNothingInItSaysThatRatherThanListingNothing(t *testing.T) {
	_, repos := workspaceFixture(t, "app")
	t.Setenv(repo.WorkspaceEnv, t.TempDir())

	_, err := Joinable(repos[0], "api")
	if err == nil {
		t.Fatal("Joinable answered out of an empty workspace")
	}

	if !strings.Contains(err.Error(), "holds none") {
		t.Errorf("the refusal %q does not say the workspace is empty", err)
	}
}

// TestAWorkspaceThatCannotBeWalkedStillRunsThePhase. The listing is an offer,
// not a requirement: a workspace pointed somewhere that is not there leaves
// the task exactly as capable as it was before the offer existed, working in
// the repository it was written against.
func TestAWorkspaceThatCannotBeWalkedStillRunsThePhase(t *testing.T) {
	_, repos := workspaceFixture(t, "app")
	t.Setenv(repo.WorkspaceEnv, filepath.Join(t.TempDir(), "not-here"))

	tk := Task{ID: "ACME-5", Text: "add the retry", Repo: repos[0]}
	if others := elsewhere(tk); others != nil {
		t.Errorf("a workspace that is not there listed %v", others)
	}

	if said := workspace(nil); said != "" {
		t.Errorf("the prompt offers a workspace it could not read: %q", said)
	}
}
