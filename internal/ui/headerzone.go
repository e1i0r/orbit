package ui

// Where the header line put things. The line is built once, and the pieces
// that answer a click report the cells they landed on as they are joined, so
// nothing here holds a column number of its own.

import (
	"fmt"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// headerZone is one thing on the drawn header line that answers a click, and
// the cells it occupies, counted from the left edge of the terminal.
type headerZone struct {
	target Target
	x, w   int
}

// headerField is one of the standing facts on the right of the line: what is
// drawn, and the name hitHeader knows it by. A field with no name is drawn
// and read but never clicked.
type headerField struct {
	name string
	text string
}

// queueBadge is one queue badge on the left of the line: what is drawn, and
// the queue that clicking it narrows the board down to.
type queueBadge struct {
	band view.Band
	text string
}

// queueBadges are the four queues as badges, with the one being filtered on
// marked.
//
// The mark is what makes measuring these unavoidable rather than tidy:
// PillActive prefixes an arrow, so the selected badge is two cells wider
// than it was and every badge after it moves.
func (m Model) queueBadges() []queueBadge {
	if len(m.board.Counts) < 4 {
		return nil
	}

	p := m.opts.Words
	pill := func(b view.Band, icon, label string, in ink, count int) string {
		text := fmt.Sprintf("%s %s %d", icon, label, count)
		if m.queueFilter != nil && *m.queueFilter == b {
			return PillActive(text, in.fg, in.bg)
		}

		return Pill(text, in.fg, in.bg)
	}

	// Indexed by the band, not by the position the badge is drawn in.
	// board.Counts is filled by view.BandOf, whose order is ToDo, NeedsYou,
	// Running, Done; these are drawn in a different one, so 0..3 down this
	// list hands the Running badge the count of the tasks waiting on the
	// reader and the Needs You badge the count of the ones still running.
	return []queueBadge{
		{view.ToDo, pill(view.ToDo, "📋",
			p.T("queue.todo", "To Do"), inkToDo, m.board.Counts[view.ToDo])},
		{view.Running, pill(view.Running, "⚡",
			p.T("queue.in_flight", "Running"), inkRunning, m.board.Counts[view.Running])},
		{view.NeedsYou, pill(view.NeedsYou, "💬",
			p.T("queue.needs_you", "Needs You"), inkNeedsYou, m.board.Counts[view.NeedsYou])},
		{view.Done, pill(view.Done, "🏁",
			p.T("queue.done", "Done"), inkDone, m.board.Counts[view.Done])},
	}
}

// badgeTexts is the badges as headerLeft joins them.
func badgeTexts(badges []queueBadge) []string {
	out := make([]string, 0, len(badges))
	for _, b := range badges {
		out = append(out, b.text)
	}

	return out
}

// fieldTexts is the fields as headerLayout joins them.
func fieldTexts(fields []headerField) []string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, f.text)
	}

	return out
}

// placeBadges walks the badges the way headerLeft joined them — one space
// between — starting at x, and says where each one landed.
func placeBadges(badges []queueBadge, x int) []headerZone {
	out := make([]headerZone, 0, len(badges))

	for _, b := range badges {
		cells := lipgloss.Width(b.text)
		out = append(out, headerZone{
			target: Target{Kind: TargetHeaderQueue, Band: b.band},
			x:      x,
			w:      cells,
		})

		x += cells + 1
	}

	return out
}

// placeFields walks the right-hand fields the way headerLayout joined them,
// starting at x, and says where each one landed. A field with no name is
// stepped over: it takes up cells and answers nothing.
func placeFields(fields []headerField, x int) []headerZone {
	out := make([]headerZone, 0, len(fields))

	for _, f := range fields {
		cells := lipgloss.Width(f.text)
		if f.name != "" {
			out = append(out, headerZone{
				target: Target{Kind: TargetHeaderField, Field: f.name},
				x:      x,
				w:      cells,
			})
		}

		x += cells + lipgloss.Width(headerGap)
	}

	return out
}
