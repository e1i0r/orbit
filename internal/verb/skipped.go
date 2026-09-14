package verb

// Getting past a rule that is in the way, without deciding anything about it.
//
// Two moments that used to be one. You are in the middle of something else
// and a rule stops you: what you need is to pass, cheaply and reversibly —
// skip it this once, or pause it and say why. Nothing more. Switching a rule
// off or rewording it decides its fate, and that is not a decision taken in a
// hurry with a task half done.
//
// Either one sends it to be looked at again, the first time, without counting
// to three: nobody skips a rule they agree with. If you skipped it, you have
// already said something without saying it.

import (
	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/task"
)

// gotPast writes down that a rule was skipped, and sends it to be looked at.
//
// It answers nothing and cannot stop the skip. Getting past the thing in your
// way is what you asked for; noticing that you have now skipped a rule twice
// is a question asked about it, and a question that could not be asked is not
// a reason to leave you stuck.
func gotPast(w World, t task.Task) {
	rule, phase := inTheWay(w, t)
	if rule == "" {
		return
	}

	_ = learn.Happened(w.Store(), learn.Turn{ //nolint:errcheck // the skip stands either way
		Rule: rule, What: learn.Skipped, By: learn.Operator, Task: t.ID, Phase: phase,
	})

	sendToReview(w, rule)
}

// inTheWay is the rule whose gate the run is stopped in front of, and the
// phase it stopped in.
//
// The last refusal in the record and not a search through them. A run waiting
// is waiting at the gate that refused it last; one that got past that gate is
// not waiting at all, so there is nothing to skip.
//
// A phase stopped by something that is not a rule — a gate the flow declares,
// a phase marked to wait — names none, and nothing is written down.
func inTheWay(w World, t task.Task) (rule, phase string) {
	events, err := task.Events(w.Store(), t)
	if err != nil {
		return "", ""
	}

	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Kind == record.GateFailed {
			return events[i].Data["rule"], events[i].Phase
		}
	}

	return "", ""
}

// sendToReview marks the rule as one waiting for a decision, and leaves it
// applying.
//
// Applying, because skipping is not disagreeing: the rule stands and the next
// run is still told it. What changed is that somebody now has to look at it.
func sendToReview(w World, rule string) {
	facts, err := w.Facts()
	if err != nil {
		return
	}

	for _, was := range facts {
		if was.ID != rule || was.Review {
			continue
		}

		now := was
		now.Review = true

		_ = w.Replace(was, now) //nolint:errcheck // the skip stands either way

		return
	}
}
