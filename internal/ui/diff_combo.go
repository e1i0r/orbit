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
		cardTop := topOf(title, p.T("diff.select_hint", "[▾ select / f]"), contentW)

		leftContent := fmt.Sprintf("%s %s  %s  %s%s", icon, theme.Paint(theme.Live).Bold(true).Render(curr.Path), stats, badge, collapseTag)
		navHint := theme.Paint(theme.Dim).Render(p.T("diff.select_nav", "(] next · [ prev · space fold)"))
		bodyText := cells.Spread(leftContent, navHint, innerW)
		fittedBody := ansi.Truncate(bodyText, innerW, "…")
		padRight := strings.Repeat(" ", max(0, innerW-lipgloss.Width(fittedBody)))

		cardBody := fmt.Sprintf("  │ %s%s │", fittedBody, padRight)
		cardBottom := fmt.Sprintf("  └%s┘", strings.Repeat("─", contentW+1))

		return heldTo([]string{cardTop, cardBody, cardBottom}, width)
	}

	title := fmt.Sprintf("📁 %s (%d)", p.T("diff.select_open_title", "Select File"), len(files))
	cardTop := topOf(title, p.T("diff.select_close_hint", "[▴ close / esc]"), contentW)

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

	return heldTo(lines, width)
}

// topOf is the box's top border: the title, some dashes, and the hint that
// says which key opens or closes the list.
//
// The dashes are counted by measuring what is already on the line rather
// than by adding up the pieces of the format string, because that sum was
// wrong and nothing could see it: the top came out one to three cells wider
// than the sides, so the border stuck out past the box on every window
// narrow enough for the two words to fill it.
//
// When there is no room for both, the hint is the half to drop. The title
// says which file is being looked at; the hint says which key opens the
// list, and a reader can find that in the cheat sheet.
func topOf(title, hint string, contentW int) string {
	const leastDashes = 2

	// The width of every line of the box, which is what the bottom border
	// below is built to.
	want := lipgloss.Width(fmt.Sprintf("  └%s┘", strings.Repeat("─", contentW+1)))

	shell := func(hint string, dashes int) string {
		return fmt.Sprintf("  ┌─ %s %s %s─┐",
			theme.Paint(theme.Accent).Bold(true).Render(title),
			theme.Paint(theme.Dim).Render(strings.Repeat("─", dashes)),
			theme.Paint(theme.Dim).Render(hint),
		)
	}

	for _, said := range []string{hint, ""} {
		bare := lipgloss.Width(shell(said, 0))
		if dashes := want - bare; dashes >= leastDashes {
			return shell(said, dashes)
		}
	}

	return shell("", leastDashes)
}

// heldTo keeps the box inside the window it is drawn in.
//
// contentW has a floor: a picker narrower than thirty columns is not one a
// reader can tell two paths apart in, so on a very narrow terminal the box
// is wider than the window on purpose. What must not follow is the box
// running off the right — a line wider than the terminal wraps where the
// terminal decides, which puts the bottom border a row below where every
// other reading of this view expects it. noteRows holds its own box the
// same way and for the same reason.
func heldTo(lines []string, width int) string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, cells.Fit(line, width))
	}

	return strings.Join(out, "\n")
}
