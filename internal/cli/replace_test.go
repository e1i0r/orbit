package cli

// Putting the new binary in place of the old one: the last step of an
// upgrade, the one that cannot be taken back, and the one nothing ran until
// a reader ran it.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTheNewBinaryLandsWhereTheOldOneWas(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orbit")

	if err := os.WriteFile(path, []byte("the old orbit"), 0o755); err != nil {
		t.Fatalf("stand an old orbit up: %v", err)
	}

	if err := replaceFile(path, []byte("the new orbit")); err != nil {
		t.Fatalf("replaceFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read what was installed: %v", err)
	}

	if string(got) != "the new orbit" {
		t.Errorf("the installed orbit is %q, want the new one", got)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat what was installed: %v", err)
	}

	// Without this the reader's next orbit is the 0600 CreateTemp made, and
	// their shell answers "permission denied" to a command that worked a
	// second ago.
	if info.Mode().Perm() != 0o755 {
		t.Errorf("the installed orbit is mode %v, so nobody can run it", info.Mode().Perm())
	}

	leftovers(t, dir)
}

// leftovers is the check that no half-written orbit is still sitting next to
// the binary. There is nothing to clean these up: they are named for a
// temporary file but they live in the reader's bin directory for good.
func leftovers(t *testing.T, dir string) {
	t.Helper()

	found, err := filepath.Glob(filepath.Join(dir, "orbit-update-*"))
	if err != nil {
		t.Fatalf("look for temporary files: %v", err)
	}

	if len(found) > 0 {
		t.Errorf("%d half-written orbit(s) left beside the binary: %v", len(found), found)
	}
}

// TestAnUpdateThatCannotBeWrittenLeavesTheOldOrbitStanding. Failing is
// allowed; failing halfway is not. A reader whose bin directory is not theirs
// to write keeps the orbit they had.
func TestAnUpdateThatCannotBeWrittenLeavesTheOldOrbitStanding(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes into a directory that has no write bit")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "orbit")

	if err := os.WriteFile(path, []byte("the old orbit"), 0o755); err != nil {
		t.Fatalf("stand an old orbit up: %v", err)
	}

	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("take the write bit off the directory: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Errorf("give the directory its write bit back: %v", err)
		}
	})

	if err := replaceFile(path, []byte("the new orbit")); err == nil {
		t.Fatal("a directory that takes no new files reported an orbit installed in it")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the orbit that was there before: %v", err)
	}

	if string(got) != "the old orbit" {
		t.Errorf("the orbit on disk is now %q; a failed update touched it", got)
	}
}

// TestAMoveThatFailsTakesTheHalfWrittenOrbitWithIt is the rename refusing
// after the bytes are already on disk. Left behind, every failed upgrade
// adds another orbit-update-* beside the binary and says nothing about it.
func TestAMoveThatFailsTakesTheHalfWrittenOrbitWithIt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orbit")

	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf("put a directory where the binary should be: %v", err)
	}

	if err := replaceFile(path, []byte("the new orbit")); err == nil {
		t.Fatal("a directory was reported as replaced by a binary")
	}

	leftovers(t, dir)
}

// TestATemporaryFileThatWillNotTakeTheNewOrbitSaysSo.
//
// The file is opened for reading and is otherwise fine: the mode can still be
// set on it and it still closes. So the only thing that can fail here is the
// write, and a write whose error goes unread is an empty file made runnable
// and moved on top of the reader's orbit.
func TestATemporaryFileThatWillNotTakeTheNewOrbitSaysSo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orbit-update-000")

	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("stand a temporary file up: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open it for reading only: %v", err)
	}

	if err := fill(f, []byte("the new orbit")); err == nil {
		t.Error("a file that takes no writes reported the new orbit written into it")
	}
}

// TestAFileThatCannotBeMadeRunnableIsRefused: /dev/null takes any bytes at
// all and belongs to root, so it is the one file that gets past the write and
// stops at the mode.
func TestAFileThatCannotBeMadeRunnableIsRefused(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root sets the mode of a file it does not own")
	}

	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}

	if err := fill(f, []byte("the new orbit")); err == nil {
		t.Error("a file whose mode could not be set was reported as ready to run")
	}
}

// TestAnOrbitLargerThanTheCeilingIsNotInstalledInHalf. This read stopped at
// the ceiling and answered no error, so an archive one byte over it was cut
// in the middle and the piece was written over the running orbit.
func TestAnOrbitLargerThanTheCeilingIsNotInstalledInHalf(t *testing.T) {
	got, err := binaryFrom(archiveOf(t, "orbit", bytes.Repeat([]byte("x"), maxDownload+1)))
	if err == nil {
		t.Fatalf("an oversized archive answered %d bytes to write over orbit", len(got))
	}

	if !strings.Contains(err.Error(), "larger than") {
		t.Errorf("the refusal does not say the binary was too large: %v", err)
	}
}
