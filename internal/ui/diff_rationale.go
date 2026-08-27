package ui

import (
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// extractFileRationales analyzes task entries and reports to extract the LLM's
// decisions and rationale for each modified file.
func extractFileRationales(entries []view.Entry, files []diffFile, p *words.Printer) map[string]string {
	rationales := make(map[string]string)
	if len(files) == 0 {
		return rationales
	}

	// 1. Scan entries (reports, tool calls, thinking) for per-file bullet points and reasons
	for _, e := range entries {
		switch e.What() {
		case view.EntryFinished, view.EntryToolCall, view.EntryThought:
			lines := strings.Split(e.Text, "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" {
					continue
				}
				for _, f := range files {
					if rationales[f.Path] != "" {
						continue // Already found a specific description
					}
					if strings.Contains(trimmed, f.Path) || strings.Contains(trimmed, filepath.Base(f.Path)) {
						if reason := extractReasonFromLine(trimmed); reason != "" {
							rationales[f.Path] = reason
						}
					}
				}
			}
		}
	}

	// 2. Fallback rationale based on file status and extension
	for _, f := range files {
		if rationales[f.Path] == "" {
			rationales[f.Path] = fallbackRationale(f, p)
		}
	}

	return rationales
}

func extractReasonFromLine(line string) string {
	line = strings.TrimPrefix(line, "•")
	line = strings.TrimPrefix(line, "-")
	line = strings.TrimPrefix(line, "*")
	line = strings.TrimSpace(line)

	for _, sep := range []string{"—", " - ", " : ", "=>", ": "} {
		parts := strings.SplitN(line, sep, 2)
		if len(parts) == 2 {
			after := strings.TrimSpace(parts[1])
			if lipgloss.Width(after) > 10 {
				return cleanRationale(after)
			}
		}
	}

	w := lipgloss.Width(line)
	if w > 15 && w < 160 {
		return cleanRationale(line)
	}
	return ""
}

func cleanRationale(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Trim(text, "`\"'")
	if lipgloss.Width(text) > 180 {
		text = ansi.Truncate(text, 180, "…")
	}
	return text
}

func fallbackRationale(f diffFile, p *words.Printer) string {
	base := filepath.Base(f.Path)
	switch {
	case strings.HasSuffix(base, "_test.go") || strings.Contains(base, ".test."):
		return p.T("diff.rationale_tests", "unit and property testing suite verifying behavior and boundary invariants")
	case f.Status == "NEW":
		return p.T("diff.rationale_new", "new implementation file created to fulfill task requirements")
	case f.Status == "DEL":
		return p.T("diff.rationale_del", "removed deprecated or redundant file")
	default:
		return p.T("diff.rationale_mod", "updated logic and signatures to fulfill task requirements")
	}
}
