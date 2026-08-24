package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

// refusedLines renders Pane 5: Permission denials and refused actions.
func (m Model) refusedLines() []string {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{"  " + Paint(Bad).Render(m.logErr.Error())}
	}

	var out []string
	count := 0
	for _, e := range m.entries {
		if e.What() == view.EntryRefused {
			count++
			toolName := e.Tool
			if toolName == "" {
				toolName = "tool"
			}
			header := fmt.Sprintf("  [%s] %s · %s %s",
				Paint(Bad).Render(p.T("refused.tag", "DENIED")),
				Paint(Accent).Render(toolName),
				Paint(Dim).Render(p.T("refused.phase", "phase")),
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
			"  " + Paint(Dim).Render(p.T("refused.empty", "no actions were refused by permissions for this task")),
		}
	}
	return out
}
