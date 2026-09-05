package repo

// The walk on its own: what it does about a directory it cannot look into,
// and finding repositories without paying three git subprocesses for each.

import (
	"errors"
	"io/fs"
	"path/filepath"
	"syscall"
	"testing"
)

func TestDotGitVerdictWalksOnWhenThereIsNoGitEntry(t *testing.T) {
	if err := dotGitVerdict(fs.ErrNotExist); err != nil {
		t.Fatalf("dotGitVerdict(ErrNotExist) = %v, want nil — an ordinary directory is walked into", err)
	}
}

func TestDotGitVerdictSkipsADirectoryItMayNotRead(t *testing.T) {
	if err := dotGitVerdict(fs.ErrPermission); !errors.Is(err, filepath.SkipDir) {
		t.Fatalf("dotGitVerdict(ErrPermission) = %v, want SkipDir — a directory that cannot be read is left alone", err)
	}
}

func TestDotGitVerdictStopsTheWalkWhenTheSystemIsOutOfFileDescriptors(t *testing.T) {
	err := dotGitVerdict(syscall.ENFILE)
	if !errors.Is(err, syscall.ENFILE) {
		t.Fatalf("dotGitVerdict(ENFILE) = %v, want the error itself — a walk that cannot look "+
			"must say so rather than descend into every directory it failed to recognise", err)
	}
}

func TestPathsFindsRepositoriesWithoutOpeningThem(t *testing.T) {
	root := t.TempDir()

	// A .git that is a directory and nothing else. git cannot open it, so a
	// walk that still reports it is one that never asked git.
	for _, name := range []string{"payments", "ledger"} {
		if err := mkdir(filepath.Join(root, name, ".git")); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	found, err := Paths(root)
	if err != nil {
		t.Fatalf("Paths: %v", err)
	}

	if len(found) != 2 {
		t.Fatalf("Paths found %d repositories, want 2", len(found))
	}

	if found[0].Name != "ledger" || found[1].Name != "payments" {
		t.Fatalf("Paths found %q and %q, want ledger and payments sorted by path",
			found[0].Name, found[1].Name)
	}
}

func TestPathsDoesNotDescendIntoARepositoryItFound(t *testing.T) {
	root := t.TempDir()

	if err := mkdir(filepath.Join(root, "payments", ".git")); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// A checkout inside a checkout is a vendored copy, not a second project.
	if err := mkdir(filepath.Join(root, "payments", "vendored", ".git")); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	found, err := Paths(root)
	if err != nil {
		t.Fatalf("Paths: %v", err)
	}

	if len(found) != 1 || found[0].Name != "payments" {
		t.Fatalf("Paths found %d repositories, want only payments", len(found))
	}
}
