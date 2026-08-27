package ui

// The log tab: one task's record, oldest at the top, newest at the bottom,
// seamed where one attempt ends and the next begins.
//
// The order is the record's own, and the newest entry is at the bottom
// because that is where a tail belongs — a log that grows upwards moves
// every line a reader is in the middle of. The seam is drawn from the
// entry's own attempt number rather than by matching on a kind here, so the
// day a kind is added the seams do not have to be taught about it.
//
// One entry is one line. The engine's output is not in this tab at all: it
// belongs to the evidence tab, where it is quoted whole, and a log that
// inlines a thousand lines of it is a log nobody can find the next event in.

import (
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// phaseCells is the log's phase column. Ten cells fits every builtin phase
// name and truncates a longer one from a user flow, which is the same trade
// the board's own columns make.
const phaseCells = 10

// clockCells is a wall clock without its date — the date is the task's, and
// it is in the heading.
const clockCells = 8

// logLines is the log tab's content, ready for the pane.
func (m Model) logLines() []string {
	w := max(m.frame.Body.W, 1)
	if m.logErr != nil {
		return []string{" " + Paint(Bad).Render(m.logErr.Error())}
	}
	if len(m.entries) == 0 {
		return []string{" " + Paint(Dim).Render(m.opts.Words.T("log.empty", "nothing has been recorded about this task yet"))}
	}
	out := make([]string, 0, len(m.entries)+4)
	for _, e := range m.entries {
		if e.Attempted() {
			out = append(out, m.seam(e, w))
		}
		out = append(out, m.logEntryLines(e, w)...)
	}
	return out
}

// seam is the line between one attempt and the next.
//
// It carries the attempt's number and the time it began, because those are
// the two things a reader comparing two attempts of the same task asks for
// first: which one this is, and how long ago it started.
func (m Model) seam(e view.Entry, w int) string {
	label := m.opts.Words.T("log.attempt", "attempt {n}", about("n", strconv.Itoa(e.Attempt)))
	head, tail := "── "+label+" ", ""
	if at := clock(e.At); at != "" {
		tail = " " + at + " ──"
	}
	rule := max(w-lipgloss.Width(head)-lipgloss.Width(tail)-1, 0)
	return " " + Paint(Dim).Render(head+strings.Repeat("─", rule)+tail)
}

// logEntryLines formats one event, wrapping long details across multiple aligned rows.
func (m Model) logEntryLines(e view.Entry, w int) []string {
	word, role := m.logWord(e)
	prefix := Paint(Dim).Render(pad(clock(e.At), clockCells, false)) + "  " +
		Paint(Dim).Render(pad(e.Phase, phaseCells, false)) + "  " +
		Paint(role).Render(word)
	detail := m.logDetail(e)
	if detail == "" {
		return []string{" " + prefix}
	}

	if !m.expandedDetail {
		return []string{" " + prefix + "  " + Paint(Dim).Render(detail)}
	}

	prefixW := clockCells + 2 + phaseCells + 2 + lipgloss.Width(word) + 2
	availW := max(20, w-prefixW-4)
	wrapped := splitIntoLines(detail, availW)
	if len(wrapped) <= 1 {
		return []string{" " + prefix + "  " + Paint(Dim).Render(detail)}
	}

	var out []string
	out = append(out, " "+prefix+"  "+Paint(Dim).Render(wrapped[0]))
	indent := strings.Repeat(" ", prefixW+1)
	for _, wl := range wrapped[1:] {
		out = append(out, indent+Paint(Dim).Render(wl))
	}
	return out
}

// logWord is what the entry says happened, and the role it is painted in.
//
// A kind this build does not know is drawn as the record spelled it. That is
// the one string on this screen that is deliberately not translated: it is
// not a word, it is a key out of somebody else's log, and inventing a
// sentence for it would be inventing the meaning too.
func (m Model) logWord(e view.Entry) (string, Role) {
	p := m.opts.Words
	switch e.What() {
	case view.EntryWritten:
		return p.T("log.written", "written down"), Dim
	case view.EntryStarted:
		return p.T("log.started", "started"), Accent
	case view.EntryFinished:
		return p.T("log.finished", "finished"), OK
	case view.EntryFailed:
		return p.T("log.failed", "failed"), Bad
	case view.EntryCancelled:
		return p.T("log.cancelled", "cancelled"), Dim
	case view.EntryTimedOut:
		return p.T("log.timed_out", "timed out"), Bad
	case view.EntryAbandoned:
		return p.T("log.abandoned", "abandoned"), Warn
	case view.EntryRead:
		return p.T("log.read", "read"), Dim
	case view.EntryWaiting:
		return p.T("log.waiting", "waiting"), Warn
	case view.EntryResumed:
		return p.T("log.resumed", "let go again"), Accent
	case view.EntryGatePassed:
		return p.T("log.gate_passed", "gate passed"), OK
	case view.EntryGateFailed:
		return p.T("log.gate_failed", "gate failed"), Warn
	case view.EntryRefused:
		return p.T("log.refused", "refused"), Bad
	case view.EntryToolCall:
		return p.T("log.tool_call", "tool call"), Live
	case view.EntryThought:
		return p.T("log.thought", "thought"), Dim
	case view.EntryUnreadable:
		return p.T("log.unreadable", "this line could not be read"), Bad
	}
	return e.Kind, Dim
}

// logDetail is the one fact worth putting beside the word, and it is always
// something the record said rather than something this package composed: the
// engine that was asked, the reason a phase stopped, or the first line of
// whatever was written down.
func (m Model) logDetail(e view.Entry) string {
	switch e.What() {
	case view.EntryStarted:
		return strings.TrimSpace(e.Engine + " " + e.Model)
	case view.EntryToolCall:
		if e.Args != "" {
			return formatLogTool(e.Tool, e.Args, m.expandedDetail)
		}
		return e.Tool
	case view.EntryRefused:
		return e.Tool + ": " + firstLine(e.Text)
	case view.EntryGatePassed, view.EntryGateFailed:
		if e.Gate != "" {
			return e.Gate
		}
	case view.EntryFailed, view.EntryCancelled, view.EntryTimedOut:
		if e.Cause != "" {
			if m.expandedDetail {
				return e.Cause
			}
			return firstLine(e.Cause)
		}
	}
	if m.expandedDetail {
		return e.Text
	}
	return firstLine(e.Text)
}

func formatLogTool(tool, args string, expanded bool) string {
	head := firstLine(args)
	if strings.HasPrefix(head, "{") {
		if idx := strings.Index(head, `"command"`); idx >= 0 {
			if after := strings.Index(head[idx:], `:"`); after >= 0 {
				val := head[idx+after+2:]
				if end := strings.Index(val, `"`); end >= 0 {
					head = val[:end]
				}
			}
		} else if idx := strings.Index(head, `"path"`); idx >= 0 {
			if after := strings.Index(head[idx:], `:"`); after >= 0 {
				val := head[idx+after+2:]
				if end := strings.Index(val, `"`); end >= 0 {
					head = val[:end]
				}
			}
		}
	}
	if !expanded && len(head) > 60 {
		head = head[:57] + "…"
	}
	if head != "" {
		return tool + ": " + head
	}
	return tool
}

// firstLine is everything up to the first newline. A multi-line Text on a
// one-line row would take the rest of the pane with it.
func firstLine(s string) string {
	head, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(head)
}

// clock is the wall time an entry was written, or nothing at all when the
// record's clock was damaged. A zero time drawn as a time would read as
// midnight on a day nobody was working.
func clock(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return at.Format("15:04:05")
}
