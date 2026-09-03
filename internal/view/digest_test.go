package view

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// ran is one task's record: a run that started, did what the kinds say, and
// ended how the last of them says.
func ran(kinds ...record.Event) []record.Event {
	return append([]record.Event{{Kind: record.TaskCreated, Text: "a task"}, {Kind: record.TaskStarted}}, kinds...)
}

func cost(kind, phase, usd string) record.Event {
	return record.Event{Kind: kind, Phase: phase, Data: map[string]string{"cost": usd}}
}

// TestADigestCountsWhatLandedApartFromWhatMerelyRan. A task that finished
// and was never merged is work nobody has decided about, and counting it as
// delivered is how a report comes to disagree with the repository.
func TestADigestCountsWhatLandedApartFromWhatMerelyRan(t *testing.T) {
	d := Digested(Digest{}, ran(
		cost(record.PhaseFinished, "implement", "0.25"),
		record.Event{Kind: record.TaskFinished},
		record.Event{Kind: record.TaskMerged},
	))
	d = Digested(d, ran(
		cost(record.PhaseFinished, "implement", "0.25"),
		record.Event{Kind: record.TaskFinished},
	))

	if d.Merged != 1 || d.Finished != 2 {
		t.Errorf("the digest counts %d merged and %d finished, want 1 and 2", d.Merged, d.Finished)
	}

	if d.Spent != 0.5 || d.SpentMerged != 0.25 {
		t.Errorf("the digest counts %v spent and %v of it merged, want 0.5 and 0.25", d.Spent, d.SpentMerged)
	}
}

// TestADigestKnowsWhatTheStuckWorkCost. A bill for what produced nothing is
// the figure that pays for reading a report at all.
func TestADigestKnowsWhatTheStuckWorkCost(t *testing.T) {
	d := Digested(Digest{}, ran(
		cost(record.PhaseRetried, "implement", "0.10"),
		cost(record.PhaseFailed, "implement", "0.15"),
		record.Event{Kind: record.TaskStuck},
	))

	if d.Stuck != 1 || d.SpentStuck != 0.25 {
		t.Errorf("the digest counts %d stuck at %v, want 1 at 0.25", d.Stuck, d.SpentStuck)
	}

	if d.Finished != 0 {
		t.Errorf("a stuck task was counted as finished")
	}
}

// TestUntouchedIsARunNobodyHadToStepInto. It is the one number that says
// whether any of this is working.
func TestUntouchedIsARunNobodyHadToStepInto(t *testing.T) {
	clean := Digested(Digest{}, ran(
		cost(record.PhaseFinished, "implement", "0.25"),
		record.Event{Kind: record.TaskFinished},
	))
	if clean.Untouched != 1 {
		t.Errorf("a run nobody was asked about counts %d untouched, want 1", clean.Untouched)
	}

	asked := Digested(Digest{}, ran(
		record.Event{Kind: record.PhaseWaiting, Phase: "review"},
		record.Event{Kind: record.PhaseResumed, Phase: "review"},
		record.Event{Kind: record.TaskFinished},
	))
	if asked.Untouched != 0 {
		t.Errorf("a run that stopped for a person counts as untouched")
	}

	if got := asked.Asked(); len(got) != 1 || got[0].Name != "review" {
		t.Errorf("the digest says people were stopped at %v, want the review phase", got)
	}
}

// TestTheBriefIsNotAnInterruption. A note left before the first run is what
// the task was written with, and counting it would report every task as one
// somebody had to step into.
func TestTheBriefIsNotAnInterruption(t *testing.T) {
	d := Digested(Digest{}, []record.Event{
		{Kind: record.TaskCreated, Text: "a task"},
		{Kind: record.TaskNoted, Text: "and mind the timezone"},
		{Kind: record.TaskStarted},
		cost(record.PhaseFinished, "implement", "0.25"),
		{Kind: record.TaskFinished},
	})

	if d.Untouched != 1 {
		t.Errorf("a note written before the run counted as somebody stepping in")
	}
}

// TestTheRoundsAreRankedByHowOftenTheyHappen. The phase at the top of the
// list is the phase of the flow that is designed wrong.
func TestTheRoundsAreRankedByHowOftenTheyHappen(t *testing.T) {
	d := Digested(Digest{}, ran(
		record.Event{Kind: record.PhaseRetried, Phase: "gates"},
		record.Event{Kind: record.PhaseRetried, Phase: "gates"},
		record.Event{Kind: record.PhaseRetried, Phase: "implement"},
		record.Event{Kind: record.TaskRequeued},
		record.Event{Kind: record.TaskFinished},
	))

	rounds := d.Rounds()
	if len(rounds) != 2 || rounds[0].Name != "gates" || rounds[0].N != 2 {
		t.Errorf("the rounds read %v, want gates twice at the top", rounds)
	}

	if d.Requeued != 1 {
		t.Errorf("the digest counts %d requeues, want 1", d.Requeued)
	}
}
