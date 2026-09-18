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

// TestARunThatWasRefusedWhatItNeededIsItsOwnEnding.
//
// Read by its exit code alone this was a success, which is how a task that
// did nothing came to sit in done. The three endings send a reader to three
// different places — a bug to look at, an allowance to wait for, and a
// permission to widen — so the fold keeps them apart and names the tool that
// was refused.
func TestARunThatWasRefusedWhatItNeededIsItsOwnEnding(t *testing.T) {
	at := time.Now().UTC()

	one := Fold([]record.Event{
		{Kind: record.TaskCreated, At: at},
		{Kind: record.TaskStarted, At: at},
		{Kind: record.PhaseStarted, At: at, Phase: "implement", Data: map[string]string{"engine": "claude"}},
		{
			Kind: record.PhaseDenied, At: at, Phase: "implement",
			Data: map[string]string{"tool": "Bash", "cost": "0.05"},
		},
		{Kind: record.TaskFailed, At: at},
	})

	if one.Reason.Key != ReasonDenied {
		t.Fatalf("the row reads %q, want it to say the run was refused", one.Reason.Key)
	}

	var tool string

	for _, a := range one.Reason.Args {
		if a.Name == "tool" {
			tool = a.Value
		}
	}

	if tool != "Bash" {
		t.Errorf("the row names the tool %q, want the one that was refused", tool)
	}

	// What a phase spent it spent, whether it got anywhere or not.
	if one.Cost == 0 {
		t.Error("a run that was refused spent nothing, want what the phase cost")
	}

	// And the task.failed behind it does not overwrite the reason: it says
	// only that a run ended badly, and this row already says which way.
	if one.Phase != "implement" {
		t.Errorf("the row lost the phase it was refused in: %q", one.Phase)
	}

	if BandOf(one) != NeedsYou {
		t.Errorf("a run that was refused is in %v, and it needs somebody", BandOf(one))
	}
}

// TestAListOfNamesIsReadByAPerson. The record writes them tight because a
// field of a log is a field of a log; a row on a board is a sentence, and
// "claude,codex,opencode" is the seam where a reader can see the machine.
func TestAListOfNamesIsReadByAPerson(t *testing.T) {
	cases := map[string]string{
		"claude,codex,opencode": "claude, codex, opencode",
		"claude":                "claude",
		"":                      "",
	}

	for tight, want := range cases {
		if got := spaced(tight); got != want {
			t.Errorf("%q is read as %q, want %q", tight, got, want)
		}
	}
}
