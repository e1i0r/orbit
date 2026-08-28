package task

// The two ends of one defence: which pids a marker may name, and which
// numbers may be handed to a signal. They are tested together because they
// are the same rule written twice on purpose — a marker reading `pid: 1`
// must never become kill(-1), which is every process the user may signal.
//
// Nothing here signals anything. Both ends are functions of their arguments
// so the boundary can be asserted with no process to aim at, which is also
// the only way to test it safely: the number under test is precisely the one
// that would take the test runner down with it.

import (
	"errors"
	"strings"
	"testing"
)

func TestAMarkerMayNotNameAPidThatIsNotARunToSignal(t *testing.T) {
	for _, tc := range []struct{ pid, why string }{
		{"1", "init, and negated for a group signal it is every process this user may signal"},
		{"0", "kill(2) reads zero as every process in the caller's own group"},
		{"-1", "already the number that means everything"},
		{"-4711", "a group named by a negative number"},
	} {
		body := "pid: " + tc.pid + "\nstarted: 2026-08-23T09:14:02Z\n"

		got, err := parsePid(body)
		if err == nil {
			t.Errorf("parsePid(%q) = %d with no error, want a refusal: %s", body, got, tc.why)
			continue
		}
		// The reader has to be told which number was refused, or a damaged
		// marker is a mystery rather than a line to go and look at.
		if !strings.Contains(err.Error(), tc.pid) {
			t.Errorf("parsePid(%q) refused without saying what it read: %v", body, err)
		}
	}
}

func TestAMarkerNamingARealProcessIsRead(t *testing.T) {
	body := "pid: 4711\nstarted: 2026-08-23T09:14:02Z\n"

	pid, err := parsePid(body)
	if err != nil || pid != 4711 {
		t.Errorf("parsePid(%q) = (%d, %v), want (4711, nil)", body, pid, err)
	}
}

func TestOnlyARunThatLeadsItsOwnGroupIsSignalledAsAGroup(t *testing.T) {
	for _, tc := range []struct {
		name      string
		pid, pgid int
		gerr      error
		want      int
	}{
		{"leads its own group, as Start spawns it", 4711, 4711, nil, -4711},
		{"in a group it did not start", 4711, 900, nil, 4711},
		{"getpgid could not say", 4711, 0, errors.New("no such process"), 4711},
		{"a pgid of 1 is never negated", 1, 1, nil, 1},
	} {
		if got := killTarget(tc.pid, tc.pgid, tc.gerr); got != tc.want {
			t.Errorf("%s: killTarget(%d, %d, %v) = %d, want %d",
				tc.name, tc.pid, tc.pgid, tc.gerr, got, tc.want)
		}
	}
}
