package cells

// Elapsed reads as nothing for no time, seconds under a minute, whole
// minutes under an hour, and hours with their minutes after that.

import (
	"testing"
	"time"
)

// TestElapsedSaysHowLongAgoInWords.
func TestElapsedSaysHowLongAgoInWords(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

	for text, since := range map[string]time.Time{
		"":    {},
		"30s": now.Add(-30 * time.Second),
		"5m":  now.Add(-5 * time.Minute),
		"1h":  now.Add(-65 * time.Minute),
		"2d":  now.Add(-49 * time.Hour),
	} {
		if got := Elapsed(now, since); got != text {
			t.Errorf("elapsed reads %q, want %q", got, text)
		}
	}

	if got := Elapsed(now, now.Add(time.Minute)); got != "0s" {
		t.Errorf("a time to come reads %q, want it clamped to now", got)
	}
}
