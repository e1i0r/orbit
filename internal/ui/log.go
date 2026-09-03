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
	lines, _, _ := m.logRows()

	return lines
}

// logRows is that content and, beside it, which entry each row that folds is
// the head of and which attempt each seam belongs to.
//
// The rows and the map are built in one pass on purpose: a hit test that
// counted the rows a second time would be a second opinion about where a row
// is, and the day the two disagree the pointer opens the entry above the one
// it is on.
func (m Model) logRows() ([]string, map[int]int, map[int]int) {
	w := max(m.frame.Body.W, 1)
	if m.logErr != nil {
		return []string{" " + Paint(Bad).Render(m.errSaid(m.logErr))}, nil, nil
	}

	if len(m.entries) == 0 {
		return []string{" " + Paint(Dim).Render(m.opts.Words.T("log.empty", "nothing has been recorded about this task yet"))}, nil, nil
	}

	out := make([]string, 0, len(m.entries)+4)
	heads, seams := map[int]int{}, map[int]int{}

	for i, e := range m.entries {
		if e.Attempted() {
			seams[len(out)] = e.Attempt
			out = append(out, m.seam(e, w))
		}

		if !m.attemptOpen(e.Attempt) {
			continue
		}

		rows, folds := m.logEntryLines(e, i, w)
		if folds {
			heads[len(out)] = i
		}

		out = append(out, rows...)
	}

	return out, heads, seams
}

// seam is the line between one attempt and the next.
//
// It carries the attempt's number and the time it began, because those are
// the two things a reader comparing two attempts of the same task asks for
// first: which one this is, and how long ago it started. The arrow in front
// of it says the rule is also the lid on everything that attempt did.
func (m Model) seam(e view.Entry, w int) string {
	label := m.opts.Words.T("log.attempt", "attempt {n}", about("n", strconv.Itoa(e.Attempt)))

	head, tail := "── "+label+" ", ""
	if at := clock(e.At); at != "" {
		tail = " " + at + " ──"
	}

	mark := foldMark(m.attemptOpen(e.Attempt))
	rule := max(w-lipgloss.Width(mark)-lipgloss.Width(head)-lipgloss.Width(tail)-1, 0)

	return " " + Text(Tertiary).Render(mark) + Paint(Dim).Render(head+strings.Repeat("─", rule)+tail)
}

// logEntryLines is one event — the clock, the phase, what happened, and what
// it said — and whether that row folds.
//
// Whether there is more to show is decided here, by wrapping the detail at
// the measure it will be drawn at, and nowhere else: a row is offered an
// arrow only when opening it puts something on the screen that was not
// already there.
func (m Model) logEntryLines(e view.Entry, i, w int) ([]string, bool) {
	word, role := m.logWord(e)
	prefix := " " + Paint(Dim).Render(pad(clock(e.At), clockCells, false)) + "  " +
		Paint(Dim).Render(pad(e.Phase, phaseCells, false)) + "  " +
		Paint(role).Render(word) + "  "

	detail := m.logDetail(e)
	if detail == "" {
		return []string{strings.TrimRight(prefix, " ")}, false
	}

	// The arrow stands in the detail's own column rather than out in the
	// margin, and a row that does not fold pays for it in spaces, so that
	// every sentence on the tab starts in the same place.
	lead := clockCells + 2 + phaseCells + 2 + lipgloss.Width(word) + 3
	availW := max(20, w-lead-lipgloss.Width(foldShut)-2)

	// Wrapped to the measure and then cut to it: a tool call is written down
	// as the arguments it was made with and JSON has no spaces to wrap at,
	// so a row can come back from the wrap longer than it was wrapped to.
	//
	// The rows and not the detail itself: a detail written over two short
	// lines fits on one row, and drawing it unwrapped would take the newline
	// it was written with onto the screen.
	wrapped := splitIntoLines(detail, availW)
	for n, wl := range wrapped {
		wrapped[n] = fit(wl, availW)
	}

	if len(wrapped) <= 1 {
		return []string{prefix + strings.Repeat(" ", lipgloss.Width(foldShut)) + Paint(Dim).Render(wrapped[0])}, false
	}

	mark := Text(Tertiary).Render(foldMark(m.rowOpen(tabTimeline, i)))

	// Closed, the detail is a qualifier of the word beside it and is set as
	// one. Open, it is what the reader asked to read.
	if !m.rowOpen(tabTimeline, i) {
		return []string{prefix + mark + Paint(Dim).Render(wrapped[0])}, true
	}

	out := make([]string, 0, len(wrapped))
	out = append(out, prefix+mark+Text(Secondary).Render(wrapped[0]))

	indent := strings.Repeat(" ", lead+lipgloss.Width(foldShut))
	for _, wl := range wrapped[1:] {
		out = append(out, indent+Text(Secondary).Render(wl))
	}

	return out, true
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
	case view.EntryRequeued:
		return p.T("log.requeued", "back in to do"), Warn
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
	case view.EntryRetried:
		return p.T("log.retried", "trying again"), Warn
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
	case view.EntryStuck:
		return p.T("log.stuck", "stuck"), Bad
	case view.EntryOverBudget:
		return p.T("log.over_budget", "over budget"), Bad
	case view.EntryOverDiff:
		return p.T("log.over_diff", "change too big"), Bad
	case view.EntryDecision:
		return p.T("log.decision", "decided"), Accent
	case view.EntrySuperseded:
		return p.T("log.superseded", "decision replaced"), Warn
	case view.EntryRepoJoined:
		return p.T("log.repo_joined", "repository joined"), Accent
	case view.EntryUnreadable:
		return p.T("log.unreadable", "this line could not be read"), Bad
	}

	return e.Kind, Dim
}

// logDetail is the one fact worth putting beside the word, and it is always
// something the record said rather than something this package composed: the
// engine that was asked, the reason a phase stopped, or whatever was written
// down. It is the whole of that fact — where it is cut to a row, and whether
// it is cut at all, is the drawing above.
func (m Model) logDetail(e view.Entry) string {
	switch e.What() {
	case view.EntryStarted:
		return strings.TrimSpace(e.Engine + " " + e.Model)
	case view.EntryToolCall:
		if e.Text != "" {
			return formatLogTool(e.Tool, e.Text)
		}

		return e.Tool
	case view.EntryRefused:
		return e.Tool + ": " + firstLine(e.Text)
	case view.EntryGatePassed, view.EntryGateFailed, view.EntryRetried:
		if e.Gate != "" {
			return e.Gate
		}
	case view.EntryFailed, view.EntryCancelled, view.EntryTimedOut:
		if e.Cause != "" {
			return e.Cause
		}
	case view.EntryRepoJoined:
		return e.Repo
	}

	return e.Said()
}

func formatLogTool(tool, args string) string {
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

	if head != "" {
		return tool + ": " + head
	}

	return tool
}

// firstLine is everything up to the first line break. A multi-line Text on a
// one-line row would take the rest of the pane with it, and a carriage
// return would put the rest of it on top of the row it is already on.
//
// SplitN and not Cut: this package may not slice a string, and Cut answers
// with two more values than the cut is asking for.
func firstLine(s string) string {
	head := strings.SplitN(s, "\n", 2)[0]
	head = strings.SplitN(head, "\r", 2)[0]

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
