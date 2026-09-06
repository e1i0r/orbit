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

	"github.com/e1i0r/orbit/internal/repo"
)

// impactRows is the pane.
func (m Model) impactRows() []string {
	p := m.opts.Words

	switch {
	case m.opts.Reader == nil:
		return []string{Paint(Dim).Render(p.T("impact.no_port", "this build cannot read the history"))}
	case m.impactAsking && !m.impactKnown:
		return []string{m.spinner(Live) + Paint(Live).Render(p.T("impact.reading", "reading the history…"))}
	case !m.impactKnown:
		return []string{Paint(Dim).Render(p.T("impact.not_yet", "nothing read yet"))}
	case m.impactErr != nil:
		return []string{Paint(Bad).Render(p.T("impact.failed", "the history could not be read: {err}",
			about("err", m.errSaid(m.impactErr))))}
	case len(m.impact.Changed) == 0:
		return []string{Paint(Dim).Render(p.T("impact.no_changes", "this task changed no files, so there is nothing to weigh"))}
	}

	rows := m.impactCoupled()

	return append(rows, m.impactContracts()...)
}

// impactCoupled is the first section: what the history says usually comes
// along, and did not.
func (m Model) impactCoupled() []string {
	p := m.opts.Words

	rows := []string{"", Paint(Accent).Bold(true).Render(p.T("impact.reach_title", "WHAT USUALLY COMES ALONG"))}
	rows = append(rows, m.explains(p.T("impact.reach_about",
		"files this repository has committed together with the ones this task changed, and that it did not touch this time."))...)
	rows = append(rows, m.explains(p.T("impact.reach_source",
		"read from the last {n} commits. It is what the history does, not a rule: a file left out on purpose is the normal case.",
		about("n", strconv.Itoa(m.impact.Commits))))...)
	rows = append(rows, "")

	if len(m.impact.Coupled) == 0 {
		return append(rows, Paint(OK).Render("  "+p.T("impact.reach_none",
			"nothing else follows these files often enough to mention"))+"\n", "")
	}

	changed := map[string][]string{}
	for _, c := range m.impact.Coupled {
		changed[c.With] = append(changed[c.With], impactLine(m, c))
	}

	for _, file := range m.impact.Changed {
		lines, held := changed[file]
		if !held {
			continue
		}

		rows = append(rows, "  "+Text(Primary).Render(file)+" "+
			Paint(Dim).Render(p.T("impact.changed_here", "changed here")))
		rows = append(rows, lines...)
		rows = append(rows, "")
	}

	return rows
}

// explains is one sentence of explanation, folded to the pane.
//
// The panes do not wrap on their own — a diff must not — so a paragraph that
// explains what a section is has to be folded where it is written, or the
// half of it past the right edge is the half nobody reads.
func (m Model) explains(sentence string) []string {
	var out []string

	for _, line := range splitIntoLines(sentence, max(m.frame.Body.W-4, 20)) {
		out = append(out, Paint(Dim).Render("  "+line))
	}

	return out
}

// impactLine is one file that follows: how often, out of how many, and that
// nobody touched it.
func impactLine(m Model, c repo.Coupled) string {
	p := m.opts.Words

	return "    " + Paint(Warn).Render("⚠ ") + Text(Primary).Render(c.File) + " " +
		Paint(Dim).Render(p.T("impact.follows", "{pct}% of the time ({times}/{of}) · not touched",
			about("pct", strconv.Itoa(int(c.Ratio()*100+0.5))),
			about("times", strconv.Itoa(c.Times)),
			about("of", strconv.Itoa(c.Of))))
}

// impactContracts is the second section: what the tests that were left out
// say they hold.
func (m Model) impactContracts() []string {
	p := m.opts.Words
	if len(m.impact.Contracts) == 0 {
		return nil
	}

	rows := []string{Paint(Accent).Bold(true).Render(p.T("impact.contracts_title", "WHAT THOSE TESTS SAY THEY HOLD"))}
	rows = append(rows, m.explains(p.T("impact.contracts_about",
		"the names of the tests in the files above, read as sentences. Nothing here was run: it is what somebody wrote down that the code guarantees."))...)
	rows = append(rows, "")

	byFile := map[string][]string{}

	var order []string

	for _, c := range m.impact.Contracts {
		if _, seen := byFile[c.File]; !seen {
			order = append(order, c.File)
		}

		byFile[c.File] = append(byFile[c.File], c.Says)
	}

	for _, file := range order {
		rows = append(rows, "  "+Text(Primary).Render(file))
		for _, says := range byFile[file] {
			rows = append(rows, "    "+Paint(Dim).Render("· ")+Text(Primary).Render(says))
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
