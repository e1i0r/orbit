package store

// The move to the flat tree, from the cases that are not the happy one: a
// run interrupted half way, a task whose directory has folders in it, and a
// copy that did not arrive whole.
//
// The check is the whole point of copying by hand rather than renaming — a
// rename is atomic and leaves nothing to compare against — so what it does
// when the comparison fails is worth being sure about.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestATaskWithFoldersInItComesUpWhole. A task's directory holds a folder
// per phase once a run has been through it, and a migration that copied only
// the files at the top would move a task and lose everything an engine said.
func TestATaskWithFoldersInItComesUpWhole(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer closed(t, s)

	was := filedTask(t, s, "/w/app", "ACME-1", map[string]string{
		"task.md": "retry the webhook on 5xx\n",
	})

	deep := filepath.Join(was, "phases", "1-implement")
	if err := os.MkdirAll(deep, dirMode); err != nil {
		t.Fatalf("make the phase directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(deep, "output.txt"), []byte("what it said"), fileMode); err != nil {
		t.Fatalf("write what the phase said: %v", err)
	}

	if _, err := s.Flatten(); err != nil {
		t.Fatalf("Flatten: %v", err)
	}

	dst, err := s.TaskDir("ACME-1")
	if err != nil {
		t.Fatalf("TaskDir: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(dst, "phases", "1-implement", "output.txt"))
	if err != nil {
		t.Fatalf("what the phase said did not come up: %v", err)
	}

	if string(body) != "what it said" {
		t.Errorf("it came up as %q", body)
	}
}

// TestARunInterruptedBetweenTheCopyAndTheLinkIsFinished.
//
// The copy happens before the link, because the file that holds the link
// lives inside the directory being copied into. A run killed between the two
// leaves a destination no repository claims — and reading that as a name
// collision named one repository as though it were two, and did so on every
// run for ever.
func TestARunInterruptedBetweenTheCopyAndTheLinkIsFinished(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer closed(t, s)

	filedTask(t, s, "/w/app", "ACME-1", map[string]string{"task.md": "pay the thing\n"})

	// The state a killed run leaves: the directory is up, and nothing
	// points at it.
	dst, err := s.TaskDir("ACME-1")
	if err != nil {
		t.Fatalf("TaskDir: %v", err)
	}

	if err := os.MkdirAll(dst, dirMode); err != nil {
		t.Fatalf("make the destination a killed run would have left: %v", err)
	}

	moved, err := s.Flatten()
	if err != nil {
		t.Fatalf("a migration finishing what a killed one left: %v", err)
	}

	if len(moved) != 0 {
		t.Errorf("it said it moved %v, and the directory was already up", moved)
	}

	joined, err := s.TaskRepos("ACME-1")
	if err != nil {
		t.Fatalf("read what the task is worked in: %v", err)
	}

	if len(joined) != 1 || joined[0] != "/w/app" {
		t.Errorf("the task is worked in %v, want the repository it came from", joined)
	}

	// And a third run over the same state moves nothing and says nothing,
	// which is what the ordinary second run does.
	if again, err := s.Flatten(); err != nil || len(again) != 0 {
		t.Errorf("a third run answered %v, %v", again, err)
	}
}

// TestACopyThatDidNotArriveWholeIsNotCalledCopied. Reading it back is the
// whole reason this is done by hand, and a mismatch has to be a failure
// naming both files rather than a task that moved and lost bytes.
func TestACopyThatDidNotArriveWholeIsNotCalledCopied(t *testing.T) {
	dir := t.TempDir()

	from := filepath.Join(dir, "task.md")
	if err := os.WriteFile(from, []byte("pay the thing"), fileMode); err != nil {
		t.Fatalf("write the source: %v", err)
	}

	// A destination that is a directory: the write fails, and the failure
	// names the file rather than the syscall.
	to := filepath.Join(dir, "dst")
	if err := os.MkdirAll(to, dirMode); err != nil {
		t.Fatalf("make the destination: %v", err)
	}

	err := copyFile(from, to)
	if err == nil {
		t.Fatal("a copy that could not be written was called copied")
	}

	if !strings.Contains(err.Error(), to) {
		t.Errorf("the failure is %q, want it to name where it was writing", err)
	}

	// And one whose source is not there names the source.
	err = copyFile(filepath.Join(dir, "not-there"), filepath.Join(dir, "anywhere"))
	if err == nil {
		t.Fatal("a copy of a file that is not there was called copied")
	}

	if !strings.Contains(err.Error(), "not-there") {
		t.Errorf("the failure is %q, want it to name what it could not read", err)
	}
}

// TestATreeThatCannotBeReadIsAFailureAndNotAnEmptyCopy. A source directory
// that is gone would otherwise copy nothing and answer that it had copied a
// task.
func TestATreeThatCannotBeReadIsAFailureAndNotAnEmptyCopy(t *testing.T) {
	dir := t.TempDir()

	err := copyTree(filepath.Join(dir, "not-there"), filepath.Join(dir, "dst"))
	if err == nil {
		t.Fatal("a tree that is not there was copied")
	}

	if !strings.Contains(err.Error(), "not-there") {
		t.Errorf("the failure is %q, want it to name what it could not read", err)
	}
}

// TestTheOldTreeIsLeftWhereItIs. It is what there is to compare against, and
// a migration that removed it would have nothing to answer "did this arrive"
// with the next time somebody asks.
func TestTheOldTreeIsLeftWhereItIs(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer closed(t, s)

	was := filedTask(t, s, "/w/app", "ACME-1", map[string]string{"task.md": "pay the thing\n"})

	if _, err := s.Flatten(); err != nil {
		t.Fatalf("Flatten: %v", err)
	}

	if _, err := os.Stat(filepath.Join(was, "task.md")); err != nil {
		t.Errorf("the old tree is gone: %v", err)
	}
}
