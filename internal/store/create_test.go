package store

// The creation side of the store: the verbs that actually make directories,
// and the permissions they make them with. store_test.go covers the path
// methods, which make nothing.

import (
	"os"
	"path/filepath"
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
