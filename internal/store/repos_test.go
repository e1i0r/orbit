package store

import (
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
