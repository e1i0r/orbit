package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

type noteItem struct {
	at      string
	sender  string
	role    Role
	status  string
	content []string
}

// notesLines renders Pane 9: Full operator dialogue, notes, and interactive CLI history.
func (m Model) notesLines() []string {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{"  " + Paint(Bad).Render(m.logErr.Error())}
	}

	var items []noteItem
	noteIndex := 0

	for _, e := range m.entries {
		timeStr := ""
		if !e.At.IsZero() {
			timeStr = e.At.Format("15:04:05")
		}

		switch e.What() {
		case view.EntryNoted:
			noteIndex++
			statusNote := "read by run"
			if e.Attempt > 0 {
				statusNote = fmt.Sprintf("read by run %d", e.Attempt)
			}
			senderLabel := fmt.Sprintf("● %d  %s", noteIndex, p.T("notes.operator", "OPERADOR"))
			content := renderMarkdown(e.Text, m.frame.Body.W, m.rawText)
			items = append(items, noteItem{
				at:      timeStr,
				sender:  senderLabel,
				role:    Accent,
				status:  statusNote,
				content: content,
			})

		case view.EntryWaiting:
			if e.Cause != "" || e.Text != "" {
				msg := e.Cause
				if msg == "" {
					msg = e.Text
				}
				items = append(items, noteItem{
					at:      timeStr,
					sender:  fmt.Sprintf("🤖 %s", p.T("notes.llm_prompt", "MODELO (consulta al operador)")),
					role:    Warn,
					status:  e.Phase,
					content: []string{"? " + msg},
				})
			}
		}
	}

	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(p.T("notes.title", "Operator Notes & LLM Dialogue")),
		"  " + Paint(Dim).Render(p.T("notes.subtitle", "everything spoken with the model, notes filed and interactive sessions")),
		"",
	}

	if len(items) == 0 {
		out = append(out,
			"  "+Paint(Dim).Render(p.T("notes.empty", "no notes or dialogue recorded for this task")),
			"",
			"  "+Paint(Dim).Render(p.T("notes.hint_action", "pulsa 'a' para dejar una nota · pulsa 'c' para abrir la CLI interactiva")),
		)
		return out
	}

	out = append(out, fmt.Sprintf("  %d %s · %s",
		len(items),
		p.T("notes.count", "entradas en el diálogo"),
		Paint(OK).Render(p.T("notes.all_filed", "sincronizado con el modelo")),
	))
	out = append(out, "")

	for _, item := range items {
		header := fmt.Sprintf("  %s  %s  %s",
			Paint(item.role).Render(item.sender),
			Paint(Dim).Render(item.at),
			Paint(Dim).Render(item.status),
		)
		out = append(out, header)

		for _, l := range item.content {
			switch {
			case strings.HasPrefix(l, "?"):
				out = append(out, "      "+Paint(Warn).Render(l))
			case strings.HasPrefix(l, "→"):
				out = append(out, "      "+Paint(OK).Render(l))
			case strings.HasPrefix(l, "[cli]"):
				out = append(out, "      "+Paint(Live).Render(l))
			default:
				out = append(out, "      "+l)
			}
		}
		out = append(out, "")
	}

	out = append(out,
		"  "+Paint(Dim).Render(p.T("notes.hint_footer", "pulsa 'a' para agregar una nota · pulsa 'c' o 't' para entrar a la CLI interactiva")),
		"",
	)

	return out
}
