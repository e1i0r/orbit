package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// renderMarkdown renders a Markdown string into terminal-styled rows.
// If raw is true, it returns the lines formatted in plain text.
// If raw is false, it renders rich headings, code blocks, lists, quotes, and badges.
func renderMarkdown(text string, width int, raw bool) []string {
	if text == "" {
		return nil
	}

	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")

	if raw {
		var out []string
		for _, l := range lines {
			out = append(out, "    "+Paint(Dim).Render(quoteMark)+l)
		}

		return out
	}

	var out []string

	inCodeBlock := false

	for _, l := range lines {
		trimmed := strings.TrimSpace(l)

		// 1. Code block fence
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				lang := strings.TrimPrefix(trimmed, "```")

				badge := " [code"
				if lang != "" {
					badge += " · " + lang
				}

				badge += "] "
				out = append(out, "    "+Paint(Dim).Render("┌─"+badge+strings.Repeat("─", max(width-lipgloss.Width(badge)-10, 4))))
			} else {
				out = append(out, "    "+Paint(Dim).Render("└"+strings.Repeat("─", max(width-10, 4))))
			}

			continue
		}

		if inCodeBlock {
			out = append(out, "    "+Paint(Dim).Render("│ ")+Paint(Accent).Render(l))
			continue
		}

		// 2. Headings
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimPrefix(trimmed, "# ")
			out = append(out, "", "    "+Paint(Accent).Bold(true).Render("◆ "+title))

			continue
		}

		if strings.HasPrefix(trimmed, "## ") {
			title := strings.TrimPrefix(trimmed, "## ")
			out = append(out, "", "    "+Paint(Live).Bold(true).Render("▶ "+title))

			continue
		}

		if strings.HasPrefix(trimmed, "### ") {
			title := strings.TrimPrefix(trimmed, "### ")
			out = append(out, "    "+Paint(Accent).Render("● "+title))

			continue
		}

		// 3. Blockquotes
		if strings.HasPrefix(trimmed, "> ") {
			q := strings.TrimPrefix(trimmed, "> ")
			for _, wl := range splitIntoLines(formatInlineMarkdown(q), max(20, width-10)) {
				out = append(out, "    "+Paint(Dim).Render("▎ ")+Paint(Dim).Render(wl))
			}

			continue
		}

		// 4. Horizontal rules
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			out = append(out, "    "+Paint(Dim).Render(strings.Repeat("─", min(width-8, 50))))
			continue
		}

		// 5. Checklist items
		if strings.HasPrefix(trimmed, "- [x] ") || strings.HasPrefix(trimmed, "* [x] ") {
			item := trimmed[6:]
			for _, wl := range splitIntoLines(formatInlineMarkdown(item), max(20, width-10)) {
				out = append(out, "    "+Paint(OK).Render("✔ ")+wl)
			}

			continue
		}

		if strings.HasPrefix(trimmed, "- [ ] ") || strings.HasPrefix(trimmed, "* [ ] ") {
			item := trimmed[6:]
			for _, wl := range splitIntoLines(formatInlineMarkdown(item), max(20, width-10)) {
				out = append(out, "    "+Paint(Dim).Render("☐ ")+wl)
			}

			continue
		}

		// 6. Bullet lists
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			item := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* ")
			for _, wl := range splitIntoLines(formatInlineMarkdown(item), max(20, width-10)) {
				out = append(out, "    "+Paint(OK).Render("• ")+wl)
			}

			continue
		}

		// 7. Numbered lists (1. , 2. etc.)
		if len(trimmed) >= 3 && trimmed[0] >= '0' && trimmed[0] <= '9' && trimmed[1] == '.' && trimmed[2] == ' ' {
			num := trimmed[:2]

			item := trimmed[3:]
			for _, wl := range splitIntoLines(formatInlineMarkdown(item), max(20, width-10)) {
				out = append(out, "    "+Paint(Live).Render(num+" ")+wl)
			}

			continue
		}

		if trimmed == "" {
			out = append(out, "")
			continue
		}

		for _, wl := range splitIntoLines(formatInlineMarkdown(l), max(20, width-8)) {
			out = append(out, "    "+wl)
		}
	}

	return out
}

// formatInlineMarkdown formats bold (**text**), code (`text`), italic (*text*).
func formatInlineMarkdown(s string) string {
	res := s
	// Bold **text**
	for {
		start := strings.Index(res, "**")
		if start == -1 {
			break
		}

		end := strings.Index(res[start+2:], "**")
		if end == -1 {
			break
		}

		end += start + 2
		boldText := res[start+2 : end]
		styled := Paint(Accent).Bold(true).Render(boldText)
		res = res[:start] + styled + res[end+2:]
	}
	// Inline code `text`
	for {
		start := strings.Index(res, "`")
		if start == -1 {
			break
		}

		end := strings.Index(res[start+1:], "`")
		if end == -1 {
			break
		}

		end += start + 1
		code := res[start+1 : end]
		styled := Paint(Accent).Render(code)
		res = res[:start] + styled + res[end+1:]
	}

	return res
}
