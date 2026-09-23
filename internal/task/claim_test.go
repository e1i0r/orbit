package task

// Who holds a task, when two runs of it start at once.

import (
	"os"
	"os/exec"
	"testing"

	"github.com/e1i0r/orbit/internal/store"
)

// TestAClaimNeverWritesOverALiveOne.
//
// Alive is asked before a run is spawned and again inside it, and a marker
// can appear between the two: `orbit task start` looks, spawns a process,
// and that process looks and writes. Two starts of one task inside that
// window — the window's autopilot, and a reader pressing enter a moment
// later — both saw nothing and both wrote, and the second marker replaced
// the first. Two engines then worked in one worktree, on one task, and
// only one of them could be cancelled.
//
// So the claim itself refuses: whatever the look before it said.
func TestAClaimNeverWritesOverALiveOne(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)

	// A process that is really there, standing in for the run that won
	// the race.
	other := exec.Command("sleep", "30")
	if err := other.Start(); err != nil {
		t.Fatalf("start a process to hold the task: %v", err)
	}

	t.Cleanup(func() {
		if err := other.Process.Kill(); err != nil {
			t.Logf("kill the holder: %v", err)
		}

		if _, err := other.Process.Wait(); err != nil {
			t.Logf("wait for the holder: %v", err)
		}
	})

	if _, err := mark(s, tk, other.Process.Pid); err != nil {
		t.Fatalf("plant the other run's marker: %v", err)
	}

	if _, err := claim(s, tk, os.Getpid()); err == nil {
		t.Error("a second run claimed a task another process is running")
	}

	// And the marker still names the run that got there first.
	held, _, found, err := readMarker(s, tk)
	if err != nil || !found {
		t.Fatalf("the marker is gone after a refused claim: found=%v err=%v", found, err)
	}

	if held != other.Process.Pid {
		t.Errorf("the marker names %d, want the process that holds it (%d)", held, other.Process.Pid)
	}
}

// TestAClaimTakesOverFromARunThatIsGone. The other half: a marker left
// behind by a run that died is not a claim, and the task can be run again
// without anybody clearing up by hand.
func TestAClaimTakesOverFromARunThatIsGone(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)

	// A pid nothing answers to.
	if _, err := mark(s, tk, 9999999); err != nil {
		t.Fatalf("plant a stale marker: %v", err)
	}

	release, err := claim(s, tk, os.Getpid())
	if err != nil {
		t.Fatalf("claim a task whose run is gone: %v", err)
	}

	held, _, found, err := readMarker(s, tk)
	if err != nil || !found {
		t.Fatalf("the marker is gone after a claim: found=%v err=%v", found, err)
	}

	if held != os.Getpid() {
		t.Errorf("the marker names %d, want this process (%d)", held, os.Getpid())
	}

	release()

	if _, _, found, err = readMarker(s, tk); err != nil || found {
		t.Errorf("the marker outlived the run that made it: found=%v err=%v", found, err)
	}
}

// TestTwoClaimsOnOneTaskLeaveOneHolder, whatever order they arrive in: the
// file is the lock, and the operating system decides.
func TestTwoClaimsOnOneTaskLeaveOneHolder(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)

	path, err := s.RunPath(tk.ID)
	if err != nil {
		t.Fatalf("RunPath: %v", err)
	}

	first, err := store.WriteIfAbsent(path, []byte("pid: 1\n"))
	if err != nil || !first {
		t.Fatalf("the first claim did not land: %v %v", first, err)
	}

	second, err := store.WriteIfAbsent(path, []byte("pid: 2\n"))
	if err != nil {
		t.Fatalf("the second claim: %v", err)
	}

	if second {
		t.Error("two processes both claimed the same task")
	}

	body, err := os.ReadFile(path)
	if err != nil || string(body) != "pid: 1\n" {
		t.Errorf("the file says %q, want the first claim's own words", body)
	}
}
