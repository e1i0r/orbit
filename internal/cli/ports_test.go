package cli

// The ports.go branches TestPortsHelpers (ports_settings_test.go) does not
// reach: startPort actually getting past task.Load to task.Start,
// lastSession failing rather than finding no session, reconcileAll's three
// error paths (s.Repos() itself, task.List per repository, task.Reconcile
// per task), and enginesPort's "not available" branch — which on a machine
// that happens to have claude/codex/opencode on $PATH (as this one does)
// never fires unless $PATH is emptied for the call.

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

func TestStartPortReachesTaskStartOnceLoaded(t *testing.T) {
	root, _ := workspace(t)
	dir := writeTask(t, root)

	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	// task.Load resolves the repository the same way `orbit new` did, by
	// canonicalising it through repo.Open — which on a machine where the
	// temp directory is itself a symlink (macOS: /var -> /private/var)
	// answers a different path than `dir` typed as a plain string. Handing
	// the port that raw string here is the mistake TestCancelTaskExecution
	// makes (see task_verbs_test.go's header): the load would fail every
	// time and this test would never reach task.Start at all.
	r, err := repo.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	sp := startPort(s)
	// The load succeeds (the task was written down above), so this reaches
	// task.Start rather than returning at the `if err != nil` guard —
	// whatever task.Start itself makes of an unread cap of 0 is internal/
	// task's business, not this port's.
	_, _ = sp(view.Task{ID: "ACME-1", Repo: r.Name, RepoPath: r.Path}, "", 0) //nolint:errcheck
}

func TestLastSessionFailsOnAnInvalidTaskID(t *testing.T) {
	root, _ := workspace(t)

	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}

	rdr := board.NewReader(s, root)
	if _, err := lastSession(rdr, view.Task{ID: "bad/id", RepoPath: root}); err == nil {
		t.Error("lastSession on an id with a path separator succeeded")
	}
}

func TestReconcileAllFailsWhenRepositoriesCannotBeListed(t *testing.T) {
	home := t.TempDir()

	s, err := store.New(home)
	if err != nil {
		t.Fatal(err)
	}

	reposDir := filepath.Join(home, "repos")
	if err := os.MkdirAll(reposDir, 0o000); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	defer func() { _ = os.Chmod(reposDir, 0o700) }() //nolint:errcheck

	if err := reconcileAll(s); err == nil {
		t.Error("reconcileAll succeeded over an unreadable repos directory")
	}
}

func TestReconcileAllReportsPerRepositoryAndPerTaskFailures(t *testing.T) {
	root, _ := workspace(t)
	writeTask(t, root)

	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}

	// A damaged run marker makes task.Reconcile fail for this one task,
	// without stopping the sweep over whatever else the state root knows
	// about.
	if err := os.WriteFile(markerPath(t, "ACME-1"), []byte("pid: not-a-number\nstarted: 2026-08-24T12:00:00Z\n"), 0o600); err != nil {
		t.Fatalf("write the run marker: %v", err)
	}

	if err := reconcileAll(s); err == nil {
		t.Error("reconcileAll succeeded despite a damaged run marker")
	}

	// A second repository whose tasks directory cannot be listed at all,
	// so the same sweep also takes the task.List error branch.
	second := filepath.Join(root, "second")
	initRepo(t, second)

	r, err := repo.Open(second)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := task.Create(s, r, "SECOND-1", "text", "quick"); err != nil {
		t.Fatal(err)
	}

	tasksDir := s.TasksDir()

	if err := os.Chmod(tasksDir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer func() { _ = os.Chmod(tasksDir, 0o700) }() //nolint:errcheck

	if err := reconcileAll(s); err == nil {
		t.Error("reconcileAll succeeded despite an unlistable tasks directory")
	}
}

