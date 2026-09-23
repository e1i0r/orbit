//go:build !windows

package repo

// The priority a comparison's checks run at.

import (
	"strings"
	"testing"
)

// TestAComparisonsCheckGivesWayToEverythingElse. Two copies of a suite that
// takes every core, at the ordinary priority, left the window waiting on
// the CPU. A check runs at the lowest priority there is, and so does
// whatever it starts.
func TestAComparisonsCheckGivesWayToEverythingElse(t *testing.T) {
	got := runCheck(t.TempDir(), "ps -o nice= -p $$")
	if got.Failed != nil {
		t.Fatalf("the check could not run: %v", got.Failed)
	}

	if nice := strings.TrimSpace(got.Out); nice != "19" {
		t.Errorf("the check ran at niceness %q, want 19", nice)
	}
}
