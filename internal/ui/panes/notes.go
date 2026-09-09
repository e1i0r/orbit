package panes

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/markdown"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
)

type noteItem struct {
	at      string
	sender  string
	role    theme.Role
	status  string
	content []string
}

// Notes is everything spoken with the model — the operator's notes, the
// sessions held beside the run, and what the model stopped to ask — and
// beside it which item each row that folds stands for, laid out in one pass
// for the reason the timeline is.
func Notes(e Env) ([]string, map[int]int) {
	p := e.Words
	if e.Failed != "" {
		return []string{"  " + theme.Paint(theme.Bad).Render(e.Failed)}, nil
	}

	var items []noteItem

	// The notes no phase has picked up yet. A note is read by the next
	// phase that starts (task.unconsumedNotes), so a note with a phase
	// start after it has been read and one without is still waiting —
	// which is the difference between steering a run and thinking you
	// steered it.
	var pending []int

	noteIndex := 0

	for _, entry := range e.Entries {
		timeStr := ""
		if !entry.At.IsZero() {
			timeStr = entry.At.Format("15:04:05")
		}

		switch entry.What() {
		case view.EntryNoted:
			noteIndex++

			// Only when the record does not already say which run took it.
			// An attempt on a note is the record's own answer, and the
			// record's answer is the one to draw.
			if entry.Attempt == 0 {
				pending = append(pending, len(items))
			}

			statusNote := p.T("notes.read_by_run", "read by the run")
			if entry.Attempt > 0 {
				statusNote = p.T("notes.read_by_run_n", "read by run {n}", about("n", strconv.Itoa(entry.Attempt)))
			}

			senderLabel := fmt.Sprintf("● %d  %s", noteIndex, p.T("notes.operator", "OPERATOR"))
			content := markdown.Render(entry.Text, e.Frame.Body.W, e.Raw)
			items = append(items, noteItem{
				at:      timeStr,
				sender:  senderLabel,
				role:    theme.Accent,
				status:  statusNote,
				content: content,
			})

		case view.EntryStarted:
			// Everything written before this phase began went into its
			// prompt, whatever else this entry is drawn as below.
			pending = nil

		case view.EntryDialogue:
			// Beside the notes and not among them. This is what a model or
			// a session did to the task, which is the other half of the
			// dialogue the reader came to this tab for — and the half no
			// phase is ever handed, so it is never mistaken for a note the
			// next run will read.
			who := entry.By
			if who == "" {
				who = p.T("notes.outsider", "outside the run")
			}

			items = append(items, noteItem{
				at:      timeStr,
				sender:  fmt.Sprintf("↔ %s", strings.ToUpper(who)),
				role:    theme.Live,
				status:  p.T("notes.unread_by_run", "the run does not read it"),
				content: turnLines(entry.Text, e.Frame.Body.W),
			})

		case view.EntryWaiting:
			if entry.Cause != "" || entry.Text != "" {
				msg := entry.Cause
				if msg == "" {
					msg = entry.Text
				}

				items = append(items, noteItem{
					at:      timeStr,
					sender:  fmt.Sprintf("🤖 %s", p.T("notes.llm_prompt", "MODEL (asking the operator)")),
					role:    theme.Warn,
					status:  entry.Phase,
					content: []string{"? " + msg},
				})
			}
		}
	}

	for _, i := range pending {
		items[i].status = p.T("notes.waiting_for_a_phase", "the next phase reads it")
	}

	out := []string{
		"",
		"  " + theme.Paint(theme.Accent).Bold(true).Render(p.T("notes.title", "Operator Notes & LLM Dialogue")),
		"  " + theme.Paint(theme.Dim).Render(p.T("notes.subtitle", "everything spoken with the model, notes filed and interactive sessions")),
		"",
	}

	if len(items) == 0 {
		return append(out,
			"  "+theme.Paint(theme.Dim).Render(p.T("notes.empty", "no notes or dialogue recorded for this task")),
			"",
			"  "+theme.Paint(theme.Dim).Render(p.T("notes.hint_action", "press 'a' to leave a note · press 'c' to open the interactive CLI")),
		), nil
	}

	out = append(out, fmt.Sprintf("  %d %s · %s",
		len(items),
		p.T("notes.count", "entries in the dialogue"),
		theme.Paint(theme.OK).Render(p.T("notes.all_filed", "in sync with the model")),
	))
	out = append(out, "")

	heads := map[int]int{}

	for i, item := range items {
		rows, folds := e.noteItemRows(item, i)
		if folds {
			heads[len(out)] = i
		}

		out = append(out, rows...)
		out = append(out, "")
	}

	out = append(out,
		"  "+theme.Paint(theme.Dim).Render(p.T("notes.hint_footer", "press 'a' to add a note · press 'c' or 't' to enter the interactive CLI")),
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
func (e Env) noteItemRows(item noteItem, i int) ([]string, bool) {
	head := "  " + theme.Paint(item.role).Render(item.sender) + "  " +
		theme.Paint(theme.Dim).Render(item.at) + "  " + theme.Paint(theme.Dim).Render(item.status)

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
			body = append(body, "      "+theme.Paint(theme.Warn).Render(l))
		case strings.HasPrefix(l, "→"):
			body = append(body, "      "+theme.Paint(theme.OK).Render(l))
		case strings.HasPrefix(l, "[cli]"):
			body = append(body, "      "+theme.Paint(theme.Live).Render(l))
		default:
			body = append(body, "      "+l)
		}
	}

	if len(body) <= 1 {
		return append([]string{"  " + head}, body...), false
	}

	open := e.row(i)
	out := []string{theme.Text(theme.Tertiary).Render(cells.Fold(open)) + head}

	if !open {
		// The opening line and a count of what is under it: a reader
		// scanning the thread needs to know which note is the long one.
		return append(out, body[0], "      "+theme.Text(theme.Tertiary).Render(
			e.Words.P("notes.more_rows", len(body)-1, "{n} more line", "{n} more lines"))), true
	}

	return append(out, body...), true
}

// turnLines is one thing said, wrapped to the pane: the arrow on the first
// line and the rest aligned under it.
//
// It was a single line until sessions were read back into the record. A
// tool call and a handover are a sentence long, so nothing was lost by
// drawing them whole; a turn somebody typed is a paragraph, and everything
// past the pane's width was.
func turnLines(text string, w int) []string {
	var out []string

	// Paragraph by paragraph, because a blank line between two thoughts is
	// something the writer put there. Wrapping folds the words inside one.
	for _, para := range strings.Split(text, "\n") {
		if strings.TrimSpace(para) == "" {
			out = append(out, "")
			continue
		}

		// The pane indents an item's content by six, and the arrow takes
		// two of what is left.
		out = append(out, cells.Lines(para, max(w-8, 20))...)
	}

	for i, line := range out {
		if i == 0 {
			out[i] = "→ " + line
			continue
		}

		out[i] = "  " + line
	}

	return out
}
