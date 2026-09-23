package verb

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/queue"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/words"
)

// TestTheLineSaysWhoRunsWhoWaitsAndWhy. The queue runs where nobody sees it
// unless it is asked, and what it answers has to be enough to know why a
// task has not started.
func TestTheLineSaysWhoRunsWhoWaitsAndWhy(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	r := queue.Report{
		Max: 3, Ceiling: 85, Memory: 72, MemoryKnown: true, Service: 4242,
		Going:   []queue.Going{{ID: "Q-2", Since: now.Add(-2 * time.Minute), Phase: "implement"}},
		Waiting: []queue.Waiter{{ID: "Q-5", Since: now.Add(-45 * time.Second), Why: queue.WhySlots}},
	}

	got := line(words.For("en"), r, now)
	for _, want := range []string{
		"3 at a time", "85%", "72%", "process 4242", "Q-2", "implement",
		"1. Q-5", "no free slot",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the line does not carry %q:\n%s", want, got)
		}
	}
}

// TestATripNamesItsPlaceAndHowItEnded, and a failure's first line comes
// without the colours the engine printed it in.
func TestATripNamesItsPlaceAndHowItEnded(t *testing.T) {
	p := words.For("en")
	r := queue.Report{Waiting: []queue.Waiter{{ID: "Q-9", Why: queue.WhyMemory}}}

	if got := milestone(p, record.Event{Kind: record.TaskQueued}, r, "Q-9"); !strings.Contains(got, "1 in line") ||
		!strings.Contains(got, "memory") {
		t.Errorf("a queued task reads %q, want its place and why", got)
	}

	failed := record.Event{
		Kind: record.PhaseFailed, Phase: "implement",
		Data: map[string]string{"error": "\x1b[91mError:\x1b[0m Unexpected error\nat line 3"},
	}
	if got := milestone(p, failed, r, "Q-9"); got != "implement → failed: Error: Unexpected error" {
		t.Errorf("a failed phase reads %q", got)
	}
}
