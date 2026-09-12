package view

// The record's vocabulary as this package reads it, kind by kind.
//
// log_test.go already asks that every kind the record writes has SOME word —
// the assertion that keeps a newer build's log from arriving as blank lines.
// That question is answerable with one comparison, and it is the wrong one to
// stop at: a switch that answered EntryStuck for task.over_budget would pass
// it, and would draw a budget overrun as a stuck task.
//
// So this is the table, one row per kind, naming the kind it has to read as.
// The switch is a table in the code and it is tested as one, because the
// alternative is a test that only ever notices a kind going missing and never
// notices a kind going wrong.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestEveryKindReadsAsItsOwnWord.
func TestEveryKindReadsAsItsOwnWord(t *testing.T) {
	cases := []struct {
		kind string
		want EntryKind
	}{
		{record.TaskCreated, EntryWritten},

		{record.TaskStarted, EntryStarted},
		{record.PhaseStarted, EntryStarted},

		{record.TaskFinished, EntryFinished},
		{record.PhaseFinished, EntryFinished},

		{record.TaskFailed, EntryFailed},
		{record.PhaseFailed, EntryFailed},

		{record.TaskCancelled, EntryCancelled},
		{record.PhaseCancelled, EntryCancelled},

		{record.TaskRequeued, EntryRequeued},
		{record.TaskTimedOut, EntryTimedOut},
		{record.TaskAbandoned, EntryAbandoned},
		{record.TaskRead, EntryRead},

		{record.PhaseWaiting, EntryWaiting},
		{record.PhaseResumed, EntryResumed},
		{record.PhaseRetried, EntryRetried},

		{record.GatePassed, EntryGatePassed},
		{record.GateFailed, EntryGateFailed},

		{record.PhaseThought, EntryThought},
		{record.PhaseToolCall, EntryToolCall},
		{record.PhaseRefused, EntryRefused},

		{record.TaskNoted, EntryNoted},
		{record.TaskDialogue, EntryDialogue},

		{record.TaskStuck, EntryStuck},
		{record.TaskOverBudget, EntryOverBudget},
		{record.TaskOverDiff, EntryOverDiff},
		{record.TaskNewDependency, EntryNewDependency},
		{record.TaskContradicts, EntryContradicts},

		{record.LoopChecked, EntryLoopChecked},
		{record.DependencyApproved, EntryApproved},

		{record.DecisionMade, EntryDecision},
		{record.DecisionSuperseded, EntrySuperseded},

		{record.RepoJoined, EntryRepoJoined},
		{record.DeliverAsked, EntryDeliverAsked},
		{record.DeliverAnswered, EntryDeliverAnswered},

		// Not a kind anything wrote: the reader synthesises it where a line
		// would not parse.
		{record.Unreadable, EntryUnreadable},
	}

	for _, c := range cases {
		if got := (Entry{Kind: c.kind}).What(); got != c.want {
			t.Errorf("%q reads as %v, want %v", c.kind, got, c.want)
		}
	}
}

// TestTwoKindsDoNotReadAsEachOther.
//
// The failure the table above cannot catch on its own: a switch that hands
// the same word to two kinds that mean different things. task.failed and
// task.cancelled are the pair worth naming, because one is a thing that broke
// and the other is a thing somebody stopped, and the reader acts on the
// difference.
func TestTwoKindsDoNotReadAsEachOther(t *testing.T) {
	failed := (Entry{Kind: record.TaskFailed}).What()
	cancelled := (Entry{Kind: record.TaskCancelled}).What()

	if failed == cancelled {
		t.Errorf("a task that broke and a task somebody stopped both read as %v", failed)
	}

	overBudget := (Entry{Kind: record.TaskOverBudget}).What()
	stuck := (Entry{Kind: record.TaskStuck}).What()

	if overBudget == stuck {
		t.Errorf("a task over its budget and a task that is stuck both read as %v", overBudget)
	}
}

// TestAKindNothingWroteIsUnknownAndKeepsItsOwnWord.
//
// The whole reason What is a method and not a field: a log written by a newer
// build reaches an older one holding a kind it has never heard of, and the
// task view still draws the line — unstyled, but drawn. A dropped line is the
// worst answer for a reader who opened this screen to find out what happened.
func TestAKindNothingWroteIsUnknownAndKeepsItsOwnWord(t *testing.T) {
	e := Entry{Kind: "task.teleported"}

	if got := e.What(); got != EntryUnknown {
		t.Errorf("a kind nothing wrote reads as %v, want EntryUnknown", got)
	}

	if e.Kind != "task.teleported" {
		t.Errorf("the unknown entry lost the record's own word: %q", e.Kind)
	}
}
