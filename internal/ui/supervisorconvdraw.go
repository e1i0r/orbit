package ui

// What the conversation list draws.

import (
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/ui/theme"
)

// conversationRows is the list, in place of the thread: one row per
// conversation with its title, and one under it saying when it was last
// spoken in and how long it is.
func (m Model) conversationRows(maxRows, cw int) []string {
	p := m.opts.Words
	convs := conversationsOf(m.supervisor.all)

	if len(convs) == 0 {
		return fill([]string{theme.Paint(theme.Dim).Render(fit(p.T("supervisor.no_conversations",
			"nothing has been said to the supervisor yet"), cw))}, maxRows)
	}

	var out []string

	for i, c := range convs {
		mark, ink := "  ", theme.Paint(theme.Dim)
		if i == m.supervisor.listSel {
			mark, ink = theme.Paint(theme.Accent).Bold(true).Render("▸ "), theme.Text(theme.Primary)
		}

		open := "○ "
		if c.id == m.supervisor.conversation {
			open = theme.Paint(theme.Live).Render("● ")
		}

		out = append(out, fit(mark+open+ink.Render(c.title), cw))
		out = append(out, fit("    "+theme.Paint(theme.Dim).Render(strings.Join([]string{
			m.said(c),
			p.P("supervisor.msg_count2", c.turns, "{n} message", "{n} messages"),
		}, " · ")), cw))
	}

	// Top-aligned, unlike the thread: a thread is read from its end and a
	// list from its start, and a list pushed to the floor of a tall
	// terminal is one nobody finds the top of.
	if len(out) <= maxRows {
		return fill(out, maxRows)
	}

	from := max(min(m.supervisor.listSel*2-maxRows/2, len(out)-maxRows), 0)

	return out[from : from+maxRows]
}

// said is when a conversation was last spoken in.
//
// Today and yesterday are named rather than counted, because that is how
// somebody looking for the conversation they had this morning thinks of it;
// anything older is a date, which is what they would look for instead.
func (m Model) said(c conversation) string {
	p := m.opts.Words

	if c.last.IsZero() {
		return ""
	}

	now := m.now
	if now.IsZero() {
		now = c.last
	}

	switch days := daysBetween(c.last, now); days {
	case 0:
		return p.T("supervisor.today", "today")
	case 1:
		return p.T("supervisor.yesterday", "yesterday")
	default:
		return c.last.Format("2 Jan")
	}
}

// daysBetween is how many calendar days apart two moments are, counted in
// the local day rather than in hours: a conversation at eleven last night is
// yesterday's at nine this morning, and ten hours is not what says so.
func daysBetween(then, now time.Time) int {
	a := time.Date(then.Year(), then.Month(), then.Day(), 0, 0, 0, 0, time.Local)
	b := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	return int(b.Sub(a).Hours() / 24)
}

// conversationWays is the line under the list.
func (m Model) conversationWays() []string {
	p := m.opts.Words

	return []string{
		theme.Paint(theme.Dim).Render(p.T("supervisor.list_note",
			"a conversation is disposable: what is worth keeping went to what Orbit knows when you said /rule or /aware")),
		"",
		theme.Paint(theme.Dim).Render(p.T("supervisor.list_ways2",
			"[↑↓] pick · [↵] open · [d] remove from the list · [ctrl+N] new · [esc] back")),
	}
}
