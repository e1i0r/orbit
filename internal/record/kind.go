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
	TaskTimedOut  = "task.timedout"  // it outlived the deadline it was given
	TaskAbandoned = "task.abandoned" // its process is gone and a reader wrote that down
	TaskRead      = "task.read"      // somebody has looked at it
	TaskNoted     = "task.noted"     // a user note left for the task
	TaskDialogue  = "task.dialogue"  // something outside a run acted on it; Data["by"] says what

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
)
