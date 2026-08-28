package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) noteRows(h, w int) []string {
	if h <= 0 || w <= 0 {
		return nil
	}

	p := m.opts.Words

	boxW := min(w-4, 76)
	if boxW < 30 {
		boxW = 30
	}

	innerW := boxW - 4

	title := p.T("note.dialog_title", "leave a note for {id}", about("id", m.note.taskID))
	prompt := p.T("note.prompt", "note") + ": "

	raw := m.note.text

	var contentLines []string
	if raw == "" {
		contentLines = []string{Paint(Dim).Render(p.T("note.placeholder", "instructions or context for the next phase..."))}
	} else {
		for _, part := range strings.Split(raw, "\n") {
			if part == "" {
				contentLines = append(contentLines, "")
			} else {
				contentLines = append(contentLines, splitIntoLines(part, innerW)...)
			}
		}
	}

	for len(contentLines) < 3 {
		contentLines = append(contentLines, "")
	}

	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#38BDF8"))
	headerBorder := "┌─ " + Paint(Accent).Bold(true).Render(title) + " "

	remWidth := boxW - lipgloss.Width(headerBorder) - 1
	if remWidth < 0 {
		remWidth = 0
	}

	top := "  " + borderStyle.Render(headerBorder+strings.Repeat("─", remWidth)+"┐")

	var out []string

	out = append(out, fit(top, w))

	for i, l := range contentLines {
		lineContent := l
		if i == 0 && raw != "" {
			lineContent = Paint(Dim).Render(prompt) + lineContent
		}

		if i == len(contentLines)-1 {
			lineContent += Paint(Sel).Render(" ")
		}

		wLine := lipgloss.Width(lineContent)

		pad := innerW - wLine
		if pad < 0 {
			pad = 0
		}

		row := "  " + borderStyle.Render("│ ") + lineContent + strings.Repeat(" ", pad) + borderStyle.Render(" │")
		out = append(out, fit(row, w))
	}

	actions := p.T("note.actions", "↵ save note · esc cancel · ^V paste")
	actionLine := "  " + Paint(Dim).Render(actions)
	wAct := lipgloss.Width(actionLine)

	padAct := innerW - wAct
	if padAct < 0 {
		padAct = 0
	}

	out = append(out, fit("  "+borderStyle.Render("│ ")+actionLine+strings.Repeat(" ", padAct)+borderStyle.Render(" │"), w))
	out = append(out, fit("  "+borderStyle.Render("└"+strings.Repeat("─", boxW-2)+"┘"), w))

	return fill(out, h)
}
