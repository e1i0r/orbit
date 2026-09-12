package store

// Taking a repository back out of a task's marker, leaving the order of
// the rest as it was — and taking nothing out when it was never in.

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUnjoinRepoKeepsTheOrderOfTheRest.
func TestUnjoinRepoKeepsTheOrderOfTheRest(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = s.Close() })

	marker, err := s.TaskReposPath("ACME-1")
	if err != nil {
		t.Fatalf("marker path: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		t.Fatalf("make the marker directory: %v", err)
	}

	if err := os.WriteFile(marker, []byte("path: /src/acme\npath: /src/ledger\n"), fileMode); err != nil {
		t.Fatalf("write the marker: %v", err)
	}

	if err := s.UnjoinRepo("ACME-1", "/src/ledger"); err != nil {
		t.Fatalf("unjoin: %v", err)
	}

	kept, err := s.TaskRepos("ACME-1")
	if err != nil {
		t.Fatalf("read the marker: %v", err)
	}

	if len(kept) != 1 || kept[0] != "/src/acme" {
		t.Errorf("the marker holds %v, want only acme", kept)
	}

	if err := s.UnjoinRepo("ACME-1", "/src/ledger"); err != nil {
		t.Errorf("unjoining what was never joined: %v", err)
	}
}
