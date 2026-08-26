package ui

// FormatLatency has one job: turn a millisecond count into the role a
// person would assign it on sight. These tests pin the three bands and
// their edges, and check that the answer never depends on when it's asked.

import (
	"fmt"
	"testing"
)

func TestFormatLatencyPicksRoleByThreshold(t *testing.T) {
	cases := []struct {
		name string
		ms   int64
		role Role
	}{
		{"zero is fast", 0, OK},
		{"just under the green ceiling", 99, OK},
		{"at the yellow threshold", 100, Warn},
		{"mid yellow", 250, Warn},
		{"just under the yellow ceiling", 499, Warn},
		{"at the red threshold", 500, Bad},
		{"well past red", 5000, Bad},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := FormatLatency(c.ms)
			want := Paint(c.role).Render(fmt.Sprintf("%dms", c.ms))
			if got != want {
				t.Errorf("FormatLatency(%d) = %q, want %q", c.ms, got, want)
			}
		})
	}
}

// TestFormatLatencyIsPure keeps FormatLatency out of the event loop, the
// same property theme_test.go checks for Paint: same input, same output,
// no clock or terminal consulted in between.
func TestFormatLatencyIsPure(t *testing.T) {
	first, second := FormatLatency(42), FormatLatency(42)
	if first != second {
		t.Errorf("FormatLatency(42) painted %q and then %q", first, second)
	}
}
