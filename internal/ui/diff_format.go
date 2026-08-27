package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/words"
)

// formatStructuredDiff renders a rich, scalable diff view with file cards, hunk tags, LLM rationale, and collapse states.
func formatStructuredDiff(diffText string, width int, p *words.Printer, rationales map[string]string, showRationale bool, collapsed map[string]bool, wrapLines bool) ([]string, []diffFile) {
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
	out := make([]string, 0, len(raw)+len(files)*6)

	// Sticky summary header
	summary := fmt.Sprintf(" %s %s %s  %s",
		Paint(Accent).Bold(true).Render(fmt.Sprintf("%d files changed", len(files))),
		Paint(OK).Render(fmt.Sprintf("+%d", totalAdd)),
		Paint(Bad).Render(fmt.Sprintf("-%d", totalDel)),
		Paint(Dim).Render(p.T("diff.nav_help", "(] / [ next/prev file · f files · space collapse · r rationale · o editor)")),
	)
	out = append(out, summary, "")

	fileIdx := 0
	for i, line := range raw {
		if strings.HasPrefix(line, "diff --git ") {
			if fileIdx < len(files) {
				f := files[fileIdx]
				if r, ok := rationales[f.Path]; ok && r != "" {
					f.Rationale = r
				}
				icon := fileIcon(f.Path)
				badge := formatFileBadge(f.Status)
				stats := fmt.Sprintf("%s %s", Paint(OK).Render(fmt.Sprintf("+%d", f.Added)), Paint(Bad).Render(fmt.Sprintf("-%d", f.Deleted)))
				pos := fmt.Sprintf("[%d/%d]", fileIdx+1, len(files))

				isCollapsed := collapsed != nil && collapsed[f.Path]
				collapsedTag := ""
				if isCollapsed {
					collapsedTag = "  " + Paint(Warn).Render("["+p.T("diff.collapsed_tag", "collapsed")+" · space]")
				}

				headerTitle := fmt.Sprintf("%s %s %s  %s%s", icon, Paint(Accent).Bold(true).Render(f.Path), stats, badge, collapsedTag)
				borderWidth := max(2, width-lipgloss.Width(headerTitle)-lipgloss.Width(pos)-8)
				border := strings.Repeat("─", borderWidth)

				cardTop := fmt.Sprintf("  ┌── %s %s %s ──┐", headerTitle, Paint(Dim).Render(border), Paint(Dim).Render(pos))
				out = append(out, "", cardTop)
				files[fileIdx].StartLine = len(out) - 1

				if showRationale && f.Rationale != "" {
					label := Paint(Warn).Bold(true).Render("💡 " + p.T("diff.rationale_label", "LLM Decision") + ":")
					wrapped := splitIntoLines(f.Rationale, max(20, width-lipgloss.Width(label)-10))
					for count, wl := range wrapped {
						if count == 0 {
							out = append(out, fmt.Sprintf("  │ %s %s", label, Paint(Dim).Render(wl)))
						} else {
							pad := strings.Repeat(" ", lipgloss.Width(label)+1)
							out = append(out, fmt.Sprintf("  │ %s%s", pad, Paint(Dim).Render(wl)))
						}
					}
				}

				if isCollapsed {
					cardBottom := fmt.Sprintf("  └──%s┘", strings.Repeat("─", max(2, width-6)))
					out = append(out, cardBottom)
				} else if showRationale && f.Rationale != "" {
					divider := strings.Repeat("─", max(2, width-6))
					out = append(out, "  ├"+Paint(Dim).Render(divider))
				}

				fileIdx++
			}
			continue
		}

		if fileIdx > 0 && collapsed != nil && collapsed[files[fileIdx-1].Path] {
			continue
		}

		if strings.HasPrefix(line, "index ") || strings.HasPrefix(line, "new file ") ||
			strings.HasPrefix(line, "deleted file ") || strings.HasPrefix(line, "--- ") ||
			strings.HasPrefix(line, "+++ ") {
			continue
		}

		if strings.HasPrefix(line, "@@") {
			formattedHunk := formatHunkHeader(line)
			out = append(out, "  "+Paint(Accent).Bold(true).Render("│"+formattedHunk))
			continue
		}

		// Line rendering
		role := diffRole(line)
		if !wrapLines || lipgloss.Width(line)+6 <= width {
			out = append(out, "  │ "+Paint(role).Render(line))
		} else {
			availW := max(20, width-8)
			wrapped := splitIntoLines(line, availW)
			for idx, wl := range wrapped {
				if idx == 0 {
					out = append(out, "  │ "+Paint(role).Render(wl))
				} else {
					out = append(out, "  │   "+Paint(role).Render(wl))
				}
			}
		}
		_ = i
	}

	return out, files
}
