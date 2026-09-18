package task

// A marker left behind by a machine that has since rebooted.

import (
	"testing"
	"time"
)

// TestARunThatBeganBeforeTheMachineBootedIsGone.
//
// A pid outlives nothing across a reboot, but the number is handed straight
// back out: the process table starts again from the low numbers, and
// something else is wearing the pid the marker names within a minute of the
// machine coming up. Asking the pid would answer yes about a stranger.
//
// The slack is the other half. The marker's timestamp is written to the
// second and the boot time is read to the second, so a run that began in the
// same second the machine finished booting must not be read as having begun
// before it — which would kill a live run off the board.
func TestARunThatBeganBeforeTheMachineBootedIsGone(t *testing.T) {
	boot, ok := bootTime()
	if !ok {
		t.Skip("this machine does not say when it booted, so there is nothing to read a marker against")
	}

	for _, one := range []struct {
		why     string
		started time.Time
		stale   bool
	}{
		{"a marker with no timestamp says nothing either way", time.Time{}, false},
		{"begun in the same second the machine came up", boot, false},
		{
			"begun a fraction before it, which is the same second written twice",
			boot.Add(-900 * time.Millisecond), false,
		},
		{"begun well before the machine came up", boot.Add(-5 * time.Second), true},
		{"begun after it, which is every live run there is", boot.Add(time.Hour), false},
	} {
		if got := staleAcrossBoot(one.started); got != one.stale {
			t.Errorf("a run %s reads as stale: %v, want %v", one.why, got, one.stale)
		}
	}
}
