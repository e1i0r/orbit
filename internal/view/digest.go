package view

// What a week of runs adds up to, as a projection of the record and nothing
// else.

import (
	"sort"

	"github.com/e1i0r/orbit/internal/record"
)

// Digest is what the record says about a body of work, in the figures worth
// acting on.
//
// Not the ones that read well. The number of tasks, the most-used model and
// the average time a run takes are all easy to compute and none of them
// changes anything a reader would do — where "the gate people keep having to
// answer" names a phase of a flow that is designed wrong, and "what the
// stuck work cost" is a bill for a thing that produced nothing.
type Digest struct {
	// Merged and Finished are work that landed and work that ran through.
	// They are apart because a task that finished and was never merged is
	// work nobody has decided about, and counting it as delivered is how a
	// report comes to disagree with the repository.
	Merged   int
	Finished int
	// Untouched is the runs that went from start to finish with nobody
	// being asked anything: no gate stopped for a person, no note was left
	// mid-run. It is the one number that says whether any of this is
	// working.
	Untouched int
	// Stuck is work that ran out of attempts, budget or patience, and Spent
	// against it is what that cost. A bill for what produced nothing is the
	// figure that pays for reading a report at all.
	Stuck      int
	SpentStuck float64
	// Spent is what everything cost, and SpentMerged the part of it that
	// reached a branch somebody merged.
	Spent       float64
	SpentMerged float64
	// AskedAt counts the phases a person was stopped at, by phase name.
	// The phase at the top of this list is the phase of the flow that is
	// designed wrong.
	AskedAt map[string]int
	// Retried counts the phases a gate sent round again, by phase name, and
	// Requeued the whole tasks somebody sent back.
	Retried  map[string]int
	Requeued int
}

// Asked is the phases people were stopped at, most first.
//
// A method rather than a sorted field, because a map is what it is counted
// in and an order is a question about how it is read.
func (d Digest) Asked() []Count { return ranked(d.AskedAt) }

// Rounds is the phases that went round again, most first.
func (d Digest) Rounds() []Count { return ranked(d.Retried) }

// Count is one name and how often it happened.
type Count struct {
	Name string
	N    int
}

// ranked is a count map as a list, biggest first and ties by name so that
// two runs of the same digest read the same.
func ranked(m map[string]int) []Count {
	out := make([]Count, 0, len(m))
	for name, n := range m {
		out = append(out, Count{Name: name, N: n})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}

		return out[i].Name < out[j].Name
	})

	return out
}

// Digested folds one task's record into a digest, and adds it to whatever is
// there already.
//
// One task at a time, because that is how the record is stored and how a
// caller with ten thousand tasks has to read them. The zero Digest is a
// digest of nothing, so a caller starts with one and hands it back.
func Digested(d Digest, events []record.Event) Digest {
	if d.AskedAt == nil {
		d.AskedAt, d.Retried = map[string]int{}, map[string]int{}
	}

	var (
		spent    float64
		asked    bool
		merged   bool
		finished bool
		stuck    bool
	)

	for _, e := range events {
		switch e.Kind {
		case record.PhaseFinished, record.PhaseFailed, record.PhaseCancelled, record.PhaseRetried:
			spent += money(e.Data["cost"])
		}

		switch e.Kind {
		case record.PhaseWaiting:
			d.AskedAt[e.Phase]++
			asked = true
		case record.TaskNoted, record.TaskDialogue:
			// A note left before the first run is the brief, not an
			// interruption: what this counts is somebody having to step in
			// while the work was going.
			asked = asked || finished || stuck
		case record.PhaseRetried:
			d.Retried[e.Phase]++
		case record.TaskRequeued:
			d.Requeued++
		case record.TaskStuck, record.TaskOverBudget, record.TaskOverDiff,
			record.TaskNewDependency, record.TaskContradicts:
			stuck = true
		case record.TaskFinished:
			finished, stuck = true, false
		case record.TaskMerged:
			merged = true
		}
	}

	d.Spent += spent

	switch {
	case merged:
		d.Merged++
		d.SpentMerged += spent
	case stuck:
		d.Stuck++
		d.SpentStuck += spent
	}

	if finished {
		d.Finished++

		if !asked {
			d.Untouched++
		}
	}

	return d
}
