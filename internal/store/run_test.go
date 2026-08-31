package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunPathSitsInsideTheTask(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	dir, err := s.TaskDir("ACME-1")
	if err != nil {
		t.Fatalf("TaskDir: %v", err)
	}

	path, err := s.RunPath("ACME-1")
	if err != nil {
		t.Fatalf("RunPath: %v", err)
	}

	if filepath.Dir(path) != dir {
		t.Errorf("run file at %q, expected inside %q", path, dir)
	}

	if filepath.Base(path) != "run" {
		t.Errorf("run file is %q, want run", filepath.Base(path))
	}
}

func TestRunPathCreatesNothing(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	path, err := s.RunPath("ACME-1")
	if err != nil {
		t.Fatalf("RunPath: %v", err)
	}

	if _, statErr := os.Stat(path); statErr == nil {
		t.Error("RunPath created the run file merely by being asked where it would be")
	} else if !os.IsNotExist(statErr) {
		t.Errorf("stat %q: %v", path, statErr)
	}
}

func TestRunPathRejectsEscapingID(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = s.RunPath("../../escape")
	if err == nil {
		t.Error("escaped path id was accepted")
	}
}
