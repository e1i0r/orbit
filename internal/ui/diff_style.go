package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/e1i0r/orbit/internal/words"
)

// diffSummaryHeader builds the top summary line showing change counts and key hints.
func diffSummaryHeader(count, totalAdd, totalDel int, navHelp string) string {
	return fmt.Sprintf(" %s %s %s  %s",
		Paint(Accent).Bold(true).Render(fmt.Sprintf("%d files changed", count)),
		Paint(OK).Render(fmt.Sprintf("+%d", totalAdd)),
		Paint(Bad).Render(fmt.Sprintf("-%d", totalDel)),
		Paint(Dim).Render(navHelp),
	)
}

// diffCardTop renders the top card boundary and metadata for a single file.
func diffCardTop(f diffFile, idx, total, width int, p *words.Printer, isCollapsed bool) string {
	icon := fileIcon(f.Path)
	badge := formatFileBadge(f.Status)
	stats := fmt.Sprintf("%s %s",
		Paint(OK).Render(fmt.Sprintf("+%d", f.Added)),
		Paint(Bad).Render(fmt.Sprintf("-%d", f.Deleted)))
	pos := fmt.Sprintf("[%d/%d]", idx+1, total)

	collapsedTag := ""
	if isCollapsed {
		collapsedTag = "  " + Paint(Warn).Render("["+p.T("diff.collapsed_tag", "collapsed")+" · space]")
	}

	headerTitle := fmt.Sprintf("%s %s %s  %s%s", icon, Paint(Accent).Bold(true).Render(f.Path), stats, badge, collapsedTag)
	borderWidth := max(2, width-lipgloss.Width(headerTitle)-lipgloss.Width(pos)-8)
	border := strings.Repeat("─", borderWidth)

	return fmt.Sprintf("  ┌── %s %s %s ──┐", headerTitle, Paint(Dim).Render(border), Paint(Dim).Render(pos))
}

// diffRationaleLines renders the LLM decision box inside a file card.
func diffRationaleLines(rationale string, width int, p *words.Printer) []string {
	if rationale == "" {
		return nil
	}
	label := Paint(Warn).Bold(true).Render("💡 " + p.T("diff.rationale_label", "LLM Decision") + ":")
	wrapped := splitIntoLines(rationale, max(20, width-lipgloss.Width(label)-10))
	var out []string
	for count, wl := range wrapped {
		if count == 0 {
			out = append(out, fmt.Sprintf("  │ %s %s", label, Paint(Dim).Render(wl)))
		} else {
			pad := strings.Repeat(" ", lipgloss.Width(label)+1)
			out = append(out, fmt.Sprintf("  │ %s%s", pad, Paint(Dim).Render(wl)))
		}
	}
	return out
}

// diffCardBottom renders the bottom boundary for a collapsed file card.
func diffCardBottom(width int) string {
	return fmt.Sprintf("  └──%s┘", strings.Repeat("─", max(2, width-6)))
}

// diffCardDivider renders the horizontal rule separating rationale from file diff content.
func diffCardDivider(width int) string {
	return "  ├" + Paint(Dim).Render(strings.Repeat("─", max(2, width-6)))
}

// diffHunkLine renders the styled hunk header with branch line.
func diffHunkLine(line string) string {
	return "  " + Paint(Accent).Bold(true).Render("│"+formatHunkHeader(line))
}

// diffContentLines renders a diff content line, wrapping across rows if wrapLines is true.
func diffContentLines(line string, role Role, width int, wrapLines bool) []string {
	if !wrapLines || lipgloss.Width(line)+6 <= width {
		return []string{"  │ " + Paint(role).Render(line)}
	}
	availW := max(20, width-8)
	wrapped := splitIntoLines(line, availW)
	var out []string
	for idx, wl := range wrapped {
		if idx == 0 {
			out = append(out, "  │ "+Paint(role).Render(wl))
		} else {
			out = append(out, "  │   "+Paint(role).Render(wl))
		}
	}
	return out
}
