package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

// thinkingLines renders Pane 11: Extended thinking blocks captured from the engine stream.
func (m Model) thinkingLines() []string {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{"  " + Paint(Bad).Render(m.logErr.Error())}
	}

	var out []string
	count := 0
	for _, e := range m.entries {
		if e.What() == view.EntryThought {
			count++
			header := fmt.Sprintf("  [%s] %s %s",
				Paint(Accent).Render(p.T("thinking.tag", "THINKING")),
				Paint(Dim).Render(p.T("thinking.phase", "phase")),
				Paint(Dim).Render(e.Phase))
			if e.Attempt > 0 {
				header += " · " + Paint(Dim).Render(fmt.Sprintf("attempt %d", e.Attempt))
			}
			out = append(out, "", header)
			if e.Text != "" {
				for _, line := range strings.Split(e.Text, "\n") {
					out = append(out, "    "+Paint(Dim).Render(quoteMark)+line)
				}
			}
		}
	}

	if count == 0 {
		return []string{
			"",
			"  " + Paint(Dim).Render(p.T("thinking.empty", "no thinking blocks captured for this task")),
		}
	}
	return out
}
