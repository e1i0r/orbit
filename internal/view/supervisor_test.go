package view

import (
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

func TestSupervisorThreadDefaultsWhoAndWhere(t *testing.T) {
	lines := SupervisorThread([]record.Event{{Kind: record.SupervisorMessage, Text: "bare"}})
	if len(lines) != 1 {
		t.Fatalf("lines = %d, want 1", len(lines))
	}
	if lines[0].By != "operator" || lines[0].Channel != "tui" {
		t.Errorf("by/channel = %q/%q, want operator/tui", lines[0].By, lines[0].Channel)
	}
}

// TestSupervisorThreadMarksARetractedLineAndDropsTheRetraction: a withdrawn
// turn stays in the thread, marked. Hiding it would leave the lines around
// it answering something a reader cannot see, and the retraction itself is
// bookkeeping about a line above it rather than a line of its own.
func TestSupervisorThreadMarksARetractedLineAndDropsTheRetraction(t *testing.T) {
	at := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)
	lines := SupervisorThread([]record.Event{
		{At: at, Kind: record.SupervisorMessage, Text: "the one I regret"},
		{At: at.Add(time.Minute), Kind: record.SupervisorMessage, Text: "still standing"},
		{At: at.Add(2 * time.Minute), Kind: record.SupervisorRetracted, Data: map[string]string{"at": record.Stamp(at)}},
	})
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2: the two turns, without the line that takes one back", len(lines))
	}
	if !lines[0].Retracted {
		t.Error("the withdrawn turn is not marked")
	}
	if lines[0].Text != "the one I regret" {
		t.Errorf("the withdrawn turn lost its text: %q", lines[0].Text)
	}
	if lines[1].Retracted {
		t.Error("a turn nobody took back is marked as retracted")
	}
}
