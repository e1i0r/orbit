package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/e1i0r/orbit/internal/words"
)

// renderDiffFileSelect renders an HTML-like dropdown / select component for changed files.
func renderDiffFileSelect(files []diffFile, activeIdx int, width int, p *words.Printer, collapsed map[string]bool, isOpen bool, cursorIdx int) string {
	if len(files) == 0 {
		return ""
	}

	contentW := max(30, width-8)
	innerW := contentW - 1

	if !isOpen {
		if activeIdx < 0 || activeIdx >= len(files) {
			activeIdx = 0
		}

		curr := files[activeIdx]
		icon := fileIcon(curr.Path)
		badge := formatFileBadge(curr.Status)
		stats := fmt.Sprintf("%s %s", Paint(OK).Render(fmt.Sprintf("+%d", curr.Added)), Paint(Bad).Render(fmt.Sprintf("-%d", curr.Deleted)))

		collapseTag := ""
		if collapsed != nil && collapsed[curr.Path] {
			collapseTag = Paint(Warn).Render(" *" + p.T("diff.collapsed_tag", "collapsed"))
		}

		title := fmt.Sprintf("📁 %s [%d/%d]", p.T("diff.select_title", "File"), activeIdx+1, len(files))
		actionHint := p.T("diff.select_hint", "[▾ select / f]")
		titleW := lipgloss.Width(title)
		hintW := lipgloss.Width(actionHint)

		borderW := max(2, contentW-titleW-hintW-4)
		topBorder := strings.Repeat("─", borderW)

		cardTop := fmt.Sprintf("  ┌─ %s %s %s─┐",
			Paint(Accent).Bold(true).Render(title),
			Paint(Dim).Render(topBorder),
			Paint(Dim).Render(actionHint),
		)

		leftContent := fmt.Sprintf("%s %s  %s  %s%s", icon, Paint(Live).Bold(true).Render(curr.Path), stats, badge, collapseTag)
		navHint := Paint(Dim).Render(p.T("diff.select_nav", "(] next · [ prev · space fold)"))
		bodyText := spread(leftContent, navHint, innerW)
		fittedBody := ansi.Truncate(bodyText, innerW, "…")
		padRight := strings.Repeat(" ", max(0, innerW-lipgloss.Width(fittedBody)))

		cardBody := fmt.Sprintf("  │ %s%s │", fittedBody, padRight)
		cardBottom := fmt.Sprintf("  └%s┘", strings.Repeat("─", contentW+1))

		return fmt.Sprintf("%s\n%s\n%s", cardTop, cardBody, cardBottom)
	}

	title := fmt.Sprintf("📁 %s (%d)", p.T("diff.select_open_title", "Select File"), len(files))
	closeHint := p.T("diff.select_close_hint", "[▴ close / esc]")
	titleW := lipgloss.Width(title)
	hintW := lipgloss.Width(closeHint)

	borderW := max(2, contentW-titleW-hintW-4)
	topBorder := strings.Repeat("─", borderW)

	cardTop := fmt.Sprintf("  ┌─ %s %s %s─┐",
		Paint(Accent).Bold(true).Render(title),
		Paint(Dim).Render(topBorder),
		Paint(Dim).Render(closeHint),
	)

	var lines []string

	lines = append(lines, cardTop)

	maxItems := 7

	start := 0
	if cursorIdx >= maxItems {
		start = cursorIdx - maxItems + 1
	}

	end := min(len(files), start+maxItems)

	for i := start; i < end; i++ {
		f := files[i]
		isSel := i == cursorIdx

		cursor := "   "
		if isSel {
			cursor = Paint(Live).Bold(true).Render(" ▸ ")
		}

		icon := fileIcon(f.Path)
		badge := formatFileBadge(f.Status)
		stats := fmt.Sprintf("%s %s", Paint(OK).Render(fmt.Sprintf("+%d", f.Added)), Paint(Bad).Render(fmt.Sprintf("-%d", f.Deleted)))

		colTag := ""
		if collapsed != nil && collapsed[f.Path] {
			colTag = Paint(Warn).Render(" *" + p.T("diff.collapsed_tag", "collapsed"))
		}

		num := fmt.Sprintf("%2d.", i+1)

		var name string
		if isSel {
			name = Paint(Live).Bold(true).Render(f.Path)
		} else {
			name = Paint(Accent).Render(f.Path)
		}

		itemText := fmt.Sprintf("%s%s %s %s  %s  %s%s", cursor, Paint(Dim).Render(num), icon, name, stats, badge, colTag)
		fittedItem := ansi.Truncate(itemText, innerW, "…")
		pad := strings.Repeat(" ", max(0, innerW-lipgloss.Width(fittedItem)))
		lines = append(lines, fmt.Sprintf("  │ %s%s │", fittedItem, pad))
	}

	helpText := Paint(Dim).Render(p.T("diff.select_help", "  [↑↓] navigate  [⏎ / click] select  [space] fold  [esc] close"))
	fittedHelp := ansi.Truncate(helpText, innerW, "…")
	padHelp := strings.Repeat(" ", max(0, innerW-lipgloss.Width(fittedHelp)))
	lines = append(lines, fmt.Sprintf("  │ %s%s │", fittedHelp, padHelp))
	lines = append(lines, fmt.Sprintf("  └%s┘", strings.Repeat("─", contentW+1)))

	return strings.Join(lines, "\n")
}
