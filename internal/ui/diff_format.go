package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/words"
)

// formatStructuredDiff renders a rich, scalable diff view with file cards and hunk tags.
func formatStructuredDiff(diffText string, width int, p *words.Printer) ([]string, []diffFile) {
	text := strings.TrimSuffix(diffText, "\n")
	if strings.TrimSpace(text) == "" {
		return []string{" " + Paint(Dim).Render(p.T("diff.unchanged", "no changes in this task's worktree"))}, nil
	}

	raw := strings.Split(text, "\n")
	files := parseDiffFiles(raw)
	if len(files) == 0 {
		out := make([]string, 0, len(raw))
		for _, line := range raw {
			out = append(out, " "+Paint(diffRole(line)).Render(line))
		}
		return out, files
	}

	totalAdd, totalDel := diffStats(files)
	out := make([]string, 0, len(raw)+len(files)*4)

	// Sticky summary header
	summary := fmt.Sprintf(" %s %s %s  %s",
		Paint(Accent).Bold(true).Render(fmt.Sprintf("%d files changed", len(files))),
		Paint(OK).Render(fmt.Sprintf("+%d", totalAdd)),
		Paint(Bad).Render(fmt.Sprintf("-%d", totalDel)),
		Paint(Dim).Render(p.T("diff.nav_help", "(] / [ next/prev file · n / N next/prev hunk · o editor)")),
	)
	out = append(out, summary, "")

	fileIdx := 0
	for i, line := range raw {
		if strings.HasPrefix(line, "diff --git ") {
			if fileIdx < len(files) {
				f := files[fileIdx]
				icon := fileIcon(f.Path)
				badge := formatFileBadge(f.Status)
				stats := fmt.Sprintf("%s %s", Paint(OK).Render(fmt.Sprintf("+%d", f.Added)), Paint(Bad).Render(fmt.Sprintf("-%d", f.Deleted)))
				pos := fmt.Sprintf("[%d/%d]", fileIdx+1, len(files))

				headerTitle := fmt.Sprintf("%s %s %s  %s", icon, Paint(Accent).Bold(true).Render(f.Path), stats, badge)
				border := strings.Repeat("─", max(2, width-lipgloss.Width(headerTitle)-lipgloss.Width(pos)-8))

				cardTop := fmt.Sprintf("  ┌── %s %s %s ──┐", headerTitle, Paint(Dim).Render(border), Paint(Dim).Render(pos))
				out = append(out, "", cardTop)
				files[fileIdx].StartLine = len(out) - 1
				fileIdx++
			}
			continue
		}

		if strings.HasPrefix(line, "index ") || strings.HasPrefix(line, "new file ") ||
			strings.HasPrefix(line, "deleted file ") || strings.HasPrefix(line, "--- ") ||
			strings.HasPrefix(line, "+++ ") {
			// Skip redundant raw diff header lines inside card
			continue
		}

		if strings.HasPrefix(line, "@@") {
			formattedHunk := formatHunkHeader(line)
			out = append(out, "  "+Paint(Accent).Bold(true).Render("│"+formattedHunk))
			continue
		}

		// Line rendering
		role := diffRole(line)
		out = append(out, "  │ "+Paint(role).Render(line))
		_ = i
	}

	return out, files
}
