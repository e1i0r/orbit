package view

// How many runs in a row have ended stuck, which is the one question the
// circuit breaker asks of the board.

import (
	"slices"
)

// StuckStreak is the runs that have most recently stopped and ended stuck,
// newest first, counting back until one of them did not.
//
// It counts endings and not tasks. A task nobody has run, and a run still
// going or stopped at a gate, have not ended: they neither add to the run
// nor break it. The question is what the last few runs to stop did, and a
// task sitting in To Do says nothing about that either way.
//
// Anything that ended and was not stuck breaks the count, a cancellation
// included. Three stuck tasks with a cancelled one among them is not a board
// where nothing is working; it is a board where somebody was there.
//
// It reads the fields that leave this package rather than the private state
// the fold keeps, so that the window drawing a board can be asked the same
// question about the same values — a breaker that could only be tested
// through the fold is a breaker no test of the window can reach.
func StuckStreak(tasks []Task) []Task {
	ended := make([]Task, 0, len(tasks))

	for _, t := range tasks {
		if Stopped(t) {
			ended = append(ended, t)
		}
	}

	// Newest first, because the run being counted is the recent one: the
	// board hands its tasks in its own order, which is the order they are
	// drawn in and not the order they stopped in.
	slices.SortStableFunc(ended, func(a, b Task) int { return b.Since.Compare(a.Since) })

	for i, t := range ended {
		if t.Reason.Key != ReasonStuck {
			return ended[:i]
		}
	}

	return ended
}

// Stopped reports whether a run of this task has ended.
//
// A reason that says how a run stopped is the whole answer for five of the
// six ways; the sixth is a run that finished, which clears its reason and
// leaves Done to say so. Gate and Held are stops inside a run that has not
// ended — something is still holding a worktree — and a task with no reason
// outside Done has either not run or is running now.
func Stopped(t Task) bool {
	switch t.Reason.Key {
	case ReasonFailed, ReasonFailedToStart, ReasonTimedOut, ReasonAbandoned, ReasonCancelled,
		ReasonStuck, ReasonOverBudget, ReasonOverDiff:
		return true
	case "":
		return BandOf(t) == Done
	default:
		return false
	}
}
