package view

// The vocabulary the window reads a record in, and the reading itself.
//
// It sits apart from the fold that builds the entries because it is a
// different question: log.go is about getting the record's lines into a
// struct, and this is about what one of those lines means to somebody
// looking at it.

import "github.com/e1i0r/orbit/internal/record"

// EntryKind is what one entry says happened, in this package's vocabulary
// rather than the record's own.
//
// It exists for the same reason Reason.Key does: internal/ui draws these and
// may not import internal/record, so a window that switched on the kind
// would have to spell "phase.finished" itself — a second copy of the
// record's vocabulary, in the one package that is forbidden from seeing the
// first. Here the mapping is made once, beside the fold that reads the
// record, and the window switches on a value it is allowed to name.
//
// The two levels share their words. A task and a phase both start, finish
// and fail, and which of the two an entry is about is Phase being set — so
// the window needs one switch of twelve cases rather than two of seven, and
// a translator writes "finished" once.
type EntryKind int

// The kinds an entry can be read as. Their trailing comments are the whole
// of what each one means; a doc comment per constant would say the same
// things twice.
const (
	EntryUnknown       EntryKind = iota // a kind this build does not know; draw Kind itself
	EntryWritten                        // the task was written down
	EntryStarted                        // a run or a phase began
	EntryFinished                       // it ran through
	EntryFailed                         // it broke
	EntryCancelled                      // somebody stopped it
	EntryRequeued                       // somebody took it back to the queue
	EntryTimedOut                       // it outlived its deadline
	EntryAbandoned                      // its process is gone
	EntryRead                           // somebody has looked at it
	EntryWaiting                        // it stopped at a gate
	EntryResumed                        // it was let go again
	EntryRetried                        // its gate refused it and the phase is being run again
	EntryGatePassed                     // verification gate passed
	EntryGateFailed                     // verification gate failed
	EntryThought                        // thinking block
	EntryToolCall                       // tool call invocation
	EntryRefused                        // permission refused
	EntryNoted                          // user note
	EntryDialogue                       // something outside a run acted on the task
	EntryStuck                          // the attempts ran out
	EntryOverBudget                     // it spent what it was allowed
	EntryOverDiff                       // it changed more than was agreed
	EntryNewDependency                  // it reached for a library nobody approved
	EntryContradicts                    // the change goes against a decision
	EntryLoopChecked                    // a turn of a loop, and what its checks answered
	EntryApproved                       // a reader said yes to those libraries
	EntryDecision                       // something was decided, and the decision is in the line
	EntrySuperseded                     // a decision replaced an earlier one
	EntryRepoJoined                     // a repository joined the task by being worked in
	EntryUnreadable                     // this line of the record itself is damaged
)

// What is the entry's kind in this package's vocabulary.
//
// It is a method rather than a field because the record's kind is the fact
// and this is a reading of it: a log written by a newer build reaches an
// older one with a kind it has never heard of, and answering EntryUnknown
// while keeping Kind intact is what lets the task view draw the line anyway.
// A dropped line is the worst possible answer for a reader who opened this
// screen to find out what happened.
func (e Entry) What() EntryKind {
	switch e.Kind {
	case record.TaskCreated:
		return EntryWritten
	case record.TaskStarted, record.PhaseStarted:
		return EntryStarted
	case record.TaskFinished, record.PhaseFinished:
		return EntryFinished
	case record.TaskFailed, record.PhaseFailed:
		return EntryFailed
	case record.TaskCancelled, record.PhaseCancelled:
		return EntryCancelled
	case record.TaskRequeued:
		return EntryRequeued
	case record.TaskTimedOut:
		return EntryTimedOut
	case record.TaskAbandoned:
		return EntryAbandoned
	case record.TaskRead:
		return EntryRead
	case record.PhaseWaiting:
		return EntryWaiting
	case record.PhaseResumed:
		return EntryResumed
	case record.PhaseRetried:
		return EntryRetried
	case record.GatePassed:
		return EntryGatePassed
	case record.GateFailed:
		return EntryGateFailed
	case record.PhaseThought:
		return EntryThought
	case record.PhaseToolCall:
		return EntryToolCall
	case record.PhaseRefused:
		return EntryRefused
	case record.TaskNoted:
		return EntryNoted
	case record.TaskDialogue:
		return EntryDialogue
	case record.TaskStuck:
		return EntryStuck
	case record.TaskOverBudget:
		return EntryOverBudget
	case record.TaskOverDiff:
		return EntryOverDiff
	case record.TaskNewDependency:
		return EntryNewDependency
	case record.TaskContradicts:
		return EntryContradicts
	case record.LoopChecked:
		return EntryLoopChecked
	case record.DependencyApproved:
		return EntryApproved
	case record.DecisionMade:
		return EntryDecision
	case record.DecisionSuperseded:
		return EntrySuperseded
	case record.RepoJoined:
		return EntryRepoJoined
	case record.Unreadable:
		// Not a kind anything wrote: the reader synthesises it where a line
		// would not parse, and it is a fact about the log rather than about
		// the task. It is named here so the task view can say so in the
		// reader's own language, instead of printing a key from a file.
		return EntryUnreadable
	}

	return EntryUnknown
}
