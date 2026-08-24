package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

// notesLines renders Pane 9: Operator notes left for this task.
func (m Model) notesLines() []string {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{"  " + Paint(Bad).Render(m.logErr.Error())}
	}

	var notes []view.Entry
	for _, e := range m.entries {
		if e.What() == view.EntryNoted {
			notes = append(notes, e)
		}
	}

	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(p.T("notes.title", "Operator Notes & Guidance")),
		"  " + Paint(Dim).Render(p.T("notes.subtitle", "what you told it, and if anything has read it")),
		"",
	}

	if len(notes) == 0 {
		out = append(out, "  "+Paint(Dim).Render(p.T("notes.empty", "no notes recorded for this task · press a to leave one")))
		return out
	}

	out = append(out, fmt.Sprintf("  %d %s · %s",
		len(notes),
		p.T("notes.count", "notes"),
		Paint(OK).Render(p.T("notes.all_filed", "all filed")),
	))
	out = append(out, "")

	for i, e := range notes {
		timeStr := ""
		if !e.At.IsZero() {
			timeStr = e.At.Format("15:04:05")
		}
		statusNote := "read by run"
		if e.Attempt > 0 {
			statusNote = fmt.Sprintf("read by run %d", e.Attempt)
		}

		bullet := fmt.Sprintf("  %s %d  %s  %s",
			Paint(Accent).Render("●"),
			i+1,
			Paint(Dim).Render(timeStr),
			Paint(Dim).Render(statusNote),
		)
		out = append(out, bullet)

		if e.Text != "" {
			for _, line := range strings.Split(e.Text, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if strings.HasPrefix(line, "?") {
					out = append(out, "      "+Paint(Warn).Render(line))
				} else if strings.HasPrefix(line, "→") {
					out = append(out, "      "+Paint(OK).Render(line))
				} else {
					out = append(out, "      "+line)
				}
			}
		}
		out = append(out, "")
	}

	return out
}
