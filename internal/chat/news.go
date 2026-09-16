package chat

// What is worth interrupting somebody for.
//
// The hard part of a notification is not sending it. It is that a channel
// which tells you everything is a channel you mute, and a muted channel is
// worse than none — you believed you would be told.
//
// So the line is drawn at **what changes where a task is**, and the list is
// closed. A task's whole arc is a handful of these: it started, it changed
// engine, it stopped for a reason, a pull request opened, it merged. Somebody
// away from their desk can follow that. What is left out is everything
// inside a phase — a phase starting, a tool call, a thought, a cost going
// up — which is a run working rather than a run moving, and there are
// hundreds of them.
//
// And every one of them carries what to do about it. A message saying
// "ACME-3 needs you" is a phone opened for nothing; the same message with the
// command that answers it is a decision made from a bus stop.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/words"
)

// A Happening is one thing the record says happened, and the task it was
// written about.
type Happening struct {
	Task  string
	Event record.Event
}

// worthTelling is every kind that moves a task, in the order a task meets
// them.
var worthTelling = map[string]bool{
	// It began, and it changed hands.
	record.TaskStarted: true,
	record.TaskRelayed: true,

	// It stopped and nothing moves on its own.
	record.PhaseWaiting:      true,
	record.TaskStuck:         true,
	record.TaskNeedsEngine:   true,
	record.TaskNoEngine:      true,
	record.TaskFailed:        true,
	record.TaskOverBudget:    true,
	record.TaskOverDiff:      true,
	record.TaskNewDependency: true,
	record.TaskContradicts:   true,

	// It ended.
	record.TaskFinished:    true,
	record.DeliverAnswered: true,
	record.TaskMerged:      true,
	record.TaskCancelled:   true,
	record.TaskTimedOut:    true,
	record.TaskAbandoned:   true,
}

// News is the message one happening is worth, and nothing for the ones that
// are not worth a message.
func News(h Happening, p *words.Printer) string {
	if !worthTelling[h.Event.Kind] {
		return ""
	}

	said, next := saying(h, p)
	if said == "" {
		return ""
	}

	out := Strong + h.Task + endStrong + " — " + said
	if next != "" {
		out += "\n\n" + next
	}

	return out
}

// saying is what happened and the command that answers it.
func saying(h Happening, p *words.Printer) (said, next string) {
	e := h.Event

	switch e.Kind {
	case record.TaskStarted:
		return p.T("news.started", "started"), "/task show " + h.Task
	case record.TaskRelayed:
		return p.T("news.relayed", "{from} handed it to {to} in {phase}",
			about("from", e.Data["from"]), about("to", e.Data["to"]),
			about("phase", e.Phase)), ""
	case record.TaskFailed:
		return p.T("news.failed", "the run stopped: {why}", about("why", firstLine(e.Text))),
			"/task show " + h.Task
	case record.TaskOverBudget:
		return p.T("news.over_budget", "it has spent {spent} of {budget}",
				about("spent", e.Data["spent"]), about("budget", e.Data["budget"])),
			"/task show " + h.Task
	case record.TaskOverDiff:
		return p.T("news.over_diff", "it changed {lines} lines of {budget}",
				about("lines", e.Data["lines"]), about("budget", e.Data["budget"])),
			"/task diff " + h.Task
	case record.TaskNewDependency:
		return p.T("news.new_dependency", "it reached for {names}", about("names", e.Data["names"])),
			"/task approve " + h.Task
	case record.TaskContradicts:
		return p.T("news.contradicts", "the change goes against {decision}",
			about("decision", e.Data["decision"])), "/task show " + h.Task
	case record.TaskCancelled:
		return p.T("news.cancelled", "cancelled"), ""
	case record.TaskTimedOut:
		return p.T("news.timed_out", "it outlived its deadline"), "/task show " + h.Task
	case record.TaskAbandoned:
		return p.T("news.abandoned", "its process is gone"), "/task start " + h.Task
	case record.TaskMerged:
		return p.T("news.merged", "merged"), ""
	case record.DeliverAnswered:
		return delivered(e, p), "/pr show " + h.Task
	case record.PhaseWaiting:
		return p.T("news.waiting", "waiting at a gate in {phase}", about("phase", e.Phase)),
			"/task continue " + h.Task
	case record.TaskStuck:
		return p.T("news.stuck", "stuck after {attempts} attempts; nothing moves on its own",
				about("attempts", e.Data["attempts"])),
			"/task show " + h.Task
	case record.TaskNeedsEngine:
		// The engines that could take it are in the event, and naming them
		// is the difference between a question and a notification: the
		// answer is one of these words.
		free := strings.ReplaceAll(e.Data["engines"], ",", ", ")

		return p.T("news.needs_engine", "{engine} ran out in {phase}; {free} could take it on",
				about("engine", e.Data["from"]), about("phase", e.Phase), about("free", free)),
			"/task start " + h.Task + " -engine " + first(e.Data["engines"])
	case record.TaskNoEngine:
		return p.T("news.no_engine", "{engine} ran out in {phase}, and no engine has anything left",
				about("engine", e.Data["from"]), about("phase", e.Phase)),
			""
	case record.TaskFinished:
		return p.T("news.finished", "finished"), "/task show " + h.Task
	}

	return "", ""
}

// delivered is what one delivery verb came back with, and nothing for the
// ones that are not a milestone.
//
// A pull request opening is where a task stops being Orbit's and starts being
// the team's, which is the one thing here somebody else will see. Bringing a
// branch up to date or answering a review is the same task carrying on.
func delivered(e record.Event, p *words.Printer) string {
	if e.Data["verb"] != "pr" {
		return ""
	}

	if why := e.Data["error"]; why != "" {
		return p.T("news.pr_failed", "the pull request did not open: {why}", about("why", firstLine(why)))
	}

	return p.T("news.pr", "a pull request is open")
}

// firstLine is as much of something as belongs in a notification.
func firstLine(text string) string {
	said, _, _ := strings.Cut(strings.TrimSpace(text), "\n")

	return said
}

// first is the first of a comma-separated list, for the command a message
// offers: one to tap, not three to choose between.
func first(list string) string {
	name, _, _ := strings.Cut(list, ",")

	return name
}

// about names a value in a sentence.
func about(name, value string) words.Arg {
	return words.Arg{Name: name, Value: value}
}
