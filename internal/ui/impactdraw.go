package ui

// What the impact pane draws.
//
// Each section says what it is before it says what it found. This reading is
// not something a reader has met before — a diff they know, a test failure
// they know, "this file usually comes along and did not" they do not — and a
// list of filenames under a heading nobody understands is a list nobody
// acts on.
//
// It also says where each thing comes from. The coupling is the history's
// word and not a proof: two files that always moved together may have been
// split apart on purpose today, and the reader is the one who knows which.

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
)

// impactRows is the pane.
func (m Model) impactRows() []string {
	p := m.opts.Words

	switch {
	case m.opts.Reader == nil:
		return []string{theme.Paint(theme.Dim).Render(p.T("impact.no_port", "this build cannot read the history"))}
	case m.weigh.reachAsking && !m.weigh.reachKnown:
		return []string{m.spinner(theme.Live) + theme.Paint(theme.Live).Render(p.T("impact.reading", "reading the history…"))}
	case !m.weigh.reachKnown:
		return []string{theme.Paint(theme.Dim).Render(p.T("impact.not_yet", "nothing read yet"))}
	case m.weigh.reachErr != nil:
		return []string{theme.Paint(theme.Bad).Render(p.T("impact.failed", "the history could not be read: {err}",
			about("err", m.errSaid(m.weigh.reachErr))))}
	case len(m.weigh.reach.Changed) == 0 && m.lastDelta() == nil && !m.weigh.checksKnown:
		return []string{theme.Paint(theme.Dim).Render(p.T("impact.no_changes", "this task changed no files, so there is nothing to weigh"))}
	}

	rows := m.compareRows()
	rows = append(rows, m.impactCoupled()...)
	rows = append(rows, m.impactContracts()...)

	return append(rows, m.impactDelta()...)
}

// impactCoupled is the first section: what the history says usually comes
// along, and did not.
func (m Model) impactCoupled() []string {
	p := m.opts.Words

	rows := []string{"", m.impactHead(p.T("impact.reach_title", "WHAT USUALLY COMES ALONG"))}
	rows = append(rows, m.explains(p.T("impact.reach_about",
		"files this repository has committed together with the ones this task changed, and that it did not touch this time."))...)
	rows = append(rows, m.explains(p.T("impact.reach_source",
		"read from the last {n} commits. It is what the history does, not a rule: a file left out on purpose is the normal case.",
		about("n", strconv.Itoa(m.weigh.reach.Commits))))...)
	rows = append(rows, "")

	if len(m.weigh.reach.Coupled) == 0 {
		return append(rows, theme.Paint(theme.OK).Render("  "+p.T("impact.reach_none",
			"nothing else follows these files often enough to mention"))+"\n", "")
	}

	changed := map[string][]string{}
	for _, c := range m.weigh.reach.Coupled {
		changed[c.With] = append(changed[c.With], impactLine(m, c))
	}

	for _, file := range m.weigh.reach.Changed {
		lines, held := changed[file]
		if !held {
			continue
		}

		rows = append(rows, "  "+theme.Text(theme.Primary).Render(file)+" "+
			theme.Paint(theme.Dim).Render(p.T("impact.changed_here", "changed here")))
		rows = append(rows, lines...)
		rows = append(rows, "")
	}

	return rows
}

// impactHead is one section's name with a rule after it.
//
// The three sections are three different kinds of claim — an exit code, a
// pattern in the history, and the engine's own account — and a reader who
// cannot see where one ends reads the third as though it carried the weight
// of the first. The rule is what says they are separate things.
func (m Model) impactHead(title string) string {
	head := theme.Paint(theme.Accent).Bold(true).Render(title) + " "

	if rule := min(m.frame.Body.W, 104) - lipgloss.Width(title) - 3; rule > 0 {
		head += theme.Paint(theme.Dim).Render(strings.Repeat("─", rule))
	}

	return head
}

// explains is one sentence of explanation, folded to the pane.
//
// The panes do not wrap on their own — a diff must not — so a paragraph that
// explains what a section is has to be folded where it is written, or the
// half of it past the right edge is the half nobody reads.
func (m Model) explains(sentence string) []string {
	var out []string

	for _, line := range splitIntoLines(sentence, max(m.frame.Body.W-4, 20)) {
		out = append(out, theme.Paint(theme.Dim).Render("  "+line))
	}

	return out
}

