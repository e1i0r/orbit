package ui

import (
	"strings"

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

	summary := diffSummaryHeader(len(files), totalAdd, totalDel,
		p.T("diff.nav_help", "(] / [ next/prev file · f files · space collapse · r rationale · o editor)"))
	out = append(out, summary, "")

	fileIdx := 0

	for _, line := range raw {
		if strings.HasPrefix(line, "diff --git ") {
			if fileIdx < len(files) {
				f := files[fileIdx]
				if r, ok := rationales[f.Path]; ok && r != "" {
					f.Rationale = r
				}

				isCollapsed := collapsed != nil && collapsed[f.Path]
				cardTop := diffCardTop(f, fileIdx, len(files), width, p, isCollapsed)
				out = append(out, "", cardTop)
				files[fileIdx].StartLine = len(out) - 1

				if showRationale && f.Rationale != "" {
					out = append(out, diffRationaleLines(f.Rationale, width, p)...)
				}

				if isCollapsed {
					out = append(out, diffCardBottom(width))
				} else if showRationale && f.Rationale != "" {
					out = append(out, diffCardDivider(width))
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
			out = append(out, diffHunkLine(line))
			continue
		}

		role := diffRole(line)
		out = append(out, diffContentLines(line, role, width, wrapLines)...)
	}

	return out, files
}
