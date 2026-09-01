package view

// One task's record, folded into the lines a reader is shown one level below
// the board.
//
// Fold answers "what is this task doing", which is a single row; this
// answers "what has this task done", which is the whole log. They are two
// folds over the same events on purpose: the board needs one Task per log
// and never the events, and the task view needs every event and never the
// Task. Sharing one type would mean handing the window a []record.Event,
// which is the import internal/ui is not allowed — and the reason it is not
// allowed is that a window able to read the record's own shape is a window
// one commit away from writing it.
//
// Everything the task view draws is a field here. That is deliberate down to
// the attempt number: the seam the log draws between one attempt and the
// next is a property of the entry it lands on, and not a string match on a
// kind inside a drawing function. A view that recomputes structure while it
// is painting is a view that disagrees with itself the day a kind is added.

import (
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// Entry is one event of one task's record, with the fields the task view
// draws already read out of Data.
//
// The fields are flat rather than a map because internal/ui measures in
// cells and formats numbers: a map of strings would put parsing in the
// drawing code, where a malformed cost is a panic in a render rather than a
// zero in a fold. Every read here is the same lenient one the board's fold
// uses — a field the record does not have is the zero value, and never a
// guess.
type Entry struct {
	At    time.Time // when it was written; zero when the record's clock was damaged
	Kind  string    // the record's own kind, as record's constants spell it
	Phase string    // the phase it belongs to, empty for the task-level kinds
	Text  string    // what was written: the task for task.created, the engine's output for a phase

	// Attempt is which run of the task this entry belongs to, counting from
	// one at the first task.started. An entry written before any attempt —
	// task.created, and a task.read on a task nobody has run — is 0, which
	// is how the task view knows not to draw a seam above it.
	Attempt int

	PhaseN  int     // the phase's 1-based place in the flow, 0 when the record does not say
	Engine  string  // which engine ran it
	Model   string  // which model it ran
	Session string  // the engine's session id, for resuming by hand
	Cause   string  // why a phase stopped, from Data["error"]
	Cost    float64 // what the phase cost
	Gate    string  // gate name, from Data["gate"]
	Exit    string  // exit code, from Data["exit"]
	Tool    string  // tool name, from Data["tool"]
	Notes   string  // note count or info, from Data["notes"]
	By      string  // what acted on the task from outside a run, from Data["by"]

	// Kept and Full are the size of the engine's output as it was written
	// and as it actually was. They differ only when internal/task cut it,
	// and Full is 0 whenever it did not — so Full > Kept is the whole of the
	// question "was anything lost", asked without parsing the marker at the
	// end of Text.
	Kept int
	Full int
}

// Truncated says the engine printed more than the record kept.
func (e Entry) Truncated() bool { return e.Full > e.Kept }

// Said is Text with an engine's stream framing left out: the words the model
// wrote, and not the hook traffic, token counters and tool calls a killed
// phase leaves on stdout around them. It is Text itself whenever Text was
// never a stream, which is everything a run that reached its own end writes.
//
// It is derived rather than stored so that an entry cannot be built holding
// one and not the other, the same reason Truncated is not a bool somebody
// has to remember to set.
func (e Entry) Said() string { return unframe(e.Text) }

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
	EntryUnknown    EntryKind = iota // a kind this build does not know; draw Kind itself
	EntryWritten                     // the task was written down
	EntryStarted                     // a run or a phase began
	EntryFinished                    // it ran through
	EntryFailed                      // it broke
	EntryCancelled                   // somebody stopped it
	EntryRequeued                    // somebody took it back to the queue
	EntryTimedOut                    // it outlived its deadline
	EntryAbandoned                   // its process is gone
	EntryRead                        // somebody has looked at it
	EntryWaiting                     // it stopped at a gate
	EntryResumed                     // it was let go again
	EntryGatePassed                  // verification gate passed
	EntryGateFailed                  // verification gate failed
	EntryThought                     // thinking block
	EntryToolCall                    // tool call invocation
	EntryRefused                     // permission refused
	EntryNoted                       // user note
	EntryDialogue                    // something outside a run acted on the task
	EntryStuck                       // the attempts ran out
	EntryDecision                    // something was decided, and the decision is in the line
	EntrySuperseded                  // a decision replaced an earlier one
	EntryRepoJoined                  // a repository joined the task by being worked in
	EntryUnreadable                  // this line of the record itself is damaged
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

// Attempted says this entry opens a new attempt, which is where the task
// view draws the seam between one run and the next.
func (e Entry) Attempted() bool { return e.Kind == record.TaskStarted }

// Log folds a whole record into its entries, oldest first — the order they
// were appended in, which is the order the record is true in.
//
// It is total, like Fold: an unknown kind is still an entry, because a
// reader looking at a log to find out what happened is exactly the reader a
// dropped line hurts. A kind this build does not know is a kind a newer
// build wrote, and showing it unstyled is a better answer than showing
// nothing where it was.
func Log(events []record.Event) []Entry {
	out := make([]Entry, 0, len(events))
	attempt := 0

	for _, e := range events {
		if e.Kind == record.TaskStarted {
			attempt++
		}

		out = append(out, entry(e, attempt))
	}

	return out
}

// entry reads one event's fields. Indexing a nil Data map is a zero value
// rather than a panic, which is why no event needs a guard of its own.
func entry(e record.Event, attempt int) Entry {
	return Entry{
		At:      e.At,
		Kind:    e.Kind,
		Phase:   e.Phase,
		Text:    e.Text,
		Attempt: attempt,
		PhaseN:  count(e.Data["n"]),
		Engine:  e.Data["engine"],
		Model:   e.Data["model"],
		Session: e.Data["session"],
		Cause:   e.Data["error"],
		Cost:    money(e.Data["cost"]),
		Gate:    e.Data["gate"],
		Exit:    e.Data["exit"],
		Tool:    e.Data["tool"],
		Notes:   e.Data["notes"],
		By:      e.Data["by"],
		Kept:    len(e.Text),
		Full:    count(e.Data["output_bytes"]),
	}
}
