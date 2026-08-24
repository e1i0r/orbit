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

	var out []string
	count := 0
	for _, e := range m.entries {
		if e.What() == view.EntryNoted {
			count++
			timeStr := ""
			if !e.At.IsZero() {
				timeStr = e.At.Format("15:04:05")
			}
			header := fmt.Sprintf("  [%s] %s",
				Paint(Accent).Render(p.T("notes.note", "NOTE")),
				Paint(Dim).Render(timeStr))
			out = append(out, "", header)
			if e.Text != "" {
				for _, line := range strings.Split(e.Text, "\n") {
					out = append(out, "    "+line)
				}
			}
		}
	}

	if count == 0 {
		return []string{
			"",
			"  " + Paint(Dim).Render(p.T("notes.empty", "no notes recorded for this task · press a to leave one")),
		}
	}
	return out
}
