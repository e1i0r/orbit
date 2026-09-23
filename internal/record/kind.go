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
	TaskQueued    = "task.queued"    // waiting for a slot; Data: repo, flow, engine, from
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
	// not want to look at this row any more. So it is an event, which the
	// fold ignores and the one query that enumerates tasks leaves out.
	TaskDeleted = "task.deleted"
	// TaskMerged is work that landed: somebody merged the pull request a
	// task opened. It is written where the merge happens rather than
	// inferred from a branch that disappeared, because a branch can vanish
	// for three other reasons and only one of them is delivery.
	TaskMerged = "task.merged"
	// DeliverAsked is a delivery verb the operator pressed in the cockpit:
	// open the pull request, bring it up to date, make its checks pass,
	// answer its reviews. Data["verb"] is which one, in the caption the key
	// was offered under, Data["by"] what was handed the work (the supervisor
	// or a command), and Data["pid"] the window it was asked in, which carries it.
	//
	// It is written where the key is pressed rather than by whatever does
	// the work, because most of these verbs are carried out by an engine
	// that answers minutes later somewhere else entirely. A reader who saw
	// nothing appear could not tell a slow verb from a dead one.
	DeliverAsked = "deliver.asked"
	// DeliverAnswered ends one of those: Data["verb"], Data["error"] for why it
	// broke, and Text for what came back.
	DeliverAnswered = "deliver.answered"
	// DeliverStep is one step its carrier took: Data["verb"], "tool", and Text.
	DeliverStep = "deliver.step"

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

	// TaskRelayed is the engine changing hands in the middle of a task.
	//
	// Data["from"] is the engine that had it, Data["to"] the one that took
	// it, Data["phase"] where it changed, and Data["why"] whether the first
	// one ran out or a person chose the second. Text is the line a human
	// reads.
	//
	// Written down because a task that passed through three engines is a
	// task whose result cannot be judged without knowing it: code that came
	// out strange is either the flow's doing or the third engine arriving
	// with half the context, and nothing else in the record tells those
	// apart. It is also the only place the real cost shows — three
	// allowances spent, one task.
	TaskRelayed = "task.relayed"

	// TaskNeedsEngine is a run that stopped because its engine ran out and
	// nobody had said to carry on without asking.
	//
	// Data["engines"] names the ones that could take it, Data["from"] the
	// one that ran out and Data["phase"] where it stopped. It is a run
	// waiting on a person and not a run that failed: nothing is broken, and
	// what it needs is a choice.
	TaskNeedsEngine = "task.needs_engine"

	// TaskNoEngine is a run that stopped because no engine has anything
	// left to spend.
	//
	// Data["back"] is how long until the first allowance comes back, in the
	// form time.Duration prints, and Data["from"] and Data["phase"] are
	// where it stopped. The duration is the whole point of the kind:
	// allowances come back, so "no engines until 16:30" is an answer and
	// "abandoned" is not.
	TaskNoEngine = "task.no_engine"

	// TaskStory is how this prompt became this diff, in the five fields the
	// task story spec settles on: entry, purpose, symptom, cause, fix. The
	// engine writes them and the record is what proves them — every claim
	// sits beside the events that would refute it.
	TaskStory = "task.story"

	// TaskDelta is what the change asks of its callers and what it promises
	// them, as the engine that wrote it says: the preconditions it added,
	// the guarantees it now holds, what it assumed about the world around
	// it, and the alternatives it discarded with the reason.
	//
	// It is the engine's claim about its own work and nothing verified it.
	// That is not a flaw to be fixed by checking it — nothing can check
	// "assumes UTC timestamps" — it is what the field is, and every reader
	// of it says so. What makes it worth keeping is that the discarded
	// alternatives exist nowhere else: the moment the run ends they are
	// gone, and the next person to touch that code pays to rediscover them.
	TaskDelta = "task.delta"

	PhaseStarted = "phase.started" // Data carries engine, model, n, and the permissions the phase was given
	// PhaseAsked is the prompt a phase was given, as the engine received it.
	//
	// The one thing Orbit sends and the only one that used to be invisible.
	// Everything else about a run can be read back — the diff, the turns,
	// the tool calls, what the engine answered — so a phase that came out
	// strange could be examined from every side except the one that caused
	// it. What goes into a prompt is not small either: the task, the phase,
	// the rules in force, what a person said, what the reviewers asked,
	// what the phase before answered, what the gates refused, and what the
	// attempt before got as far as doing.
	//
	// Text is the prompt and Data["engine"] is who was given it, which
	// after a relay is not the engine the flow named. Data["bytes"] is how
	// long it really was when the record could not keep all of it.
	//
	// One event per attempt, written before the engine is called — so a
	// phase that never answers still says what it was asked.
	PhaseAsked = "phase.asked"
	// PhaseFinished ends a phase that ran through. Text is what the engine
	// printed, and Data carries what it spent doing so: cost where the
	// engine prices itself, and tokens_in, tokens_out, cache_read and
	// cache_write where it counts. Failed and cancelled phases carry the
	// same fields — a phase that broke halfway still spent what it spent.
	PhaseFinished = "phase.finished"
	// PhaseFailed ends a phase whose engine broke. Text is what it printed
	// and Data["error"] why it stopped.
	PhaseFailed = "phase.failed"
	// PhaseRanOut ends a phase whose engine had nothing left to spend.
	//
	// Apart from PhaseFailed because they send a reader to do opposite
	// things. A phase that broke is a bug to go and look at; a phase that
	// ran out is a wait, or the same work handed to another engine — and a
	// task that said only "it broke" made somebody open the log to find out
	// which of the two it had been.
	//
	// Text is what the engine printed and Data["error"] why it stopped, the
	// same as a failure: what changes is the name, because the name is what
	// a reader acts on.
	PhaseRanOut = "phase.ran_out"
	// PhaseDenied ends a phase that was refused what it needed and left
	// nothing behind.
	//
	// Apart from PhaseFinished because it is the opposite of one, and apart
	// from PhaseFailed because nothing broke. A headless run has nobody to
	// ask, so a tool the posture does not grant is denied without a word:
	// the engine handles it, writes a sentence saying it could not, and
	// exits zero. Read by its exit code alone that is a success, and a task
	// that did nothing sat in done where nobody would look at it again.
	//
	// Data["tool"] is what it was denied. What it sends a reader to do is
	// neither wait nor debug: it is to look at what the phase was allowed.
	PhaseDenied = "phase.denied"

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

	// Decided is the decision engine answering about a run that stopped.
	// Data carries choice, confidence, by and mode. Written whether or
	// not it was acted on: a supervisor that decided quietly is one no
	// reader can check. See internal/hunch.
	Decided = "decision.made"

	// GatePassed is a phase's verification check answering yes.
	// Data["left_running"] is written when the gate's shell exited but
	// something it started still held the output open: the check passed,
	// and Text is what it printed before the run stopped waiting on it.
	GatePassed = "gate.passed"
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

	// SupervisorConversationRemoved takes a whole conversation out of the
	// thread, naming it by its id in Data["conversation"].
	//
	// It marks and does not erase, for the reason a retraction does: this
	// record is appended to and never rewritten, so taking something out
	// puts something in. What changes is whether those turns are listed and
	// whether they are put in front of the model again — never whether they
	// were said. orbit export still has them.
	SupervisorConversationRemoved = "supervisor.conversation_removed"

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
