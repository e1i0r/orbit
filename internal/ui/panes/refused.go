package panes

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
)

// Refused is what the sandbox would not let this run do, and beside it
// which denial each row that folds stands for — laid out in one pass for
// the reason the timeline is.
func Refused(e Env) ([]string, map[int]int) {
	p := e.Words
	if e.Failed != "" {
		return []string{"  " + theme.Paint(theme.Bad).Render(e.Failed)}, nil
	}

	var denials []view.Entry

	for _, entry := range e.Entries {
		if entry.What() == view.EntryRefused {
			denials = append(denials, entry)
		}
	}

	out := []string{
		"",
		"  " + theme.Paint(theme.Accent).Bold(true).Render(p.T("refused.title", "Permissions & Security Sandbox")),
		"  " + theme.Paint(theme.Dim).Render(p.T("refused.subtitle", "what the sandbox forbids, and what it attempted")),
		"",
	}

	// What the sandbox stopped this run doing, above the standing rules it
	// stopped them by.
	out = append(out, "  "+theme.Paint(theme.Accent).Render(p.T("refused.in_this_run", "IN THIS RUN")))

	heads := map[int]int{}

	if len(denials) == 0 {
		out = append(out,
			"    "+theme.Paint(theme.OK).Render(p.T("refused.none_denied", "no commands or actions were denied")),
			"    "+theme.Paint(theme.Dim).Render(p.T("refused.all_allowed", "everything it attempted was permitted to run")),
		)
	} else {
		for i, d := range denials {
			rows, folds := e.denialRows(d, i)
			if folds {
				heads[len(out)] = i
			}

			out = append(out, rows...)
		}
	}

	out = append(out, "")

	// And the standing rules it stopped them by.
	out = append(out,
		"  "+theme.Paint(theme.Accent).Render(p.T("refused.rules_title", "THE RULES · sandbox constraints")),
		rule("psql / mongosh", p.T("refused.rule_db", "protected databases, readable only")),
		rule("aws / cloud-cli", p.T("refused.rule_cloud", "cloud services and outside credentials")),
		rule("git push", p.T("refused.rule_push", "the branch belongs to the operator or the runner")),
		rule("git remote / config", p.T("refused.rule_remote", "the repository's own configuration")),
		rule("gh pr merge", p.T("refused.rule_merge", "merging and publishing pull requests")),
		"",
		"  "+theme.Paint(theme.Dim).Render(p.T("refused.policy_note", "a forbidden action fails on the spot and the model carries on")),
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
func (e Env) denialRows(d view.Entry, i int) ([]string, bool) {
	tool := d.Tool
	if tool == "" {
		tool = e.Words.T("refused.unnamed_tool", "command")
	}

	head := "    " + theme.Paint(theme.Bad).Render("✗") + " " + theme.Paint(theme.Accent).Render(tool) + ": "

	lead := 4 + 1 + 1 + lipgloss.Width(tool) + 2
	availW := max(20, max(e.Frame.Body.W, 1)-lead-lipgloss.Width(cells.FoldShut)-2)

	var body []string

	for _, l := range strings.Split(strings.TrimSpace(d.Text), "\n") {
		if l = strings.TrimSpace(l); l == "" {
			continue
		}

		for _, wl := range cells.Lines(l, availW) {
			body = append(body, cells.Fit(wl, availW))
		}
	}

	if len(body) == 0 {
		return []string{strings.TrimRight(head, ": ")}, false
	}

	if len(body) == 1 {
		return []string{head + strings.Repeat(" ", lipgloss.Width(cells.FoldShut)) + theme.Paint(theme.Bad).Render(body[0])}, false
	}

	open := e.row(i)
	mark := theme.Text(theme.Tertiary).Render(cells.Fold(open))

	if !open {
		return []string{head + mark + theme.Paint(theme.Bad).Render(body[0])}, true
	}

	out := []string{head + mark + theme.Paint(theme.Bad).Render(body[0])}

	indent := strings.Repeat(" ", lead+lipgloss.Width(cells.FoldShut))
	for _, l := range body[1:] {
		out = append(out, indent+theme.Paint(theme.Bad).Render(l))
	}

	return out, true
}

// rule is one line of the sandbox's standing refusals: the command, and what
// it is that Orbit will not let a run reach.
func rule(command, why string) string {
	return fmt.Sprintf("    %s %-24s %s",
		theme.Paint(theme.Dim).Render("✗"), command, theme.Paint(theme.Dim).Render(why))
}
