package ui

import (
	"fmt"
	"strconv"
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

			statusNote := p.T("notes.read_by_run", "read by the run")
			if e.Attempt > 0 {
				statusNote = p.T("notes.read_by_run_n", "read by run {n}", about("n", strconv.Itoa(e.Attempt)))
			}

			senderLabel := fmt.Sprintf("● %d  %s", noteIndex, p.T("notes.operator", "OPERATOR"))
			content := renderMarkdown(e.Text, m.frame.Body.W, m.rawText)
			items = append(items, noteItem{
				at:      timeStr,
				sender:  senderLabel,
				role:    Accent,
				status:  statusNote,
				content: content,
			})

		case view.EntryDialogue:
			// Beside the notes and not among them. This is what a model or
			// a session did to the task, which is the other half of the
			// dialogue the reader came to this tab for — and the half no
			// phase is ever handed, so it is never mistaken for a note the
			// next run will read.
			who := e.By
			if who == "" {
				who = p.T("notes.outsider", "outside the run")
			}

			items = append(items, noteItem{
				at:      timeStr,
				sender:  fmt.Sprintf("↔ %s", strings.ToUpper(who)),
				role:    Live,
				status:  p.T("notes.unread_by_run", "the run does not read it"),
				content: []string{"→ " + e.Text},
			})

		case view.EntryWaiting:
			if e.Cause != "" || e.Text != "" {
				msg := e.Cause
				if msg == "" {
					msg = e.Text
				}

				items = append(items, noteItem{
					at:      timeStr,
					sender:  fmt.Sprintf("🤖 %s", p.T("notes.llm_prompt", "MODEL (asking the operator)")),
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
			"  "+Paint(Dim).Render(p.T("notes.hint_action", "press 'a' to leave a note · press 'c' to open the interactive CLI")),
		)

		return out
	}

	out = append(out, fmt.Sprintf("  %d %s · %s",
		len(items),
		p.T("notes.count", "entries in the dialogue"),
		Paint(OK).Render(p.T("notes.all_filed", "in sync with the model")),
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
		"  "+Paint(Dim).Render(p.T("notes.hint_footer", "press 'a' to add a note · press 'c' or 't' to enter the interactive CLI")),
		"",
	)

	return out
}
