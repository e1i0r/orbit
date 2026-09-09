package panes

import (
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/ui/patch"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/words"
)

// How much of a diff is drawn.
//
// Every line drawn here is a line styled by lipgloss, and the window redraws
// twice a second and again on every key. A task whose worktree held a
// 126,000-line reformat — three generated files a codegen step had rewritten
// end to end — took 303ms per draw on an M1 Pro, which is a cockpit that
// does not answer the keyboard (#133). Generated files are large and common:
// API docs, protobuf, mocks, lockfiles, snapshots. A big diff is an expected
// input, not an edge case.
//
// So a file past mostLinesForOneFile keeps its card and loses its hunks, and
// once mostLinesDrawn lines are spoken for the rest keep their cards too.
// What a reader loses is the text of a file no one can read on a terminal
// anyway; what they keep is the list, the counts, and a window that moves.
const (
	mostLinesForOneFile = 2000
	mostLinesDrawn      = 20000
)

// undrawn is the files whose hunks are left out, by path, and how many lines
// each of them would have cost.
//
// In the order git wrote them, so that what a reader sees drawn is the front
// of their diff rather than whichever files happened to be small.
func undrawn(files []patch.File) map[string]int {
	left := mostLinesDrawn
	out := make(map[string]int)

	for _, f := range files {
		n := f.EndLine - f.StartLine + 1
		if n > mostLinesForOneFile || n > left {
			out[f.Path] = n

			continue
		}

		left -= n
	}

	return out
}

// formatStructuredDiff renders a rich, scalable diff view with file cards, hunk tags, LLM rationale, and collapse states.
func formatStructuredDiff(diffText string, width int, p *words.Printer, rationales map[string]string, showRationale bool, collapsed map[string]bool, wrapLines bool) ([]string, []patch.File) {
	text := strings.TrimSuffix(diffText, "\n")
	if strings.TrimSpace(text) == "" {
		return []string{" " + theme.Paint(theme.Dim).Render(p.T("diff.unchanged", "no changes in this task's worktree"))}, nil
	}

	raw := strings.Split(text, "\n")

	files := patch.Files(raw)
	if len(files) == 0 {
		out := make([]string, 0, len(raw))
		for _, line := range raw {
			out = append(out, " "+theme.Paint(diffRole(line)).Render(line))
		}

		return out, files
	}

	big := undrawn(files)
	totalAdd, totalDel := patch.Stats(files)
	out := make([]string, 0, len(raw)+len(files)*6)

	summary := diffSummaryHeader(len(files), totalAdd, totalDel,
		p.T("diff.nav_help", "(] / [ next/prev file · f files · space collapse · r rationale · o editor)"), p)
	out = append(out, summary, "")

	fileIdx := 0

	for _, line := range raw {
		if strings.HasPrefix(line, "diff --git ") {
			if fileIdx < len(files) {
				f := files[fileIdx]
				if r, ok := rationales[f.Path]; ok && r != "" {
					f.Rationale = r
				}

				// A file too big to draw is drawn as a collapsed one: the
				// card, the counts, and no hunks under it.
				isCollapsed := big[f.Path] > 0 || (collapsed != nil && collapsed[f.Path])

				cardTop := diffCardTop(f, fileIdx, len(files), width, p, isCollapsed)
				out = append(out, "", cardTop)
				files[fileIdx].StartLine = len(out) - 1

				if showRationale && f.Rationale != "" {
					out = append(out, diffRationaleLines(f.Rationale, width, p)...)
				}

				if n := big[f.Path]; n > 0 {
					out = append(out, " "+theme.Paint(theme.Dim).Render(p.T("diff.too_large",
						"{n} lines not drawn — a file this size stops the window; open it with [o]",
						words.Arg{Name: "n", Value: strconv.Itoa(n)})))
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

		if fileIdx > 0 && big[files[fileIdx-1].Path] > 0 {
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