// impactLine is one file that follows: how often, out of how many, and that
// nobody touched it.
func impactLine(m Model, c repo.Coupled) string {
	p := m.opts.Words

	return "    " + theme.Paint(theme.Warn).Render("⚠ ") + theme.Text(theme.Primary).Render(c.File) + " " +
		theme.Paint(theme.Dim).Render(p.T("impact.follows", "{pct}% of the time ({times}/{of}) · not touched",
			about("pct", strconv.Itoa(int(c.Ratio()*100+0.5))),
			about("times", strconv.Itoa(c.Times)),
			about("of", strconv.Itoa(c.Of))))
}

// impactContracts is the second section: what the tests that were left out
// say they hold.
func (m Model) impactContracts() []string {
	p := m.opts.Words
	if len(m.weigh.reach.Contracts) == 0 {
		return nil
	}

	rows := []string{m.impactHead(p.T("impact.contracts_title", "WHAT THOSE TESTS SAY THEY HOLD"))}
	rows = append(rows, m.explains(p.T("impact.contracts_about",
		"the names of the tests in the files above, read as sentences. Nothing here was run: it is what somebody wrote down that the code guarantees."))...)
	rows = append(rows, "")

	byFile := map[string][]string{}

	var order []string

	for _, c := range m.weigh.reach.Contracts {
		if _, seen := byFile[c.File]; !seen {
			order = append(order, c.File)
		}

		byFile[c.File] = append(byFile[c.File], c.Says)
	}

	for _, file := range order {
		rows = append(rows, "  "+theme.Text(theme.Primary).Render(file))
		for _, says := range byFile[file] {
			rows = append(rows, "    "+theme.Paint(theme.Dim).Render("· ")+theme.Text(theme.Primary).Render(says))
		}

		rows = append(rows, "")
	}

	return rows
}

// impactMark is the count beside the tab's name, and nothing when the
// reading found nothing.
func (m Model) impactMark() string {
	n := m.impactWarnings()
	if n == 0 {
		return ""
	}

	return fmt.Sprintf(" ⚠%s", strings.TrimSpace(strconv.Itoa(n)))
}

// impactDelta is the third section: what the engine says its own change
// asks, promises, assumed and decided against.
//
// Last, and marked. The two sections above it are the repository's own
// history; this one is a claim by the thing whose work is being weighed, and
// nothing verified it — nothing can, for "assumes UTC timestamps". It is
// here because the discarded alternatives exist nowhere else: they die with
// the run, and the next person to touch that code pays again to find out why
// the obvious approach was not taken.
func (m Model) impactDelta() []string {
	p := m.opts.Words

	d := m.lastDelta()
	if d == nil {
		return nil
	}

	rows := []string{m.impactHead(p.T("impact.delta_title", "WHAT THE AGENT SAYS IT DID"))}
	rows = append(rows, m.explains(p.T("impact.delta_about",
		"the engine's own account of what this change asks of its callers and what it now promises them. Nobody verified it — no command can — and the last part is the only place a rejected approach is written down."))...)
	rows = append(rows, "")

	for _, part := range []struct {
		head  string
		lines []string
	}{
		{p.T("impact.delta_needs", "callers must now"), d.Needs},
		{p.T("impact.delta_guarantees", "it now holds"), d.Guarantees},
		{p.T("impact.delta_assumes", "it took for granted"), d.Assumes},
		{p.T("impact.delta_instead", "considered and not taken"), d.Instead},
	} {
		if len(part.lines) == 0 {
			continue
		}

		rows = append(rows, "  "+theme.Paint(theme.Dim).Render(part.head))
		for _, line := range part.lines {
			rows = append(rows, "    "+theme.Paint(theme.Dim).Render("· ")+theme.Text(theme.Primary).Render(line))
		}

		rows = append(rows, "")
	}

	return rows
}

// lastDelta is the one the attempt that stands wrote, and nothing when no
// attempt wrote one.
//
// The last, for the reason the report pane draws the last: a task run three
// times said this three times, and the two before it are about work that was
// thrown away.
func (m Model) lastDelta() *view.Delta {
	for i := len(m.entries) - 1; i >= 0; i-- {
		if m.entries[i].Delta != nil {
			return m.entries[i].Delta
		}
	}

	return nil
}
