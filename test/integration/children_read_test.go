//go:build integration

package integration

// What sweep is allowed to kill, asked of a listing rather than of the
// machine: the rule is two lines of code and the whole of the risk, since
// every pid it answers is sent SIGKILL.

import "testing"

func TestLeftoversAreOnlyThisHarnessAndOnlyOrphans(t *testing.T) {
	listing := `    1     0 /sbin/launchd
14097     1 /var/folders/T/orbit-integration-595751429/orbit run -repo /tmp/ledger LED-3
14098 14052 /var/folders/T/orbit-integration-777/orbit run -repo /tmp/ledger LED-4
  900     1 /usr/local/bin/orbit run -repo /home/me/code PAY-9
  901     1 /var/folders/T/orbit-integration-595751429/claude
`

	got := leftovers(listing)
	want := []int{14097, 901}

	if len(got) != len(want) {
		t.Fatalf("leftovers %v, want %v", got, want)
	}

	for i, pid := range want {
		if got[i] != pid {
			t.Errorf("leftover %d is %d, want %d", i, got[i], pid)
		}
	}
}
