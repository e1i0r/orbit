package cli

// Which of two versions is ahead, which is the one thing standing between a
// locally built orbit and being replaced by an older release.

import "testing"

// TestNewerThan.
func TestNewerThan(t *testing.T) {
	for _, c := range []struct {
		released, running string
		want              bool
	}{
		{"1.2.3", "1.2.2", true},
		{"1.3.0", "1.2.9", true},
		{"2.0.0", "1.9.9", true},
		{"1.2.3", "1.2.3", false},
		{"1.2.3", "1.2.4", false},
		{"1.2.3", "1.10.0", false},
		{"1.2.3", "2.0.0", false},
		// Ten is after nine, which is the comparison a string gets wrong.
		{"0.10.0", "0.9.0", true},
		// Neither is a version this can order, so anything different is
		// worth installing — the rule that was there before.
		{"nightly", "1.2.3", true},
		{"1.2.3-rc1", "1.2.3", true},
		{"nightly", "nightly", false},
	} {
		if got := newerThan(c.released, c.running); got != c.want {
			t.Errorf("newerThan(%q, %q) = %v, want %v", c.released, c.running, got, c.want)
		}
	}
}
