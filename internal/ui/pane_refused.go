package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// refusedLines is the refused tab's content, ready for the pane.
func (m Model) refusedLines() []string {
	lines, _ := m.refusedRows()

	return lines
}

// refusedRows is that content and, beside it, which denial each row that
// folds stands for, laid out in one pass for the reason logRows is.
func (m Model) refusedRows() ([]string, map[int]int) {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{"  " + Paint(Bad).Render(m.errSaid(m.logErr))}, nil
	}

	var denials []view.Entry

	for _, e := range m.entries {
		if e.What() == view.EntryRefused {
			denials = append(denials, e)
		}
	}

	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(p.T("refused.title", "Permissions & Security Sandbox")),
		"  " + Paint(Dim).Render(p.T("refused.subtitle", "what the sandbox forbids, and what it attempted")),
		"",
	}

	// What the sandbox stopped this run doing, above the standing rules it
	// stopped them by.
	out = append(out, "  "+Paint(Accent).Render(p.T("refused.in_this_run", "IN THIS RUN")))

	heads := map[int]int{}

	if len(denials) == 0 {
		out = append(out,
			"    "+Paint(OK).Render(p.T("refused.none_denied", "no commands or actions were denied")),
			"    "+Paint(Dim).Render(p.T("refused.all_allowed", "everything it attempted was permitted to run")),
		)
	} else {
		for i, d := range denials {
			rows, folds := m.denialRows(d, i)
			if folds {
				heads[len(out)] = i
			}

			out = append(out, rows...)
		}
	}

	out = append(out, "")

	// 2. Las reglas fijas del sandbox
	out = append(out,
		"  "+Paint(Accent).Render(p.T("refused.rules_title", "THE RULES · sandbox constraints")),
		fmt.Sprintf("    %s %-24s %s", Paint(Dim).Render("✗"), "psql / mongosh", Paint(Dim).Render(p.T("refused.rule_db", "protected databases, readable only"))),
		fmt.Sprintf("    %s %-24s %s", Paint(Dim).Render("✗"), "aws / cloud-cli", Paint(Dim).Render(p.T("refused.rule_cloud", "cloud services and outside credentials"))),
		fmt.Sprintf("    %s %-24s %s", Paint(Dim).Render("✗"), "git push", Paint(Dim).Render(p.T("refused.rule_push", "the branch belongs to the operator or the runner"))),
		fmt.Sprintf("    %s %-24s %s", Paint(Dim).Render("✗"), "git remote / config", Paint(Dim).Render(p.T("refused.rule_remote", "the repository's own configuration"))),
		fmt.Sprintf("    %s %-24s %s", Paint(Dim).Render("✗"), "gh pr merge", Paint(Dim).Render(p.T("refused.rule_merge", "merging and publishing pull requests"))),
		"",
		"  "+Paint(Dim).Render(p.T("refused.policy_note", "a forbidden action fails on the spot and the model carries on")),
		"",
	)

	return out, heads
}

// denialRows is one refusal — what was reached for, and what the sandbox said
// back — and whether there is more of it than the row is showing.
//
// What the sandbox writes down is a paragraph often enough that it cannot be
// set on one row: a refusal drawn unwrapped loses everything past the margin,
// which is the half that says why.
func (m Model) denialRows(d view.Entry, i int) ([]string, bool) {
	tool := d.Tool
	if tool == "" {
		tool = m.opts.Words.T("refused.unnamed_tool", "command")
	}

	head := "    " + Paint(Bad).Render("✗") + " " + Paint(Accent).Render(tool) + ": "

	lead := 4 + 1 + 1 + lipgloss.Width(tool) + 2
	availW := max(20, max(m.frame.Body.W, 1)-lead-lipgloss.Width(foldShut)-2)

	var body []string

	for _, l := range strings.Split(strings.TrimSpace(d.Text), "\n") {
		if l = strings.TrimSpace(l); l == "" {
			continue
		}

		for _, wl := range splitIntoLines(l, availW) {
			body = append(body, fit(wl, availW))
		}
	}

	if len(body) == 0 {
		return []string{strings.TrimRight(head, ": ")}, false
	}

	if len(body) == 1 {
		return []string{head + strings.Repeat(" ", lipgloss.Width(foldShut)) + Paint(Bad).Render(body[0])}, false
	}

	open := m.rowOpen(tabRefused, i)
	mark := Text(Tertiary).Render(foldMark(open))

	if !open {
		return []string{head + mark + Paint(Bad).Render(body[0])}, true
	}

	out := []string{head + mark + Paint(Bad).Render(body[0])}

	indent := strings.Repeat(" ", lead+lipgloss.Width(foldShut))
	for _, l := range body[1:] {
		out = append(out, indent+Paint(Bad).Render(l))
	}

	return out, true
}
