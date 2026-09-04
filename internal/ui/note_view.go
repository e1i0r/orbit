package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) noteRows(h, w int) []string {
	if h <= 0 || w <= 0 {
		return nil
	}

	boxW := min(w-4, 76)
	if boxW < 30 {
		boxW = 30
	}

	innerW := boxW - 4

	said := m.boxWords()
	title := said.title
	prompt := said.prompt + ": "

	raw := m.note.text

	var contentLines []string
	if raw == "" {
		contentLines = []string{Paint(Dim).Render(said.placeholder)}
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

	actions := said.actions
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

// boxWords is what the message box calls itself, which follows the command
// the typing will be handed to: a note is left for the next phase to read,
// and a directive stops the run that is going.
//
// Both sets are written out here rather than assembled from the verb. The
// translation gate counts a sentence it can read at the call site, and a key
// built out of a variable is a sentence nobody counted — it would go
// untranslated without anything saying so.
type boxWords struct{ title, prompt, placeholder, actions string }

func (m Model) boxWords() boxWords {
	p := m.opts.Words
	id := m.note.taskID

	if m.note.verb == verbDirect {
		return boxWords{
			title:       p.T("direct.dialog_title", "redirect {id}", about("id", id)),
			prompt:      p.T("direct.prompt", "directive"),
			placeholder: p.T("direct.placeholder", "what to do instead — this stops the run in flight..."),
			actions:     p.T("direct.actions", "↵ send directive · esc cancel · ^V paste"),
		}
	}

	return boxWords{
		title:       p.T("note.dialog_title", "leave a note for {id}", about("id", id)),
		prompt:      p.T("note.prompt", "note"),
		placeholder: p.T("note.placeholder", "instructions or context for the next phase..."),
		actions:     p.T("note.actions", "↵ save note · esc cancel · ^V paste"),
	}
}
