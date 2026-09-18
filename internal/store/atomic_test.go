package store

// Putting a small file on disk in one step, and what is left behind when
// that cannot be done.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAWriteThatFailedLeavesTheDirectoryAsItFoundIt.
//
// The temporary is written beside the target and not in the system's temp
// directory, because a rename is atomic only within one filesystem. That
// puts it under the state root, where somebody looking at their own files
// will see it — and a write that failed and left one behind leaves another
// every time it fails.
func TestAWriteThatFailedLeavesTheDirectoryAsItFoundIt(t *testing.T) {
	dir := t.TempDir()

	// A directory standing where the file should go, so that the rename is
	// what fails: it is the last step, and by then the temporary is written.
	at := filepath.Join(dir, "settings.json")
	if err := os.Mkdir(at, dirMode); err != nil {
		t.Fatalf("stand a directory in the way: %v", err)
	}

	if err := WriteAtomically(at, []byte(`{"engine":"claude"}`)); err == nil {
		t.Fatal("writing a file over a directory was not refused")
	}

	left, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the directory back: %v", err)
	}

	for _, e := range left {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("a write that failed left %q behind", e.Name())
		}
	}

	if len(left) != 1 {
		t.Errorf("the directory holds %d entries, want the one it held before", len(left))
	}
}

// TestAWriteThatWorkedLeavesTheFileAndNothingElse, which is the other half:
// the temporary is the mechanism and never something a reader of the state
// root has to know about.
func TestAWriteThatWorkedLeavesTheFileAndNothingElse(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, "settings.json")

	if err := os.WriteFile(at, []byte("a much longer file than the one replacing it"), fileMode); err != nil {
		t.Fatalf("write what is being replaced: %v", err)
	}

	if err := WriteAtomically(at, []byte(`{"engine":"claude"}`)); err != nil {
		t.Fatalf("write: %v", err)
	}

	left, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the directory back: %v", err)
	}

	if len(left) != 1 || left[0].Name() != "settings.json" {
		t.Errorf("the directory holds %d entries, want only the file that was written", len(left))
	}

	// Whole, and not the longer file with a shorter one written over its
	// front: that is the half-written state the rename exists to prevent.
	body, err := os.ReadFile(at)
	if err != nil {
		t.Fatalf("read the file back: %v", err)
	}

	if string(body) != `{"engine":"claude"}` {
		t.Errorf("the file reads %q", body)
	}
}
