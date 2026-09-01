package record

// The kinds an event can have, and the one place they are written down.
//
// Bare literals in internal/task, which writes them, and a second set of
// constants in internal/view, which reads them, cannot see a rename on each
// other: a kind spelled differently by the writer folds to nothing at all in
// the reader, silently, and the record still looks like a record. The layering forbids those two packages from
// importing each other — that absence is what keeps the window from being
// able to append an event — but both may import this one, and the record's
// own vocabulary is exactly what belongs here.
//
// A kind is added, never removed or respelled. The log is append-only and a
// reader meets logs older than itself; a constant deleted here is a line in
// somebody's events.jsonl that stops meaning anything.
const (
	TaskCreated   = "task.created"   // written down; Text is the whole of task.md
	TaskStarted   = "task.started"   // an attempt begins, and the boundary between one and the next
	TaskFinished  = "task.finished"  // every phase of the flow ran through
	TaskFailed    = "task.failed"    // the run stopped and Text says why
	TaskCancelled = "task.cancelled" // a reader stopped it
	// TaskRequeued is a task a reader took back: whatever was running was
	// stopped, and the task goes to the front of the queue rather than into
	// the record as cancelled. The two are different sentences. A run
	// cancelled is a piece of work that is over; a run requeued is one
	// somebody means to start again once they have fixed what was wrong
	// with it, and filing it under Done hides it from the person who has to
	// do that. Data["by"] is who took it back and Text is why, if they said.
	TaskRequeued  = "task.requeued"
	TaskTimedOut  = "task.timedout"  // it outlived the deadline it was given
	TaskAbandoned = "task.abandoned" // its process is gone and a reader wrote that down
	TaskRead      = "task.read"      // somebody has looked at it
	TaskNoted     = "task.noted"     // a user note left for the task
	TaskDialogue  = "task.dialogue"  // something outside a run acted on it; Data["by"] says what
	// TaskDeleted takes a task off every listing without unwriting a word
	// of what it did. The record is the only account of what an engine was
	// asked, what it cost and what it changed, and a reader tidying a board
	// is not saying they want that account gone — they are saying they do
	// not want to look at this row any more. So it is an event like the
	// others rather than a row removed: the fold ignores it, and the one
	// query that enumerates tasks leaves out whatever has it.
	TaskDeleted = "task.deleted"
	// TaskStuck is a task that ran out of attempts. It is not a failure of
	// one run — task.failed already says that — it is the run after the
	// last one the flow was allowed: nothing will move until a reader
	// looks. Data["attempts"] is how many were spent, Text is the line a
	// human reads about why it stopped.
	TaskStuck = "task.stuck"

	PhaseStarted   = "phase.started"   // Data carries engine, model, n, and the permissions the phase was given
	PhaseFinished  = "phase.finished"  // the phase ran through; Text is what the engine printed
	PhaseFailed    = "phase.failed"    // the engine broke; Text is what it printed, Data["error"] why it stopped
	PhaseCancelled = "phase.cancelled" // the phase was stopped from outside; Text is what it printed first
	PhaseWaiting   = "phase.waiting"   // stopped at a gate; Data["why"] says whose gate
	PhaseResumed   = "phase.resumed"   // let go again
	PhaseThought   = "phase.thought"   // a thinking block from the engine stream
	PhaseToolCall  = "phase.tool_call" // a tool call invoked by the engine (Bash, Edit, Read, etc.)
	PhaseRefused   = "phase.refused"   // a tool call the engine was denied by permissions

	GatePassed = "gate.passed" // a phase gate verification check passed
	GateFailed = "gate.failed" // a phase gate verification check failed

	SupervisorMessage    = "supervisor.message"    // a dialogue turn in the global supervisor thread
	SupervisorBriefing   = "supervisor.briefing"   // directive / briefing from the operator
	SupervisorDebriefing = "supervisor.debriefing" // summary / status report from the supervisor
	SupervisorAction     = "supervisor.action"     // autonomous action taken by the supervisor
	// SupervisorRetracted takes back an earlier line of the thread, naming
	// it in data.at with record.Stamp. It is how somebody unsays something
	// in a log that cannot erase: the withdrawn line stays where it is, and
	// stops being repeated into the model's prompt.
	SupervisorRetracted = "supervisor.retracted"

	// A decision is what somebody chose and why, written down where the
	// work happened rather than in a document beside it. The event is the
	// decision's home; a file under .orbit/decisions/ is a copy of it, and
	// the other way around there would be two truths.
	//
	// Data["id"] names it so a later line can point back at it, Data["scope"]
	// lists the paths it governs — that is what makes a decision checkable
	// against a diff rather than prose nobody reads — and Text is the
	// decision itself.
	DecisionMade = "decision.made"
	// DecisionSuperseded replaces an earlier decision with the one this
	// event carries. It names the earlier one in Data["at"] by record.Stamp,
	// the way supervisor.retracted names the line it takes back: the log
	// cannot erase, so what changes is what a decision still governs, never
	// whether it was made.
	DecisionSuperseded = "decision.superseded"

	// RepoJoined is a repository joining the task by being worked in. The
	// scope of a task is not declared and then checked — it is observed:
	// opening a worktree is what joining is, in whichever phase it happens.
	// Data["repo"] is the repository's name and Data["path"] is where its
	// worktree was made.
	RepoJoined = "repo.joined"
)
