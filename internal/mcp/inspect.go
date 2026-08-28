package mcp

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/task"
)

// What one inspection is allowed to cost the caller's context.
//
// These are budgets and not truncation for its own sake. A phase that
// printed forty thousand characters is a phase whose tail is the useful
// part — the error is at the end — and a supervisor that spends its whole
// window on one log cannot then read the diff. Every field that is cut says
// so in the answer rather than being quietly shortened, because a log the
// model believes it read in full is worse than one it knows it did not.
const (
	timelineTail = 30
	thoughtTail  = 5
	thoughtChars = 600
	outputChars  = 4000
)

// inspectTask answers the question the cockpit's inspector answers: what is
// this task, what has it done, and what went wrong.
//
// It reads the record rather than the board row, because the row is the
// summary and the record is the evidence. The first version of this tool
// described itself as returning logs, thinking, the last error and the notes
// and returned none of them — twelve board columns under a description that
// promised the log.
func (sn Session) inspectTask(args map[string]any) CallToolResult {
	sb, err := sn.readBoard()
	if err != nil {
		return refuse(err)
	}

	row0, err := findTask(sb.board, stringArg(args, "task_id"), stringArg(args, "repo"))
	if err != nil {
		return refuse(err)
	}

	r, err := openTaskRepo(row0)
	if err != nil {
		return refuse(err)
	}

	t, err := task.Load(sb.store, r, row0.ID)
	if err != nil {
		return refuse(fmt.Errorf("load task %s: %w", row0.ID, err))
	}

	events, err := task.Events(sb.store, t)
	if err != nil {
		return refuse(fmt.Errorf("read the record of task %s: %w", row0.ID, err))
	}

	answer := row(row0)
	answer["text"] = t.Text
	answer["events_total"] = len(events)
	answer["timeline"] = timelineOf(events)
	answer["notes"] = notesOf(events)
	answer["dialogue"] = dialogueOf(events)
	answer["thinking"] = thinkingOf(events)
	answer["phases"] = phaseOutcomes(events)
	answer["gates"] = gatesOf(events)
	answer["last_error"] = lastErrorOf(events)
	answer["last_output"] = lastOutputOf(events)

	return reply(answer)
}

// timelineOf is the tail of the record, one entry per event, which is what
// the inspector's chronology tab draws.
func timelineOf(events []record.Event) []map[string]any {
	from := max(len(events)-timelineTail, 0)
	tail := events[from:]

	out := make([]map[string]any, 0, len(tail))
	for _, e := range tail {
		entry := map[string]any{"at": e.At, "kind": e.Kind}
		if e.Phase != "" {
			entry["phase"] = e.Phase
		}

		if e.Text != "" {
			entry["text"] = clip(e.Text, thoughtChars)
		}

		out = append(out, entry)
	}

	return out
}

// notesOf is every note left on this task, oldest first. Notes are not
// tailed: they are what a human or a supervisor chose to write down, there
// are never many of them, and dropping the first one is dropping the
// instruction the rest were responses to.
func notesOf(events []record.Event) []map[string]any {
	var notes []map[string]any

	for _, e := range events {
		if e.Kind != record.TaskNoted {
			continue
		}

		notes = append(notes, map[string]any{"at": e.At, "text": e.Text})
	}

	return notes
}

// dialogueOf is what has been done to this task from outside a run: the
// calls a supervisor made through this server, and the sessions somebody
// opened on it from the cockpit.
//
// It is beside the notes rather than among them because the two are read by
// different parties. A note is handed to the next phase; this is handed to
// nobody, and is the answer to the question a supervisor asks before acting
// twice — whether it, or the reader, has already tried this.
func dialogueOf(events []record.Event) []map[string]any {
	var out []map[string]any

	for _, e := range events {
		if e.Kind != record.TaskDialogue {
			continue
		}

		out = append(out, map[string]any{"at": e.At, "by": e.Data["by"], "text": e.Text})
	}

	return out
}

// thinkingOf is the last few thinking blocks the engine emitted, which is
// where the reason a task went wrong is usually stated before it goes wrong.
func thinkingOf(events []record.Event) []map[string]any {
	var all []record.Event

	for _, e := range events {
		if e.Kind == record.PhaseThought {
			all = append(all, e)
		}
	}

	from := max(len(all)-thoughtTail, 0)

	out := make([]map[string]any, 0, len(all)-from)
	for _, e := range all[from:] {
		out = append(out, map[string]any{"at": e.At, "phase": e.Phase, "text": clip(e.Text, thoughtChars)})
	}

	return out
}

// phaseOutcomes is what happened to each phase, in the order they ran.
func phaseOutcomes(events []record.Event) []map[string]any {
	var out []map[string]any

	for _, e := range events {
		switch e.Kind {
		case record.PhaseStarted:
			out = append(out, map[string]any{
				"phase":  e.Phase,
				"at":     e.At,
				"engine": e.Data["engine"],
				"model":  e.Data["model"],
				"state":  "started",
			})
		case record.PhaseFinished, record.PhaseFailed, record.PhaseCancelled, record.PhaseWaiting:
			state := strings.TrimPrefix(e.Kind, "phase.")
			if n := len(out) - 1; n >= 0 && out[n]["phase"] == e.Phase {
				out[n]["state"] = state
				if why := e.Data["error"]; why != "" {
					out[n]["error"] = why
				}

				if why := e.Data["why"]; why != "" {
					out[n]["waiting_on"] = why
				}

				continue
			}

			out = append(out, map[string]any{"phase": e.Phase, "at": e.At, "state": state})
		}
	}

	return out
}

// gatesOf is every gate check this task's phases ran, and how each answered.
func gatesOf(events []record.Event) []map[string]any {
	var gates []map[string]any

	for _, e := range events {
		if e.Kind != record.GatePassed && e.Kind != record.GateFailed {
			continue
		}

		gates = append(gates, map[string]any{
			"at":     e.At,
			"phase":  e.Phase,
			"passed": e.Kind == record.GatePassed,
			"text":   clip(e.Text, thoughtChars),
		})
	}

	return gates
}

// lastErrorOf is the most recent thing that went wrong, or nil when nothing
// has. It is nil and not an empty string: a supervisor branching on "did
// this fail" must not have to tell "" from absent.
func lastErrorOf(events []record.Event) map[string]any {
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		switch e.Kind {
		case record.TaskFailed, record.PhaseFailed, record.TaskTimedOut, record.TaskAbandoned:
			return map[string]any{
				"at":     e.At,
				"kind":   e.Kind,
				"phase":  e.Phase,
				"reason": e.Data["error"],
				"text":   clip(e.Text, outputChars),
			}
		}
	}

	return nil
}

// lastOutputOf is the tail of the last thing an engine printed, whether the
// phase it printed it in finished, failed or was cancelled.
func lastOutputOf(events []record.Event) map[string]any {
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		switch e.Kind {
		case record.PhaseFinished, record.PhaseFailed, record.PhaseCancelled:
			if e.Text == "" {
				continue
			}

			return map[string]any{
				"phase":    e.Phase,
				"kind":     e.Kind,
				"text":     tail(e.Text, outputChars),
				"complete": len(e.Text) <= outputChars,
			}
		}
	}

	return nil
}

// clip shortens from the front, for text whose beginning is the point.
func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}

	return s[:n] + fmt.Sprintf("… (%d more characters)", len(s)-n)
}

// tail shortens from the back, for a log whose end is the point — an engine
// that failed says why in its last lines, and a head-truncated log is the
// half that does not contain the answer.
func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}

	return fmt.Sprintf("(%d earlier characters omitted) …", len(s)-n) + s[len(s)-n:]
}
