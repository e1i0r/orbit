package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/patch"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/words"
)

// renderDiffFileSelect renders an HTML-like dropdown / select component for changed files.
func renderDiffFileSelect(files []patch.File, activeIdx int, width int, p *words.Printer, collapsed map[string]bool, isOpen bool, cursorIdx int) string {
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
		icon := patch.Icon(curr.Path)
		badge := patch.Badge(curr.Status)
		stats := fmt.Sprintf("%s %s", theme.Paint(theme.OK).Render(fmt.Sprintf("+%d", curr.Added)), theme.Paint(theme.Bad).Render(fmt.Sprintf("-%d", curr.Deleted)))

		collapseTag := ""
		if collapsed != nil && collapsed[curr.Path] {
			collapseTag = theme.Paint(theme.Warn).Render(" *" + p.T("diff.collapsed_tag", "collapsed"))
		}

		title := fmt.Sprintf("📁 %s [%d/%d]", p.T("diff.select_title", "File"), activeIdx+1, len(files))
		actionHint := p.T("diff.select_hint", "[▾ select / f]")
		titleW := lipgloss.Width(title)
		hintW := lipgloss.Width(actionHint)

		borderW := max(2, contentW-titleW-hintW-4)
		topBorder := strings.Repeat("─", borderW)

		cardTop := fmt.Sprintf("  ┌─ %s %s %s─┐",
			theme.Paint(theme.Accent).Bold(true).Render(title),
			theme.Paint(theme.Dim).Render(topBorder),
			theme.Paint(theme.Dim).Render(actionHint),
		)

		leftContent := fmt.Sprintf("%s %s  %s  %s%s", icon, theme.Paint(theme.Live).Bold(true).Render(curr.Path), stats, badge, collapseTag)
		navHint := theme.Paint(theme.Dim).Render(p.T("diff.select_nav", "(] next · [ prev · space fold)"))
		bodyText := cells.Spread(leftContent, navHint, innerW)
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
		theme.Paint(theme.Accent).Bold(true).Render(title),
		theme.Paint(theme.Dim).Render(topBorder),
		theme.Paint(theme.Dim).Render(closeHint),
	)

	var lines []string

	lines = append(lines, cardTop)

	maxItems := 7

	start := 0
	if cursorIdx >= maxItems {
		start = cursorIdx - maxItems + 1
	}

	end := min(len(files), start+maxItems)

	// The rail down the right edge, where every other list in this window
	// has one. Nineteen files in a box that shows seven said nothing about
	// the other twelve: the reader learned they were there by holding the
	// arrow key down.
	track := cells.Track(maxItems, len(files), start)

	for i := start; i < end; i++ {
		f := files[i]
		isSel := i == cursorIdx

		cursor := "   "
		if isSel {
			cursor = theme.Paint(theme.Live).Bold(true).Render(" ▸ ")
		}

		icon := patch.Icon(f.Path)
		badge := patch.Badge(f.Status)
		stats := fmt.Sprintf("%s %s", theme.Paint(theme.OK).Render(fmt.Sprintf("+%d", f.Added)), theme.Paint(theme.Bad).Render(fmt.Sprintf("-%d", f.Deleted)))

		colTag := ""
		if collapsed != nil && collapsed[f.Path] {
			colTag = theme.Paint(theme.Warn).Render(" *" + p.T("diff.collapsed_tag", "collapsed"))
		}

		num := fmt.Sprintf("%2d.", i+1)

		var name string
		if isSel {
			name = theme.Paint(theme.Live).Bold(true).Render(f.Path)
		} else {
			name = theme.Paint(theme.Accent).Render(f.Path)
		}

		itemText := fmt.Sprintf("%s%s %s %s  %s  %s%s", cursor, theme.Paint(theme.Dim).Render(num), icon, name, stats, badge, colTag)
		fittedItem := ansi.Truncate(itemText, innerW, "…")
		pad := strings.Repeat(" ", max(0, innerW-lipgloss.Width(fittedItem)))

		edge := "│"
		if track != nil {
			edge = track[i-start]
		}

		lines = append(lines, fmt.Sprintf("  │ %s%s %s", fittedItem, pad, edge))
	}

	helpText := theme.Paint(theme.Dim).Render(p.T("diff.select_help", "  [↑↓] navigate  [⏎ / click] select  [space] fold  [esc] close"))
	fittedHelp := ansi.Truncate(helpText, innerW, "…")
	padHelp := strings.Repeat(" ", max(0, innerW-lipgloss.Width(fittedHelp)))
	lines = append(lines, fmt.Sprintf("  │ %s%s │", fittedHelp, padHelp))
	lines = append(lines, fmt.Sprintf("  └%s┘", strings.Repeat("─", contentW+1)))

	return strings.Join(lines, "\n")
}
