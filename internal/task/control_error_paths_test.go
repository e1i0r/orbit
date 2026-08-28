package task

// The control.go branches control_test.go and task_lifecycle_coverage_test.go
// do not reach: Control and take's own error returns, and take's tolerance
// of a word this version does not know.

import (
	"io"
	"os"
	"strings"
	"testing"
)

// TestControlErrorPaths covers Control's two error returns.
func TestControlErrorPaths(t *testing.T) {
	s, r := fixture(t)

	// 1. Bad id: refused before the door even asks whether the word is one
	// a run understands.
	bad := Task{ID: "has/slash", Repo: r}
	if err := Control(s, bad, "pause"); err == nil {
		t.Error("Control with a slash in the id should have failed")
	}

	// 2. A well-formed id whose directory was never created: the word has
	// nowhere to land.
	neverCreated := Task{ID: "NEVER-CREATED-CTRL", Repo: r}
	if err := Control(s, neverCreated, "pause"); err == nil {
		t.Error("Control into a task directory that does not exist should have failed")
	}
}

// TestTakeErrorPaths covers take's error returns and its tolerance of a word
// that is not one of the five, which a hand-edited file can leave behind.
func TestTakeErrorPaths(t *testing.T) {
	s, r := fixture(t)

	// 1. Bad id.
	bad := Task{ID: "has/slash", Repo: r}
	if _, err := take(s, bad); err == nil {
		t.Error("take with a slash in the id should have failed")
	}

	tk, err := Create(s, r, "TAKE-ERR-1", "take error test", "quick")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	path, err := s.ControlPath(r.Path, tk.ID)
	if err != nil {
		t.Fatalf("ControlPath: %v", err)
	}

	// 2. Something other than "not exist" reading the word: a directory
	// sitting where the control file should be.
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	if _, err := take(s, tk); err == nil {
		t.Error("take over a directory should have failed")
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// 3. A word this version does not know is taken off and treated as
	// though there had been none — the opposite of readMarker's refusal,
	// and deliberately so (see take's doc comment).
	if err := os.WriteFile(path, []byte("not-a-real-word\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	word, err := take(s, tk)
	if err != nil {
		t.Fatalf("take: %v", err)
	}

	if word != "" {
		t.Errorf("take on an unknown word = %q, want empty", word)
	}

	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Error("take left the unknown word's file behind")
	}

	// 4. Cannot remove the word once read: the directory has lost its
	// write bit, so the read succeeds but the clean-up fails.
	if err := os.WriteFile(path, []byte("pause\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	dir, err := s.TaskDir(r.Path, tk.ID)
	if err != nil {
		t.Fatalf("TaskDir: %v", err)
	}

	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) //nolint:errcheck

	if _, err := take(s, tk); err == nil {
		t.Error("take on a read-only directory should have failed to clear the word")
	}
}

// TestAWordIsNeverHalfWritten is the fix.
//
// Control wrote with os.WriteFile, which truncates the file and then writes
// into it. take, reading in that instant, got an empty file — not one of the
// five words — and did what it does with a word it does not know: removed
// it. The word was unlinked before it was ever whole, and the run never saw
// the cancel somebody typed. The gate calls take once per poll for as long
// as a phase waits, so that instant is sampled constantly rather than never.
//
// The race itself is microseconds wide and a test that tries to land inside
// it proves nothing when it misses. What is asserted instead is the property
// that closes it, and it is exactly the one mark's comment claims for the
// run marker: a reader sees the old word or the new one and never half of
// either. A reader that opened the file before the write still holds the old
// word afterwards, because the write replaced the file rather than emptying
// and refilling the one in its hand.
func TestAWordIsNeverHalfWritten(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "CTRL-1", "a word must survive being read", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Control(s, tk, wordPause); err != nil {
		t.Fatalf("Control(pause): %v", err)
	}

	path, err := s.ControlPath(tk.Repo.Path, tk.ID)
	if err != nil {
		t.Fatalf("ControlPath: %v", err)
	}
	// A reader in the middle of take: it has the file open and has not read
	// from it yet.
	reader, err := os.Open(path)
	if err != nil {
		t.Fatalf("open the control file: %v", err)
	}

	defer func() {
		if err := reader.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	}()

	if err := Control(s, tk, wordCancel); err != nil {
		t.Fatalf("Control(cancel): %v", err)
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read what the reader was holding: %v", err)
	}

	if got := strings.TrimSpace(string(body)); got != wordPause {
		t.Errorf("a reader holding the file read %q; it must read %q, the whole word that was there when it opened", got, wordPause)
	}
	// And the word on disk is the new one, whole.
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the control file: %v", err)
	}

	if got := strings.TrimSpace(string(onDisk)); got != wordCancel {
		t.Errorf("the control file holds %q, want %q", got, wordCancel)
	}
}

// TestNoStrayFilesAreLeftBesideTheWord. Writing atomically buys the fix
// above at the price of a temporary beside the target, so the price is
// asserted: a Control that returned nil leaves the control file and nothing
// else in the task's directory.
func TestNoStrayFilesAreLeftBesideTheWord(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "CTRL-2", "no litter", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Control(s, tk, wordPause); err != nil {
		t.Fatalf("Control: %v", err)
	}

	dir, err := s.CreateTaskDir(tk.Repo.Path, tk.ID)
	if err != nil {
		t.Fatalf("CreateTaskDir: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("a temporary was left behind: %s", e.Name())
		}
	}
}
