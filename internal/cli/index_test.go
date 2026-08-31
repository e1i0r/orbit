package cli

// The index kept up by the commands, without any command being about it.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/e1i0r/orbit/internal/index"
	"github.com/e1i0r/orbit/internal/store"
)

func TestRunningACommandFoldsTheRecordIntoTheIndex(t *testing.T) {
	root, orbitHome := workspace(t)
	repoDir := filepath.Join(root, "payments")

	code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "retry the webhook on 5xx")
	if code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	// The write above was folded by the command after it, which is the whole
	// arrangement: nothing inserts a row, the record is appended to and the
	// next command folds what it gained.
	if code, _, errOut = run(t, "list", "-repo", repoDir); code != 0 {
		t.Fatalf("list exited %d: %s", code, errOut)
	}

	path := filepath.Join(orbitHome, "index.db")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the index is not there: %v", err)
	}

	x, err := index.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	defer func() {
		if err := x.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	n, err := x.CountOfKind("task.created")
	if err != nil {
		t.Fatalf("CountOfKind: %v", err)
	}

	if n != 1 {
		t.Errorf("the index holds %d task.created, want the one that was written", n)
	}
}

// The index is the one file in the tree that can be deleted on a hunch.
func TestTheIndexComesBackAfterBeingDeleted(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	if code, _, errOut := run(t, "new", "-repo", repoDir, "-id", "ACME-1", "one"); code != 0 {
		t.Fatalf("new exited %d: %s", code, errOut)
	}

	s, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}

	if code, _, errOut := run(t, "list", "-repo", repoDir); code != 0 {
		t.Fatalf("list exited %d: %s", code, errOut)
	}

	if err := os.Remove(s.IndexPath()); err != nil {
		t.Fatalf("remove the index: %v", err)
	}

	if code, _, errOut := run(t, "list", "-repo", repoDir); code != 0 {
		t.Fatalf("list over a deleted index exited %d: %s", code, errOut)
	}

	x, err := index.Open(s.IndexPath())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	defer func() {
		if err := x.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	n, err := x.CountOfKind("task.created")
	if err != nil {
		t.Fatalf("CountOfKind: %v", err)
	}

	if n != 1 {
		t.Errorf("the rebuilt index holds %d task.created, want one", n)
	}
}
