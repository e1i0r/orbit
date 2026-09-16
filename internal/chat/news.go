package chat

// What is worth interrupting somebody for.
//
// The hard part of a notification is not sending it. It is that a channel
// which tells you everything is a channel you mute, and a muted channel is
// worse than none — you believed you would be told.
//
// So the list is short and it is closed. Five endings, all of them already
// named in the record, and nothing else: a phase starting, a tool call, a
// cost going up are things happening, not news. What they have in common is
// that each one is a run that has stopped and will not start again on its
// own.
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

// worthTelling is every kind that stops a run and needs a person.
var worthTelling = map[string]bool{
	record.PhaseWaiting:    true,
	record.TaskStuck:       true,
	record.TaskNeedsEngine: true,
	record.TaskNoEngine:    true,
	record.TaskFinished:    true,
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
