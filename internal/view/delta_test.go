package view

// What an engine said about the change it made.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestADeltaWithNothingInItIsNotADelta. Three of the four fields empty is a
// whole answer — a change that adds no precondition has nothing to say under
// needs — but a task.delta with none of them said nothing at all.
func TestADeltaWithNothingInItIsNotADelta(t *testing.T) {
	if got := deltaOf(record.Event{Kind: record.TaskDelta}); got != nil {
		t.Errorf("an empty delta read as %+v", got)
	}

	if got := deltaOf(record.Event{Kind: record.PhaseStarted}); got != nil {
		t.Errorf("an event that is not a delta read as %+v", got)
	}

	one := deltaOf(record.Event{
		Kind: record.TaskDelta,
		Data: map[string]string{"guarantees": "the webhook retries on 5xx\n\n"},
	})
	if one == nil || !one.Any() {
		t.Fatalf("a delta with one field read as %+v", one)
	}

	// The trailing newline a record leaves is not a sentence.
	if len(one.Guarantees) != 1 {
		t.Errorf("one sentence read as %d: %v", len(one.Guarantees), one.Guarantees)
	}
}