// TestEnginesPortMarksEveryEngineUnavailableWithNoPath. An empty PATH is no
// longer the whole question — the engines also look in the directories their
// own installers use — so the home this runs against is emptied too, and
// what it asserts is a machine with none of the three.
func TestEnginesPortMarksEveryEngineUnavailableWithNoPath(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("HOME", t.TempDir())

	engFn := enginesPort(map[string]engine.Engine{
		"claude":   engine.NewClaude(),
		"codex":    engine.NewCodex(),
		"opencode": engine.NewOpenCode(),
	})
	for _, info := range engFn() {
		if info.Available {
			t.Errorf("engine %q reported available with an empty $PATH", info.Name)
		}

		if info.Setup == nil {
			t.Errorf("engine %q reported unavailable with no setup guide", info.Name)
			continue
		}

		if len(info.Setup(words.For("en"))) == 0 {
			t.Errorf("engine %q reported unavailable with an empty setup guide", info.Name)
		}
		// What an engine offers and whether this machine can run it are
		// two facts. Dials filled in only for engines that are installed
		// leave the window with nothing to draw from unless the reader
		// already has the engine — so the window keeps a copy of the
		// catalogue, and the copy is what drifts.
		if len(info.Models) == 0 || len(info.Efforts) == 0 {
			t.Errorf("engine %q offers no models or efforts because it is not installed; both are facts about the engine, not about $PATH", info.Name)
		}
	}
}

// The delete gesture. It is a soft delete — task.Delete says why — so what
// this port is asked for is that the row leaves every listing and that the
// account of what the task did is still there to read afterwards.

func TestDeletingATaskTakesItOffTheListingAndKeepsItsRecord(t *testing.T) {
	root, _ := workspace(t)
	dir := writeTask(t, root)

	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}

	r, err := repo.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := deleteTaskPort(s)(view.Task{ID: "ACME-1", Repo: r.Name, RepoPath: r.Path}); err != nil {
		t.Fatalf("delete: %v", err)
	}

	ids, err := task.List(s, r)
	if err != nil {
		t.Fatalf("list the tasks of %s: %v", r.Name, err)
	}

	if slices.Contains(ids, "ACME-1") {
		t.Errorf("the deleted task is still listed: %v", ids)
	}

	events, err := task.Events(s, task.Task{ID: "ACME-1", Repo: r})
	if err != nil {
		t.Fatalf("read the record of the deleted task: %v", err)
	}

	if len(events) == 0 {
		t.Error("the record of the deleted task is empty, and it was the only account of what it did")
	}
}

// TestDeletingATaskGivesTheWorktreeBackToGit. os.RemoveAll takes the checkout
// away and leaves the entry under .git/worktrees behind, in a repository
// Orbit does not own — and the next `git worktree add` on that branch meets
// it.
func TestDeletingATaskGivesTheWorktreeBackToGit(t *testing.T) {
	root, _ := workspace(t)
	dir := writeTask(t, root)

	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}

	r, err := repo.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	wtDir, err := s.WorktreeDir(r.Path, "ACME-1")
	if err != nil {
		t.Fatal(err)
	}

	if err := r.AddWorktree(wtDir, "orbit/ACME-1"); err != nil {
		t.Fatalf("add a worktree: %v", err)
	}

	if err := deleteTaskPort(s)(view.Task{ID: "ACME-1", Repo: r.Name, RepoPath: r.Path}); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := os.Stat(wtDir); err == nil {
		t.Errorf("the checkout at %q is still there", wtDir)
	}

	if listed := worktrees(t, r.Path); strings.Contains(listed, wtDir) {
		t.Errorf("git still lists the worktree that was deleted:\n%s", listed)
	}
}

// TestADeleteThatCouldNotHappenSaysSo. An id the store refuses is the one
// failure this port can be handed on purpose, and the window has a band to
// put the sentence in.
func TestADeleteThatCouldNotHappenSaysSo(t *testing.T) {
	workspace(t)

	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}

	if err := deleteTaskPort(s)(view.Task{ID: "../etc", Repo: "payments", RepoPath: "payments"}); err == nil {
		t.Error("a delete the store refused came back as a success")
	}
}

// worktrees is what git says the repository has, which is the half of a
// worktree that lives inside the repository rather than in the state root.
func worktrees(t *testing.T, repoPath string) string {
	t.Helper()

	cmd := exec.Command("git", "worktree", "list")
	cmd.Dir = repoPath

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git worktree list: %v\n%s", err, out)
	}

	return string(out)
}
