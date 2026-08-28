package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReposIsEmptyWhenTheRootHasNoRepos(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	got, err := s.Repos()
	if err != nil {
		t.Fatalf("Repos: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("got %d repos from a root that never created any, want 0", len(got))
	}
}

func TestReposListsWhatCreateRepoDirWrote(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := s.CreateTaskDir("/tmp/one", "ACME-1"); err != nil {
		t.Fatalf("CreateTaskDir: %v", err)
	}

	if _, err := s.CreateTaskDir("/tmp/two", "ACME-2"); err != nil {
		t.Fatalf("CreateTaskDir: %v", err)
	}

	got, err := s.Repos()
	if err != nil {
		t.Fatalf("Repos: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d repos, want 2", len(got))
	}

	byKey := map[string]string{}
	for _, r := range got {
		byKey[r.Key] = r.Path
	}

	oneDir, err := s.RepoDir("/tmp/one")
	if err != nil {
		t.Fatalf("RepoDir: %v", err)
	}

	oneKey := filepath.Base(oneDir)
	if byKey[oneKey] != "/tmp/one" {
		t.Errorf("Repos()[%q] = %q, want %q", oneKey, byKey[oneKey], "/tmp/one")
	}
}

func TestReposSkipsADirectoryWithNoMarker(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := s.CreateTaskDir("/tmp/one", "ACME-1"); err != nil {
		t.Fatalf("CreateTaskDir: %v", err)
	}
	// A half-created repo directory: made, but the marker was never written.
	half := filepath.Join(s.Root(), "repos", "deadbeefcafe")
	if err := os.MkdirAll(half, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got, err := s.Repos()
	if err != nil {
		t.Fatalf("Repos: %v — a half-created directory is not an error the reader can do anything about", err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d repos, want 1 — the marker-less directory must be skipped, not reported", len(got))
	}
}

func TestReposReturnsGoodReposEvenWhenOneMarkerIsDamaged(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := s.CreateTaskDir("/tmp/one", "ACME-1"); err != nil {
		t.Fatalf("CreateTaskDir: %v", err)
	}

	if _, err := s.CreateTaskDir("/tmp/two", "ACME-2"); err != nil {
		t.Fatalf("CreateTaskDir: %v", err)
	}

	// A marker that exists but was never a valid "path: /abs/path" line —
	// damage, not a half-finished write.
	damagedDir := filepath.Join(s.Root(), "repos", "deadbeefcafe")
	if err := os.MkdirAll(damagedDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	damagedMarker := filepath.Join(damagedDir, "repo")
	if err := os.WriteFile(damagedMarker, []byte("not the marker format"), 0o600); err != nil {
		t.Fatalf("write damaged marker: %v", err)
	}

	got, err := s.Repos()
	if err == nil {
		t.Fatal("Repos: got nil error with a damaged marker present, want a non-nil error naming it")
	}

	if !strings.Contains(err.Error(), damagedMarker) {
		t.Errorf("error %q does not name the damaged marker %q", err.Error(), damagedMarker)
	}

	if len(got) != 2 {
		t.Fatalf("got %d repos, want 2 — a damaged marker must not blank the repos that read fine", len(got))
	}
}

func TestReposTreatsARelativePathMarkerAsDamaged(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	relDir := filepath.Join(s.Root(), "repos", "deadbeefcafe")
	if err := os.MkdirAll(relDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	relMarker := filepath.Join(relDir, "repo")
	if err := os.WriteFile(relMarker, []byte("path: relative/path\n"), 0o600); err != nil {
		t.Fatalf("write relative-path marker: %v", err)
	}

	got, err := s.Repos()
	if err == nil {
		t.Fatal("Repos: got nil error with a relative-path marker present, want a non-nil error")
	}

	if len(got) != 0 {
		t.Fatalf("got %d repos from a relative-path marker, want 0 — it must not be returned as a RepoRef", len(got))
	}
}

func TestReposSurvivesARealIOErrorPartwayThroughTheListing(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	root := filepath.Join(s.Root(), "repos")

	writeMarker := func(name, path string) {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}

		if err := os.WriteFile(filepath.Join(dir, "repo"), []byte("path: "+path+"\n"), 0o600); err != nil {
			t.Fatalf("write marker %s: %v", name, err)
		}
	}

	// Directory names are chosen so os.ReadDir's lexical order puts the
	// broken one in the middle: a-first, b-second (broken), c-third,
	// d-fourth. This is the ordering the reviewer's reproduction used —
	// repos sorted after a real I/O error must still come back.
	writeMarker("a-first", "/tmp/first")

	// A marker path that is itself a directory: os.ReadFile on it fails with
	// a genuine I/O error, not os.ErrNotExist.
	brokenMarker := filepath.Join(root, "b-second", "repo")
	if err := os.MkdirAll(brokenMarker, 0o700); err != nil {
		t.Fatalf("mkdir broken marker: %v", err)
	}

	writeMarker("c-third", "/tmp/third")
	writeMarker("d-fourth", "/tmp/fourth")

	got, err := s.Repos()
	if err == nil {
		t.Fatal("Repos: got nil error with an unreadable marker present, want a non-nil error naming it")
	}

	if !strings.Contains(err.Error(), brokenMarker) {
		t.Errorf("error %q does not name the unreadable marker %q", err.Error(), brokenMarker)
	}

	byKey := map[string]string{}
	for _, r := range got {
		byKey[r.Key] = r.Path
	}

	if byKey["a-first"] != "/tmp/first" {
		t.Errorf("the repo before the I/O failure is missing: got %+v", got)
	}

	if byKey["c-third"] != "/tmp/third" || byKey["d-fourth"] != "/tmp/fourth" {
		t.Errorf("the repos sorted after the I/O failure did not survive: got %+v", got)
	}

	if len(got) != 3 {
		t.Fatalf("got %d repos, want 3 — every readable, parseable marker must still come back", len(got))
	}
}

// TestReposTellsAnUnlistableDirectoryFromADamagedMarker: both come back with
// no repositories and a non-nil error, so a caller that has to decide
// whether an empty answer is the whole story cannot use the slice to tell
// them apart. The type is what it asks instead.
func TestReposTellsAnUnlistableDirectoryFromADamagedMarker(t *testing.T) {
	unlistable, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	dir := filepath.Join(unlistable.Root(), "repos")
	if err := os.WriteFile(dir, []byte("not a directory\n"), 0o600); err != nil {
		t.Fatalf("write a file where repos/ goes: %v", err)
	}

	got, err := unlistable.Repos()

	var listing *ReposError
	if !errors.As(err, &listing) {
		t.Fatalf("Repos on an unlistable directory gave %v, want a *ReposError", err)
	}

	if listing.Dir != dir {
		t.Errorf("the error names %q, want %q", listing.Dir, dir)
	}

	if len(got) != 0 {
		t.Errorf("got %d repos from a directory that could not be listed", len(got))
	}

	damaged, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	damagedDir := filepath.Join(damaged.Root(), "repos", "deadbeefcafe")
	if err := os.MkdirAll(damagedDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(damagedDir, "repo"), []byte("not the marker format"), 0o600); err != nil {
		t.Fatalf("write damaged marker: %v", err)
	}

	if _, err := damaged.Repos(); errors.As(err, &listing) {
		t.Errorf("a damaged marker came back as %T: it is one repository, not the listing", listing)
	} else if err == nil {
		t.Error("a damaged marker gave no error at all")
	}
}

func TestForgetRepoRemovesTheRecordAndNothingElse(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := s.CreateTaskDir("/tmp/one", "ACME-1"); err != nil {
		t.Fatalf("CreateTaskDir: %v", err)
	}

	if _, err := s.CreateTaskDir("/tmp/two", "ACME-2"); err != nil {
		t.Fatalf("CreateTaskDir: %v", err)
	}
	// The worktree of the repository being forgotten: it is a checkout git
	// has registered, and it lives outside repos/ so that forgetting the
	// record cannot take it with it.
	worktree, err := s.CreateWorktreeParent("/tmp/one", "ACME-1")
	if err != nil {
		t.Fatalf("CreateWorktreeParent: %v", err)
	}

	dir, err := s.ForgetRepo("/tmp/one")
	if err != nil {
		t.Fatalf("ForgetRepo: %v", err)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the record is still at %q after ForgetRepo: %v", dir, err)
	}

	if _, err := os.Stat(filepath.Dir(worktree)); err != nil {
		t.Errorf("forgetting the record took the worktree with it: %v", err)
	}

	rest, err := s.Repos()
	if err != nil {
		t.Fatalf("Repos: %v", err)
	}

	if len(rest) != 1 || rest[0].Path != "/tmp/two" {
		t.Errorf("Repos() = %+v, want only the repository that was not forgotten", rest)
	}
}

// "Forgotten" and "never known" are different answers, and a caller that
// misspelled a path deserves to hear which one it got.
func TestForgetRepoSaysWhenThereIsNoRecord(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = s.ForgetRepo("/tmp/never")
	if err == nil {
		t.Fatal("forgetting a repository the root never knew was reported as a success")
	}

	if !strings.Contains(err.Error(), "/tmp/never") {
		t.Errorf("the error does not name the repository: %v", err)
	}
}
