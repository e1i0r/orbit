package view

// A run whose engine had nothing left to spend.

import (
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// TestTheBoardSaysWhichOfTheTwoItWas.
//
// "It ran out" and "it broke" are read by somebody deciding what to do next
// — wait, or hand the work to another engine, or go and look at a bug — and
// a row that said the same for both would send them to the log to find out.
func TestTheBoardSaysWhichOfTheTwoItWas(t *testing.T) {
	at := time.Now().UTC()

	one := Fold([]record.Event{
		{Kind: record.TaskCreated, At: at},
		{Kind: record.TaskStarted, At: at},
		{Kind: record.PhaseStarted, At: at, Phase: "implement", Data: map[string]string{"engine": "claude"}},
		{Kind: record.PhaseRanOut, At: at, Phase: "implement"},
		{Kind: record.TaskFailed, At: at},
	})

	if one.Reason.Key != ReasonRanOut {
		t.Errorf("the row reads %q, want it to say the allowance ran out", one.Reason.Key)
	}

	// The task.failed that follows says only that a run ended badly —
	// internal/task writes the same one for every way of ending badly — so
	// reading it as a failure here would throw away the one thing the
	// reader needs.
	if one.Phase != "implement" {
		t.Errorf("the row lost the phase it ran out in: %q", one.Phase)
	}

	if BandOf(one) != NeedsYou {
		t.Errorf("a run that ran out is in %v, and it needs somebody", BandOf(one))
	}
}

// TestARunThatBrokeStillReadsAsBroken, because nothing about the other
// ending is allowed to make this one quieter.
func TestARunThatBrokeStillReadsAsBroken(t *testing.T) {
	at := time.Now().UTC()

	one := Fold([]record.Event{
		{Kind: record.TaskCreated, At: at},
		{Kind: record.TaskStarted, At: at},
		{Kind: record.PhaseStarted, At: at, Phase: "implement"},
		{Kind: record.PhaseFailed, At: at, Phase: "implement"},
		{Kind: record.TaskFailed, At: at},
	})

	if one.Reason.Key != ReasonFailed {
		t.Errorf("a run that broke reads %q", one.Reason.Key)
	}
}
