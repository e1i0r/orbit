package task

// The ceiling on what the supervisor thread puts in front of the model.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestHistoryKeepsTheWholeThreadWhenItFits is the ordinary case: nothing is
// dropped and nothing is announced.
func TestHistoryKeepsTheWholeThreadWhenItFits(t *testing.T) {
	got := history([]record.Event{
		{Kind: record.SupervisorMessage, Text: "first", Data: map[string]string{"by": "elio", "channel": "tui"}},
		{Kind: record.SupervisorMessage, Text: "second", Data: map[string]string{"by": "claude", "channel": "supervisor"}},
	})

	want := "[elio via tui]: first\n[claude via supervisor]: second\n"
	if got != want {
		t.Errorf("history = %q, want %q", got, want)
	}
}

// TestHistoryDefaultsWhoAndWhere pins the two fallbacks a turn written
// without them gets.
func TestHistoryDefaultsWhoAndWhere(t *testing.T) {
	got := history([]record.Event{{Kind: record.SupervisorMessage, Text: "bare"}})
	if got != "[operator via tui]: bare\n" {
		t.Errorf("history = %q, want the operator/tui defaults", got)
	}
}

// TestHistoryNamesTheTaskATurnIsAbout keeps the task id in the line.
func TestHistoryNamesTheTaskATurnIsAbout(t *testing.T) {
	got := history([]record.Event{{
		Kind: record.SupervisorMessage,
		Text: "look at this one",
		Data: map[string]string{"by": "mcp", "channel": "mcp", "task_id": "TASK-9"},
	}})
	if !strings.Contains(got, "(TASK-9)") {
		t.Errorf("history = %q, want the task id in the line", got)
	}
}

// TestHistoryKeepsTheRecentTurnsAndSaysHowManyItDropped is the fix.
//
// The thread is global and append-only and nothing prunes it, so an
// uncapped history put every call before this one into every prompt. What
// has to survive the cap is the end of the conversation — the turn being
// answered is a reply to the last ones — and the fact that there was more.
func TestHistoryKeepsTheRecentTurnsAndSaysHowManyItDropped(t *testing.T) {
	// Turns of about a kilobyte each, far past the ceiling in total.
	body := strings.Repeat("x", 1024)

	var events []record.Event
	for i := range 200 {
		events = append(events, record.Event{
			Kind: record.SupervisorMessage,
			Text: fmt.Sprintf("turn-%03d %s", i, body),
			Data: map[string]string{"by": "elio", "channel": "tui"},
		})
	}

	got := history(events)
	if len(got) > maxHistory+200 {
		t.Errorf("history is %d bytes, over the %d ceiling", len(got), maxHistory)
	}

	if !strings.Contains(got, "turn-199") {
		t.Error("the most recent turn was dropped; the cap has to keep the end of the conversation")
	}

	if strings.Contains(got, "turn-000") {
		t.Error("the oldest turn survived a thread far past the ceiling")
	}

	if !strings.Contains(got, "earlier turns are not shown") {
		t.Errorf("history dropped turns without saying so: %.120q", got)
	}
}

// TestHistoryOfAnEmptyThreadIsEmpty: a supervisor nobody has spoken to yet
// gets no history section at all, not a header over nothing.
func TestHistoryOfAnEmptyThreadIsEmpty(t *testing.T) {
	if got := history(nil); got != "" {
		t.Errorf("history(nil) = %q, want empty", got)
	}
}
