package panes

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

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
)

// phaseCells is the log's phase column. Ten cells fits every builtin phase
// name and truncates a longer one from a user flow, which is the same trade
// the board's own columns make.
const phaseCells = 10

// clockCells is a wall clock without its date — the date is the task's, and
// it is in the heading.
const clockCells = 8

// Drawn is a pane laid out: the rows, and which row each entry and each
// attempt's rule ended up on.
//
// Three values came back side by side and two of them were the same type, so
// the pointer read the seams as the heads for as long as nobody looked. The
// map a pane holds is written by the render that produced it — a second count
// is a second opinion about where a row is.
type Drawn struct {
	Rows  []string
	Heads map[int]int
	Seams map[int]int
}

// Timeline is one task's record, oldest at the top, newest at the bottom,
// seamed where one attempt ends and the next begins.
//
// The rows and the maps are built in one pass on purpose: a hit test that
// counted the rows a second time would be a second opinion about where a row
// is, and the day the two disagree the pointer opens the entry above the one
// it is on.
func Timeline(e Env) Drawn {
	w := max(e.Frame.Body.W, 1)
	if e.Failed != "" {
		return Drawn{Rows: []string{" " + theme.Paint(theme.Bad).Render(e.Failed)}}
	}

	if len(e.Entries) == 0 {
		return Drawn{Rows: []string{" " + theme.Paint(theme.Dim).Render(
			e.Words.T("log.empty", "nothing has been recorded about this task yet"))}}
	}

	out := make([]string, 0, len(e.Entries)+4)
	heads, seams := map[int]int{}, map[int]int{}

	for i, entry := range e.Entries {
		if entry.Attempted() {
			seams[len(out)] = entry.Attempt
			out = append(out, e.seam(entry, w))
		}

		if !e.attempt(entry.Attempt) {
			continue
		}

		rows, folds := e.logEntryLines(entry, i, w)
		if folds {
			heads[len(out)] = i
		}

		out = append(out, rows...)
	}

	return Drawn{Rows: out, Heads: heads, Seams: seams}
}

// seam is the line between one attempt and the next.
//
// It carries the attempt's number and the time it began, because those are
// the two things a reader comparing two attempts of the same task asks for
// first: which one this is, and how long ago it started. The arrow in front
// of it says the rule is also the lid on everything that attempt did.
func (e Env) seam(entry view.Entry, w int) string {
	label := e.Words.T("log.attempt", "attempt {n}", about("n", strconv.Itoa(entry.Attempt)))

	head, tail := "── "+label+" ", ""
	if at := clock(entry.At); at != "" {
		tail = " " + at + " ──"
	}

	mark := cells.Fold(e.attempt(entry.Attempt))
	rule := max(w-lipgloss.Width(mark)-lipgloss.Width(head)-lipgloss.Width(tail)-1, 0)

	return " " + theme.Text(theme.Tertiary).Render(mark) + theme.Paint(theme.Dim).Render(head+strings.Repeat("─", rule)+tail)
}

// logEntryLines is one event — the clock, the phase, what happened, and what
// it said — and whether that row folds.
//
// Whether there is more to show is decided here, by wrapping the detail at
// the measure it will be drawn at, and nowhere else: a row is offered an
// arrow only when opening it puts something on the screen that was not
// already there.
func (e Env) logEntryLines(entry view.Entry, i, w int) ([]string, bool) {
	word, role := e.logWord(entry)
	prefix := " " + theme.Paint(theme.Dim).Render(cells.Pad(clock(entry.At), clockCells, false)) + "  " +
		theme.Paint(theme.Dim).Render(cells.Pad(entry.Phase, phaseCells, false)) + "  " +
		theme.Paint(role).Render(word) + "  "

	detail := logDetail(entry)
	if detail == "" {
		return []string{strings.TrimRight(prefix, " ")}, false
	}

	// The arrow stands in the detail's own column rather than out in the
	// margin, and a row that does not fold pays for it in spaces, so that
	// every sentence on the tab starts in the same place.
	lead := clockCells + 2 + phaseCells + 2 + lipgloss.Width(word) + 3
	availW := max(20, w-lead-lipgloss.Width(cells.FoldShut)-2)

	// Wrapped to the measure, and the wrap breaks what has no break in it:
	// a tool call is written down as the arguments it was made with, and a
	// path or a JSON document has no space to wrap at. Cutting the row to
	// the measure instead is what this did, and what was cut was then on no
	// row at all — the fold counts rows, so it was told there was nothing to
	// open, and the reader had no way to reach the rest of the command.
	//
	// The rows and not the detail itself: a detail written over two short
	// lines fits on one row, and drawing it unwrapped would take the newline
	// it was written with onto the screen.
	wrapped := cells.Lines(detail, availW)

	if len(wrapped) <= 1 {
		return []string{prefix + strings.Repeat(" ", lipgloss.Width(cells.FoldShut)) + theme.Paint(theme.Dim).Render(wrapped[0])}, false
	}

	mark := theme.Text(theme.Tertiary).Render(cells.Fold(e.row(i)))

	// Closed, the detail is a qualifier of the word beside it and is set as
	// one. Open, it is what the reader asked to read.
	if !e.row(i) {
		return []string{prefix + mark + theme.Paint(theme.Dim).Render(wrapped[0])}, true
	}

	out := make([]string, 0, len(wrapped))
	out = append(out, prefix+mark+theme.Text(theme.Secondary).Render(wrapped[0]))

	indent := strings.Repeat(" ", lead+lipgloss.Width(cells.FoldShut))
	for _, wl := range wrapped[1:] {
		out = append(out, indent+theme.Text(theme.Secondary).Render(wl))
	}

	return out, true
}

