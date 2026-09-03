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
	// ReviewComment is something a reviewer wrote on the pull request,
	// read back into the record so a phase can answer it. Data["by"] is who
	// wrote it, Data["where"] the file and line it is about, and
	// Data["url"] where it can be read in full. It is consumed the way a
	// note is: the phase that runs next is told, and the one after it is
	// not told again.
	ReviewComment = "review.comment"

	// TaskCritical marks a task as one that reaches something that
	// matters, and every critical.* event below happens only for a task
	// that carries it. Data["on"] is whether it is being turned on or off
	// and Data["by"] is who said so — a mark that can be lifted is a mark
	// somebody can put on by mistake without being stuck with it.
	TaskCritical = "task.critical"

	// The five steps of the critical protocol, in the order they happen.
	//
	// critical.snapshot is how the world stood before, critical.backup is
	// the ref that can put it back, critical.approved or .rejected is what
	// a person said about the plan, and critical.applied is what was done
	// and where it left things. Data["revert"] travels with the last three
	// because the command that undoes it is what a reader is owed at every
	// point after the question was asked.
	CriticalSnapshot = "critical.snapshot"
	CriticalBackup   = "critical.backup"
	CriticalApproved = "critical.approved"
	CriticalRejected = "critical.rejected"
	CriticalApplied  = "critical.applied"
	TaskDialogue     = "task.dialogue" // something outside a run acted on it; Data["by"] says what
	// TaskDeleted takes a task off every listing without unwriting a word
	// of what it did. The record is the only account of what an engine was
	// asked, what it cost and what it changed, and a reader tidying a board
	// is not saying they want that account gone — they are saying they do
	// not want to look at this row any more. So it is an event like the
	// others rather than a row removed: the fold ignores it, and the one
	// query that enumerates tasks leaves out whatever has it.
	TaskDeleted = "task.deleted"
	// TaskMerged is work that landed: somebody merged the pull request a
	// task opened. It is written where the merge happens rather than
	// inferred from a branch that disappeared, because a branch can vanish
	// for three other reasons and only one of them is delivery.
	TaskMerged = "task.merged"
	// TaskStuck is a task that ran out of attempts. It is not a failure of
	// one run — task.failed already says that — it is the run after the
	// last one the flow was allowed: nothing will move until a reader
	// looks. Data["attempts"] is how many were spent, Text is the line a
	// human reads about why it stopped.
	TaskStuck = "task.stuck"

	// TaskOverBudget is a task that has spent what it was allowed. Like
	// task.stuck it is the end of a run that nothing will move on its own,
	// and unlike it nothing was wrong with the work: the run stopped
	// because of a number somebody chose. Data["spent"] and
	// Data["budget"] are the two figures, and Text is the line a human
	// reads about which phase did not run.
	TaskOverBudget = "task.over_budget"

	// TaskOverDiff is a task whose change grew past what its flow allowed,
	// or reached a file the plan never named. Like task.over_budget it is
	// a run stopped by a number somebody chose rather than by anything
	// going wrong. Data["lines"], Data["budget"] and Data["unplanned"] are
	// what a reader compares.
	TaskOverDiff = "task.over_diff"

	// TaskNewDependency is a task that added a library nobody has approved.
	// It is a run stopped by a decision that is not the agent's to make:
	// what a project carries — its licences, its maintenance, its security
	// updates — is the reader's. Data["names"] is what was added.
	TaskNewDependency = "task.new_dependency"

	// TaskContradicts is a change that went against a decision this task
	// had already made. Data["decision"] names it and Text is why the
	// judge said so — the two things a reader needs to choose between the
	// only two answers there are: change the code back, or supersede the
	// decision.
	TaskContradicts = "task.contradicts"

	// TaskStory is how this prompt became this diff, in the five fields the
	// task story spec settles on: entry, purpose, symptom, cause, fix. The
	// engine writes them and the record is what proves them — every claim
	// sits beside the events that would refute it.
	TaskStory = "task.story"

	PhaseStarted = "phase.started" // Data carries engine, model, n, and the permissions the phase was given
	// PhaseFinished ends a phase that ran through. Text is what the engine
	// printed, and Data carries what it spent doing so: cost where the
	// engine prices itself, and tokens_in, tokens_out, cache_read and
	// cache_write where it counts. Failed and cancelled phases carry the
	// same fields — a phase that broke halfway still spent what it spent.
	PhaseFinished  = "phase.finished"
	PhaseFailed    = "phase.failed"    // the engine broke; Text is what it printed, Data["error"] why it stopped
	PhaseCancelled = "phase.cancelled" // the phase was stopped from outside; Text is what it printed first
	PhaseWaiting   = "phase.waiting"   // stopped at a gate; Data["why"] says whose gate
	PhaseResumed   = "phase.resumed"   // let go again
	// PhaseRetried is the seam between one attempt at a phase and the next:
	// the gate refused the work and the flow allows another run of the same
	// phase. Data["gate"] is the gate that refused, Data["exit"] what it
	// returned, Data["attempt"] which attempt has just ended and
	// Data["attempts"] how many the flow allows, so a reader can see how
	// much rope is left without counting the events themselves.
	//
	// It is written between the two attempts rather than at the end of the
	// phase because a reader watching a run needs to know it is going round
	// again while it is going round, not once it stops.
	PhaseRetried  = "phase.retried"
	PhaseThought  = "phase.thought"   // a thinking block from the engine stream
	PhaseToolCall = "phase.tool_call" // a tool call invoked by the engine (Bash, Edit, Read, etc.)
	PhaseRefused  = "phase.refused"   // a tool call the engine was denied by permissions

	// LoopChecked is one turn of a loop and what its checks answered.
	// Data["turn"] and Data["turns"] are where it is of what it was
	// allowed, Data["passed"] is whether the loop can stop, and on a turn
	// that did not pass Data["check"] names the command and Text is what
	// it printed — which is what the next turn is told.
	LoopChecked = "loop.checked"

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

	// DependencyApproved is a reader saying yes to the libraries a task
	// added. Data["names"] is what they were shown and accepted, so the
	// gate can let exactly those past and stop for anything else — per
	// name and not per run, because the same library added again by a
	// later phase is a question that was already answered.
	DependencyApproved = "dependency.approved"

	// RepoJoined is a repository joining the task by being worked in. The
	// scope of a task is not declared and then checked — it is observed:
	// opening a worktree is what joining is, in whichever phase it happens.
	// Data["repo"] is the repository's name and Data["path"] is where its
	// worktree was made.
	RepoJoined = "repo.joined"
)
