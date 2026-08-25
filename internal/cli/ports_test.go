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
	"path/filepath"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/view"
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
	root, orbitHome := workspace(t)
	writeTask(t, root)

	s, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}

	// A damaged run marker makes task.Reconcile fail for this one task,
	// without stopping the sweep over whatever else the state root knows
	// about.
	events := findFile(t, orbitHome, "events.jsonl")
	marker := filepath.Join(filepath.Dir(events), "run")
	if err := os.WriteFile(marker, []byte("pid: not-a-number\nstarted: 2026-08-24T12:00:00Z\n"), 0o600); err != nil {
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
	tasksDir, err := s.TasksDir(r.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(tasksDir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer func() { _ = os.Chmod(tasksDir, 0o700) }() //nolint:errcheck

	if err := reconcileAll(s); err == nil {
		t.Error("reconcileAll succeeded despite an unlistable tasks directory")
	}
}

func TestEnginesPortMarksEveryEngineUnavailableWithNoPath(t *testing.T) {
	t.Setenv("PATH", "")
	engFn := enginesPort(map[string]engine.Engine{
		"claude":   engine.NewClaude(),
		"codex":    engine.NewCodex(),
		"opencode": engine.NewOpenCode(),
	})
	for _, info := range engFn() {
		if info.Available {
			t.Errorf("engine %q reported available with an empty $PATH", info.Name)
		}
		if len(info.Setup) == 0 {
			t.Errorf("engine %q reported unavailable with no setup guide", info.Name)
		}
	}
}