// logWord is what the entry says happened, and the role it is painted in.
//
// A kind this build does not know is drawn as the record spelled it. That is
// the one string on this screen that is deliberately not translated: it is
// not a word, it is a key out of somebody else's log, and inventing a
// sentence for it would be inventing the meaning too.
func (e Env) logWord(entry view.Entry) (string, theme.Role) {
	p := e.Words

	switch entry.What() {
	case view.EntryWritten:
		return p.T("log.written", "written down"), theme.Dim
	case view.EntryStarted:
		return p.T("log.started", "started"), theme.Accent
	case view.EntryFinished:
		return p.T("log.finished", "finished"), theme.OK
	case view.EntryFailed:
		return p.T("log.failed", "failed"), theme.Bad
	case view.EntryCancelled:
		return p.T("log.cancelled", "cancelled"), theme.Dim
	case view.EntryRequeued:
		return p.T("log.requeued", "back in to do"), theme.Warn
	case view.EntryTimedOut:
		return p.T("log.timed_out", "timed out"), theme.Bad
	case view.EntryAbandoned:
		return p.T("log.abandoned", "abandoned"), theme.Warn
	case view.EntryRead:
		return p.T("log.read", "read"), theme.Dim
	case view.EntryWaiting:
		return p.T("log.waiting", "waiting"), theme.Warn
	case view.EntryResumed:
		return p.T("log.resumed", "let go again"), theme.Accent
	case view.EntryRetried:
		return p.T("log.retried", "trying again"), theme.Warn
	case view.EntryGatePassed:
		return p.T("log.gate_passed", "gate passed"), theme.OK
	case view.EntryGateFailed:
		return p.T("log.gate_failed", "gate failed"), theme.Warn
	case view.EntryRefused:
		return p.T("log.refused", "refused"), theme.Bad
	case view.EntryToolCall:
		return p.T("log.tool_call", "tool call"), theme.Live
	case view.EntryThought:
		return p.T("log.thought", "thought"), theme.Dim
	case view.EntryStuck:
		return p.T("log.stuck", "stuck"), theme.Bad
	case view.EntryOverBudget:
		return p.T("log.over_budget", "over budget"), theme.Bad
	case view.EntryOverDiff:
		return p.T("log.over_diff", "change too big"), theme.Bad
	case view.EntryNewDependency:
		return p.T("log.new_dependency", "new dependency"), theme.Bad
	case view.EntryContradicts:
		return p.T("log.contradicts", "against a decision"), theme.Bad
	case view.EntryLoopChecked:
		return p.T("log.loop_checked", "loop checked"), theme.Accent
	case view.EntryApproved:
		return p.T("log.approved", "dependency approved"), theme.OK
	case view.EntryDecision:
		return p.T("log.decision", "decided"), theme.Accent
	case view.EntrySuperseded:
		return p.T("log.superseded", "decision replaced"), theme.Warn
	case view.EntryRepoJoined:
		return p.T("log.repo_joined", "repository joined"), theme.Accent
	case view.EntryDeliverAsked:
		return p.T("log.deliver_asked", "asked for"), theme.Accent
	case view.EntryDeliverAnswered:
		// The one kind here whose word depends on what it carries: the
		// same event ends a verb that worked and one that broke, and a
		// timeline that called both of them "answered" would make the
		// reader open the row to find out which.
		if entry.Cause != "" {
			return p.T("log.deliver_broke", "came back broken"), theme.Bad
		}

		return p.T("log.deliver_answered", "came back"), theme.OK
	case view.EntryUnreadable:
		return p.T("log.unreadable", "this line could not be read"), theme.Bad
	}

	return entry.Kind, theme.Dim
}

// logDetail is the one fact worth putting beside the word, and it is always
// something the record said rather than something this package composed: the
// engine that was asked, the reason a phase stopped, or whatever was written
// down. It is the whole of that fact — where it is cut to a row, and whether
// it is cut at all, is the drawing above.
func logDetail(e view.Entry) string {
	switch e.What() {
	case view.EntryStarted:
		return strings.TrimSpace(e.Engine + " " + e.Model)
	case view.EntryToolCall:
		if e.Text != "" {
			return view.ToolLine(e.Tool, e.Text)
		}

		return e.Tool
	case view.EntryRefused:
		// The whole of what was refused, and not its first line. The row is
		// wrapped and folded like every other row here, so a reason written
		// over three lines is three rows under an arrow — cutting it at the
		// first break put the rest of it on no row at all, and left the row
		// with nothing to open.
		return e.Tool + ": " + e.Text
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
	case view.EntryDeliverAsked:
		if e.By != "" {
			return e.Verb + " · " + e.By
		}

		return e.Verb
	case view.EntryDeliverAnswered:
		if e.Cause != "" {
			return e.Verb + ": " + e.Cause
		}

		return strings.TrimSpace(e.Verb + " " + e.Said())
	}

	return e.Said()
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
