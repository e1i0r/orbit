package repo

// Where a task looks for the repositories it might reach into, and how a
// layout that is not side by side says so.

import (
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"testing"
)

// sideBySide is the arrangement Workspace guesses at: one directory holding
// a checkout per project.
func sideBySide(t *testing.T, names ...string) (root string, repos []Repo) {
	t.Helper()

	root = t.TempDir()

	for _, name := range names {
		dir := filepath.Join(root, name)
		if err := mkdir(dir); err != nil {
			t.Fatalf("mkdir %q: %v", dir, err)
		}

		makeRepo(t, dir, "main", "origin")

		r, err := Open(dir)
		if err != nil {
			t.Fatalf("Open %q: %v", dir, err)
		}

		repos = append(repos, r)
	}

	return root, repos
}

// found is the names of a listing, sorted, so a test says which repositories
// were seen rather than in what order the walk happened to see them.
func found(repos []Repo) []string {
	names := make([]string, 0, len(repos))
	for _, r := range repos {
		names = append(names, r.Name)
	}

	slices.Sort(names)

	return names
}

// TestTheCheckoutsNextToOneAreItsWorkspace. Nobody configured this and no
// task declared it: api, app and scripts are candidates for one another's
// work because that is how they are checked out.
func TestTheCheckoutsNextToOneAreItsWorkspace(t *testing.T) {
	t.Setenv(WorkspaceEnv, "")

	_, repos := sideBySide(t, "api", "app", "scripts")

	if got, want := Workspace(repos[0]), filepath.Dir(repos[0].Path); got != want {
		t.Errorf("the workspace of %q is %q, want %q", repos[0].Name, got, want)
	}

	near, err := Siblings(repos[0])
	if err != nil {
		t.Fatalf("Siblings: %v", err)
	}

	// The repository the task was written against is in its own listing.
	// Whether it is already joined is a question about the task, and this
	// answer is about the workspace.
	if got := found(near); !slices.Equal(got, []string{"api", "app", "scripts"}) {
		t.Errorf("the workspace holds %v, want all three", got)
	}
}

// TestAWorkspaceSaidOutrightIsTheOneUsed. The guess is right for checkouts
// kept side by side and wrong for anything else, so it is a guess a user can
// answer — and a phase running in a worktree under the state root answers it
// the same way, because its own parent directory says nothing about where the
// projects live.
func TestAWorkspaceSaidOutrightIsTheOneUsed(t *testing.T) {
	_, elsewhere := sideBySide(t, "ledger")

	// A repository checked out on its own, nowhere near the others.
	lone := filepath.Join(t.TempDir(), "deep", "payments")
	if err := mkdir(lone); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	makeRepo(t, lone, "main", "origin")

	r, err := Open(lone)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	said := filepath.Dir(elsewhere[0].Path)
	t.Setenv(WorkspaceEnv, said)

	if got := Workspace(r); got != said {
		t.Errorf("the workspace is %q, want the one it was told: %q", got, said)
	}

	near, err := Siblings(r)
	if err != nil {
		t.Fatalf("Siblings: %v", err)
	}

	if got := found(near); !slices.Equal(got, []string{"ledger"}) {
		t.Errorf("the workspace holds %v, want [ledger]", got)
	}
}

// TestAWorkspaceThatIsNotThereIsAnError, and not an empty listing. A task
// whose only repository has been moved away has no candidates for a reason
// worth saying out loud, and reading that back as "there is nothing to join"
// is how a run goes quiet about something a user has to fix.
func TestAWorkspaceThatIsNotThereIsAnError(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "not-here")
	t.Setenv(WorkspaceEnv, gone)

	if _, err := Siblings(Repo{Path: gone, Name: "not-here"}); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Siblings of a directory that is not there gave %v, want a not-exist error", err)
	}
}

// TestATaskWithNoRepositoryHasNoWorkspaceToGuessFrom. The guess is the
// parent of the repository the task is against, and filepath.Dir("") is "."
// — so a task that has reached into nothing would name the directory the
// process happens to be in. That is a workspace nobody chose, holding
// whatever checkouts are below it, walked on every phase. The answer is
// nothing, and a listing of nothing is what Siblings gives back.
func TestATaskWithNoRepositoryHasNoWorkspaceToGuessFrom(t *testing.T) {
	t.Setenv(WorkspaceEnv, "")

	if got := Workspace(Repo{}); got != "" {
		t.Errorf("the workspace of no repository is %q, want nothing", got)
	}

	near, err := Siblings(Repo{})
	if err != nil {
		t.Fatalf("Siblings of no repository: %v", err)
	}

	if len(near) != 0 {
		t.Errorf("the workspace of no repository holds %v", found(near))
	}
}
