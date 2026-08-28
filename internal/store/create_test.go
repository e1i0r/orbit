package store

// The creation side of the store: the verbs that actually make directories,
// and the permissions they make them with. store_test.go covers the path
// methods, which make nothing.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCreatesTheRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "orbit")
	s, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s.Root() != root {
		t.Errorf("Root() = %q, want %q", s.Root(), root)
	}
	info, err := os.Stat(root)
	if err != nil {
		t.Fatalf("stat root: %v", err)
	}
	if !info.IsDir() {
		t.Error("root is not a directory")
	}
}

func TestCreateTaskDirMakesTheWholeChain(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	dir, err := s.CreateTaskDir("/tmp/one", "ACME-1")
	if err != nil {
		t.Fatalf("CreateTaskDir: %v", err)
	}
	want, err := s.TaskDir("/tmp/one", "ACME-1")
	if err != nil {
		t.Fatalf("TaskDir: %v", err)
	}
	if dir != want {
		t.Errorf("CreateTaskDir made %q, TaskDir points at %q", dir, want)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("the task directory was not created: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("task directory is %o, want 700 — the state root is private", perm)
	}
}

func TestCreateRepoDirWritesWhichRepositoryItIs(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := s.CreateTaskDir("/tmp/one", "ACME-1"); err != nil {
		t.Fatalf("CreateTaskDir: %v", err)
	}
	repoDir, err := s.RepoDir("/tmp/one")
	if err != nil {
		t.Fatalf("RepoDir: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(repoDir, "repo"))
	if err != nil {
		t.Fatalf("a hash directory with nothing saying what it is: %v", err)
	}
	if string(body) != "path: /tmp/one\n" {
		t.Errorf("repo file = %q, want %q", body, "path: /tmp/one\n")
	}
}

func TestCreateWorktreeParentLeavesTheLeafForGit(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	dir, err := s.CreateWorktreeParent("/tmp/one", "ACME-1")
	if err != nil {
		t.Fatalf("CreateWorktreeParent: %v", err)
	}
	if _, statErr := os.Stat(dir); statErr == nil {
		t.Error("worktree leaf exists but must not; git worktree add will fail")
	}
	info, err := os.Stat(filepath.Dir(dir))
	if err != nil {
		t.Fatalf("worktree parent was not created: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("worktree parent is %o, want 700 — the state root is private", perm)
	}
}

func TestTheRootIsPrivate(t *testing.T) {
	root := filepath.Join(t.TempDir(), "orbit")
	if _, err := New(root); err != nil {
		t.Fatalf("New: %v", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		t.Fatalf("stat root: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("the state root is %o, want 700 — it holds checkouts of private repositories, every task, and one day credentials", perm)
	}
}

// RegisterRepo is the same directory CreateTaskDir would have made, made
// without a task: the listing was a side effect of running something, and a
// caller that cannot walk a directory needs to be able to say "know about
// this one" on its own.
func TestRegisterRepoPutsARepositoryInTheListing(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	dir, err := s.RegisterRepo("/tmp/one")
	if err != nil {
		t.Fatalf("RegisterRepo: %v", err)
	}
	want, err := s.RepoDir("/tmp/one")
	if err != nil {
		t.Fatalf("RepoDir: %v", err)
	}
	if dir != want {
		t.Errorf("RegisterRepo made %q, want %q", dir, want)
	}
	marker, err := os.ReadFile(filepath.Join(dir, "repo"))
	if err != nil {
		t.Fatalf("read the marker: %v", err)
	}
	if string(marker) != "path: /tmp/one\n" {
		t.Errorf("marker = %q, want it to name the repository", marker)
	}
	repos, err := s.Repos()
	if err != nil {
		t.Fatalf("Repos: %v", err)
	}
	if len(repos) != 1 || repos[0].Path != "/tmp/one" {
		t.Errorf("Repos() = %+v, want the one repository that was registered", repos)
	}
}

// Registering twice is not an error and does not rewrite the marker: a
// caller registering a repository that already has tasks is the ordinary
// case, and it must not disturb what is already recorded there.
func TestRegisterRepoTwiceLeavesTheRecordAlone(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := s.CreateTaskDir("/tmp/one", "ACME-1"); err != nil {
		t.Fatalf("CreateTaskDir: %v", err)
	}
	if _, err := s.RegisterRepo("/tmp/one"); err != nil {
		t.Fatalf("RegisterRepo: %v", err)
	}
	task, err := s.TaskDir("/tmp/one", "ACME-1")
	if err != nil {
		t.Fatalf("TaskDir: %v", err)
	}
	if _, err := os.Stat(task); err != nil {
		t.Errorf("registering an already-known repository disturbed its tasks: %v", err)
	}
}

// A hash collision cannot be produced on purpose, so these two stand in for
// one the way the code sees it: a repository directory whose marker names
// somewhere else. createRepoDir cannot tell this apart from the real thing.

func TestCreateRepoDirRefusesADirectoryThatNamesAnotherRepository(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	dir, err := s.RepoDir("/tmp/one")
	if err != nil {
		t.Fatalf("RepoDir: %v", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "repo"), []byte("path: /tmp/two\n"), 0o600); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	_, err = s.RegisterRepo("/tmp/one")
	if err == nil {
		t.Fatal("RegisterRepo filed a repository under a key whose marker names another one")
	}
	for _, want := range []string{"/tmp/one", "/tmp/two"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q, so nobody can tell which two collided", err, want)
		}
	}
	if _, err := s.CreateTaskDir("/tmp/one", "ACME-1"); err == nil {
		t.Error("CreateTaskDir wrote a task into a record belonging to another repository")
	}
}

func TestCreateRepoDirLeavesADamagedMarkerAlone(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	dir, err := s.RepoDir("/tmp/one")
	if err != nil {
		t.Fatalf("RepoDir: %v", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	marker := filepath.Join(dir, "repo")
	if err := os.WriteFile(marker, []byte("nonsense\n"), 0o600); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	if _, err := s.RegisterRepo("/tmp/one"); err != nil {
		t.Fatalf("RegisterRepo refused a damaged marker instead of leaving it: %v", err)
	}
	body, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if string(body) != "nonsense\n" {
		t.Errorf("marker = %q, want it untouched: Repos is what reports damage, and a rewrite erases it", body)
	}
}
