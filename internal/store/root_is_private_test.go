package store

// The state root is private, and kept that way every time it is opened.

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAWideStateRootIsNarrowedToItsOwner.
//
// The root holds every task's record and every private checkout under it. A
// directory somebody made with a loose umask, or copied from somewhere with
// one, is readable by anybody with an account on the machine — and nobody
// goes looking at the mode of a directory a tool made for them.
func TestAWideStateRootIsNarrowedToItsOwner(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	if err := os.Mkdir(root, 0o777); err != nil {
		t.Fatalf("make a wide root: %v", err)
	}

	// Mkdir goes through the umask, so it is said again where nothing can
	// take it back.
	if err := os.Chmod(root, 0o777); err != nil {
		t.Fatalf("widen the root: %v", err)
	}

	if _, err := New(root); err != nil {
		t.Fatalf("New: %v", err)
	}

	info, err := os.Stat(root)
	if err != nil {
		t.Fatalf("read the root back: %v", err)
	}

	if perm := info.Mode().Perm(); perm != dirMode {
		t.Errorf("the state root is %o, want %o — it holds every task's record", perm, dirMode)
	}
}

// TestARootDeliberatelyMadeReadOnlyStaysReadOnly.
//
// What is taken away is what group and other were given, and the owner's own
// bits are left as they are. Widening a root somebody narrowed on purpose
// would be this deciding it knows better, and the write that was meant to
// fail would quietly succeed.
func TestARootDeliberatelyMadeReadOnlyStaysReadOnly(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	if err := os.Mkdir(root, 0o500); err != nil {
		t.Fatalf("make a read-only root: %v", err)
	}

	if err := os.Chmod(root, 0o500); err != nil {
		t.Fatalf("narrow the root: %v", err)
	}

	if _, err := New(root); err != nil {
		t.Fatalf("New: %v", err)
	}

	info, err := os.Stat(root)
	if err != nil {
		t.Fatalf("read the root back: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0o500 {
		t.Errorf("the root is %o, want the 500 somebody chose", perm)
	}
}
