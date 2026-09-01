package view

// The second fold: a whole record into the entries the task view draws. The
// first one, into a single Task, is in fold_test.go, and the two share this
// package's fixtures deliberately — a case that folds one way and not the
// other is a disagreement between the board and the screen below it.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestTheLogIsEveryEventInTheOrderItWasWritten: the fold is total. A record
// with a kind this build has never heard of still produces a line, because
// the reader who opened this screen is the reader a dropped line hurts.
func TestTheLogIsEveryEventInTheOrderItWasWritten(t *testing.T) {
	entries := Log([]record.Event{
		{At: at(0), Kind: record.TaskCreated, Text: "Retry the webhook on 5xx"},
		{At: at(1), Kind: "task.teleported"},
		{At: at(2), Kind: record.TaskStarted},
	})
	if len(entries) != 3 {
		t.Fatalf("three events folded to %d entries, want 3", len(entries))
	}

	if got := entries[1].What(); got != EntryUnknown {
		t.Errorf("a kind this build does not know reads as %v, want EntryUnknown", got)
	}

	if entries[1].Kind != "task.teleported" {
		t.Errorf("the unknown entry says %q, want the record's own word kept", entries[1].Kind)
	}

	if !entries[2].At.Equal(at(2)) {
		t.Errorf("the third entry is timed %v, want %v", entries[2].At, at(2))
	}
}

// TestTheAttemptIsAPropertyOfTheEntry is the seam the log tab draws, and the
// reason it is counted here rather than in the window: a view that matched
// on a kind while it was painting would disagree with itself the day another
// kind opens an attempt.
func TestTheAttemptIsAPropertyOfTheEntry(t *testing.T) {
	entries := Log([]record.Event{
		{At: at(0), Kind: record.TaskCreated, Text: "Retry the webhook on 5xx"},
		{At: at(1), Kind: record.TaskStarted},
		{At: at(2), Kind: record.PhaseFailed, Phase: "gates"},
		{At: at(3), Kind: record.TaskFailed},
		{At: at(4), Kind: record.TaskStarted},
		{At: at(5), Kind: record.PhaseFinished, Phase: "gates"},
	})

	want := []int{0, 1, 1, 1, 2, 2}
	for i, n := range want {
		if entries[i].Attempt != n {
			t.Errorf("entry %d (%s) belongs to attempt %d, want %d", i, entries[i].Kind, entries[i].Attempt, n)
		}
	}
	// Attempted is what opens a seam, and only the second task.started may.
	for i, e := range entries {
		if e.Attempted() != (i == 1 || i == 4) {
			t.Errorf("entry %d (%s) opens an attempt: %v", i, e.Kind, e.Attempted())
		}
	}
}

// TestAnEntryReadsThePhaseOutOfItsData covers the fields the evidence tab is
// made of. They are read here, once, rather than in the drawing code: a cost
// that will not parse is a zero in a fold and a panic in a render.
func TestAnEntryReadsThePhaseOutOfItsData(t *testing.T) {
	entries := Log([]record.Event{{
		At: at(1), Kind: record.PhaseFinished, Phase: "implement", Text: "wrote retry.go",
		Data: data("n", "1", "engine", "claude", "model", "opus",
			"session", "8f2c31", "cost", "0.42", "error", "none of it"),
	}})

	got := entries[0]
	if got.PhaseN != 1 || got.Engine != "claude" || got.Model != "opus" || got.Session != "8f2c31" {
		t.Errorf("the entry reads %+v, want the phase, engine, model and session out of Data", got)
	}

	if got.Cost != 0.42 {
		t.Errorf("the cost reads %v, want 0.42", got.Cost)
	}

	if got.Cause != "none of it" {
		t.Errorf("the cause reads %q, want what Data[\"error\"] said", got.Cause)
	}

	if got.What() != EntryFinished {
		t.Errorf("a finished phase reads as %v, want EntryFinished", got.What())
	}
}

// TestTruncationIsAnswerableWithoutReadingTheText is what lets the evidence
// tab say "38 of 1,048,583 bytes kept" honestly. Data["output_bytes"] is
// written only when the output was actually cut, so its absence is not a
// missing fact — it is the fact that nothing was lost.
func TestTruncationIsAnswerableWithoutReadingTheText(t *testing.T) {
	cut := Log([]record.Event{{
		At: at(1), Kind: record.PhaseFailed, Phase: "gates",
		Text: "go vet: unreachable code", Data: data("output_bytes", "1048583"),
	}})[0]
	if !cut.Truncated() {
		t.Error("an output the record cut does not say so")
	}

	if cut.Kept != len("go vet: unreachable code") || cut.Full != 1048583 {
		t.Errorf("kept %d of %d, want %d of 1048583", cut.Kept, cut.Full, len("go vet: unreachable code"))
	}

	whole := Log([]record.Event{{
		At: at(1), Kind: record.PhaseFailed, Phase: "gates",
		Text: "go vet: unreachable code",
	}})[0]
	if whole.Truncated() {
		t.Error("an output nothing cut claims to have been cut")
	}

	if whole.Full != 0 {
		t.Errorf("Full = %d on an output nothing cut, want 0", whole.Full)
	}
}

// TestEveryKindTheRecordWritesHasAWord is the guard on the vocabulary. A
// kind the record can write and this fold reads as EntryUnknown would reach
// the task view as its own raw string — "phase.timed_out" in the middle of a
// translated screen — and no test of the window would catch it, because the
// window is drawing exactly what it was handed.
func TestEveryKindTheRecordWritesHasAWord(t *testing.T) {
	kinds := []string{
		record.TaskCreated, record.TaskStarted, record.TaskFinished, record.TaskFailed,
		record.TaskCancelled, record.TaskRequeued, record.TaskTimedOut, record.TaskAbandoned, record.TaskRead,
		record.TaskNoted, record.TaskDialogue,
		record.PhaseStarted, record.PhaseFinished, record.PhaseFailed, record.PhaseCancelled,
		record.PhaseWaiting, record.PhaseResumed,
		// Not written by anything: the reader puts it where a line of the
		// log would not parse. It is on this list because a damaged record
		// is exactly the one a reader opens this screen to look at.
		record.Unreadable,
	}
	for _, kind := range kinds {
		if got := (Entry{Kind: kind}).What(); got == EntryUnknown {
			t.Errorf("the record writes %q and the task view has no word for it", kind)
		}
	}
}

// A dialogue entry says what acted on the task as well as what it did. The
// notes tab draws that name, and an entry that lost it would be drawn as
// having come from nowhere.
func TestADialogueEntryCarriesWhatActed(t *testing.T) {
	got := Log([]record.Event{{
		At:   at(1),
		Kind: record.TaskDialogue,
		Text: "a model cancelled this task over mcp",
		Data: map[string]string{"by": "mcp"},
	}})[0]
	if got.What() != EntryDialogue {
		t.Errorf("a dialogue event reads as %v, want EntryDialogue", got.What())
	}

	if got.By != "mcp" {
		t.Errorf("By = %q, want what the record says acted", got.By)
	}

	if bare := (Entry{Kind: record.TaskDialogue}); bare.By != "" {
		t.Errorf("By = %q on an event that named nothing, want empty and not a guess", bare.By)
	}
}
