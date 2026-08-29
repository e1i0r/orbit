package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

type noteItem struct {
	at      string
	sender  string
	role    Role
	status  string
	content []string
}

// notesLines is the notes tab's content, ready for the pane.
func (m Model) notesLines() []string {
	lines, _ := m.notesRows()

	return lines
}

// notesRows is that content and, beside it, which item each row that folds
// stands for, laid out in one pass for the reason logRows is.
func (m Model) notesRows() ([]string, map[int]int) {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{"  " + Paint(Bad).Render(m.errSaid(m.logErr))}, nil
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
		return append(out,
			"  "+Paint(Dim).Render(p.T("notes.empty", "no notes or dialogue recorded for this task")),
			"",
			"  "+Paint(Dim).Render(p.T("notes.hint_action", "press 'a' to leave a note · press 'c' to open the interactive CLI")),
		), nil
	}

	out = append(out, fmt.Sprintf("  %d %s · %s",
		len(items),
		p.T("notes.count", "entries in the dialogue"),
		Paint(OK).Render(p.T("notes.all_filed", "in sync with the model")),
	))
	out = append(out, "")

	heads := map[int]int{}

	for i, item := range items {
		rows, folds := m.noteItemRows(item, i)
		if folds {
			heads[len(out)] = i
		}

		out = append(out, rows...)
		out = append(out, "")
	}

	out = append(out,
		"  "+Paint(Dim).Render(p.T("notes.hint_footer", "press 'a' to add a note · press 'c' or 't' to enter the interactive CLI")),
		"",
	)

	return out, heads
}

// noteItemRows is one turn of the dialogue — who said it, when, what became of it
// and what it said — and whether there is more of it than the rows show.
//
// A note is written in Markdown and is as long as the operator made it. A tab
// that sets ten of them open is a tab where the eleventh cannot be found, so
// everything past the opening line waits behind the arrow.
func (m Model) noteItemRows(item noteItem, i int) ([]string, bool) {
	head := "  " + Paint(item.role).Render(item.sender) + "  " +
		Paint(Dim).Render(item.at) + "  " + Paint(Dim).Render(item.status)

	// Trailing blanks are what a note was typed with, not part of what it
	// says: kept, they pad the gap under an open note and are counted as
	// lines a closed one is hiding.
	content := item.content
	for len(content) > 0 && strings.TrimSpace(ansi.Strip(content[len(content)-1])) == "" {
		content = content[:len(content)-1]
	}

	body := make([]string, 0, len(content))

	for _, l := range content {
		switch {
		case strings.HasPrefix(l, "?"):
			body = append(body, "      "+Paint(Warn).Render(l))
		case strings.HasPrefix(l, "→"):
			body = append(body, "      "+Paint(OK).Render(l))
		case strings.HasPrefix(l, "[cli]"):
			body = append(body, "      "+Paint(Live).Render(l))
		default:
			body = append(body, "      "+l)
		}
	}

	if len(body) <= 1 {
		return append([]string{"  " + head}, body...), false
	}

	open := m.rowOpen(tabNotes, i)
	out := []string{Text(Tertiary).Render(foldMark(open)) + head}

	if !open {
		// The opening line and a count of what is under it: a reader
		// scanning the thread needs to know which note is the long one.
		return append(out, body[0], "      "+Text(Tertiary).Render(
			m.opts.Words.P("notes.more_rows", len(body)-1, "{n} more line", "{n} more lines"))), true
	}

	return append(out, body...), true
}
