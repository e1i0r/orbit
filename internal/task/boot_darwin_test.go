//go:build darwin

package task

import (
	"encoding/binary"
	"testing"
	"time"
)

// timeval is what the kernel would hand back for a machine that booted at
// this moment: seconds, then microseconds, both little-endian.
func timeval(sec uint64) string {
	b := make([]byte, 16)
	binary.LittleEndian.PutUint64(b[:8], sec)

	return string(b)
}

// TestBootTimeIsReadOffTheSecondsOfTheTimeval, and off nothing else. The
// microseconds are deliberately dropped: the other side of every comparison
// is a marker written to the second.
func TestBootTimeIsReadOffTheSecondsOfTheTimeval(t *testing.T) {
	want := time.Date(2026, 8, 29, 7, 15, 0, 0, time.UTC)

	got, ok := parseBootTime(timeval(uint64(want.Unix())))
	if !ok {
		t.Fatal("a well-formed timeval was refused")
	}

	if !got.Equal(want) {
		t.Errorf("parseBootTime = %s, want %s", got, want)
	}
}

// TestAnAnswerThatIsNotABootTimeIsRefused covers both ways the kernel can
// leave this without one. Neither is an error: a machine that cannot say
// when it booted makes Alive fall back to asking only whether the pid is
// there, which is what it did everywhere before this file existed. What
// matters is that it does not answer with the epoch, because a run started
// after 1970 is every run there has ever been — every task would be read as
// stale across a boot and killed off the board.
func TestAnAnswerThatIsNotABootTimeIsRefused(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{"too few bytes to hold a seconds field", "\x01\x02\x03\x04"},
		{"nothing at all", ""},
		{"a seconds field of zero", timeval(0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := parseBootTime(tc.raw); ok {
				t.Errorf("parseBootTime = (%s, true), want a refusal", got)
			}
		})
	}
}

// TestThisMachineSaysWhenItBooted is the half no test can fake: the sysctl
// itself. It asserts only what is true of any running darwin — it booted,
// and it booted in the past — because anything sharper would be a fact about
// the machine the suite happens to be on.
func TestThisMachineSaysWhenItBooted(t *testing.T) {
	got, ok := bootTime()
	if !ok {
		t.Fatal("kern.boottime gave no answer on a machine that is running")
	}

	if got.After(time.Now()) {
		t.Errorf("this machine booted at %s, which has not happened yet", got)
	}

	if got.Before(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("this machine booted at %s, which is the clock being wrong or the bytes being misread", got)
	}
}
