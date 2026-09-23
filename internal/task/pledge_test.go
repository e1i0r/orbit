package task

// A run counts from the moment it is spawned, not from when it gets round
// to saying so.

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestASpawnedRunCountsBeforeItHasDoneAnything. The marker is written in
// the child's name by whoever spawned it, so a look in the next instant
// sees a run going, and the queue does not start another in its slot.
func TestASpawnedRunCountsBeforeItHasDoneAnything(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)

	child := exec.Command("sleep", "30")
	if err := child.Start(); err != nil {
		t.Fatalf("start a process to stand in for the run: %v", err)
	}

	t.Cleanup(func() {
		if err := child.Process.Kill(); err != nil {
			t.Logf("kill the child: %v", err)
		}

		if _, err := child.Process.Wait(); err != nil {
			t.Logf("wait for the child: %v", err)
		}
	})

	if err := pledge(s, tk, child.Process.Pid); err != nil {
		t.Fatalf("pledge: %v", err)
	}

	held, alive, err := Alive(s, tk)
	if err != nil || !alive || held != child.Process.Pid {
		t.Errorf("Alive = %d, %v, %v; want the spawned run, going", held, alive, err)
	}

	// Pledged twice in the same name is the same marker, not a refusal.
	if err := pledge(s, tk, child.Process.Pid); err != nil {
		t.Errorf("a second pledge for the same run was refused: %v", err)
	}
}

// TestARunKeepsTheMarkerPledgedInItsName. The child finds its own pid on
// the marker and holds it, rather than refusing a run that is itself; and
// letting go takes it off.
func TestARunKeepsTheMarkerPledgedInItsName(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)

	if err := pledge(s, tk, os.Getpid()); err != nil {
		t.Fatalf("pledge: %v", err)
	}

	release, err := hold(s, tk)
	if err != nil {
		t.Fatalf("a run refused the marker pledged in its own name: %v", err)
	}

	release()

	if _, _, found, err := readMarker(s, tk); err != nil || found {
		t.Errorf("the marker is still there after the run let go: found=%v err=%v", found, err)
	}
}

// TestARunThatNeverBeganSaysSo. A task whose run could not even read it
// ends as failed, with the reason, rather than with nothing written.
func TestARunThatNeverBeganSaysSo(t *testing.T) {
	s, r := fixture(t)
	tk := written(t, s, r)

	if err := NotStarted(s, tk.ID, errors.New("flow \"gone\" not found")); err != nil {
		t.Fatalf("NotStarted: %v", err)
	}

	events, err := Events(s, tk)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}

	last := events[len(events)-1]
	if last.Kind != record.TaskFailed || last.Text != "flow \"gone\" not found" {
		t.Errorf("the last event is %s %q, want task.failed with the reason", last.Kind, last.Text)
	}
}
