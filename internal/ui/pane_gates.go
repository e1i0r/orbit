package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

// gatesLines renders Pane 3: Verification gates per phase and attempt.
func (m Model) gatesLines() []string {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{"  " + Paint(Bad).Render(m.logErr.Error())}
	}
	var out []string
	count := 0
	for _, e := range m.entries {
		if e.What() == view.EntryGatePassed || e.What() == view.EntryGateFailed {
			count++
			role := OK
			status := p.T("gates.passed", "PASSED")
			if e.What() == view.EntryGateFailed {
				role = Bad
				status = p.T("gates.failed", "FAILED")
			}
			gateName := e.Gate
			if gateName == "" {
				gateName = "gate"
			}
			exitStr := e.Exit
			if exitStr == "" {
				exitStr = "0"
			}
			header := fmt.Sprintf("  [%s] %s · %s %s · exit %s",
				Paint(role).Render(status),
				Paint(Accent).Render(gateName),
				Paint(Dim).Render(p.T("gates.phase", "phase")),
				Paint(Dim).Render(e.Phase),
				exitStr)
			out = append(out, "", header)
			if e.Text != "" {
				for _, line := range strings.Split(e.Text, "\n") {
					out = append(out, "    "+Paint(Dim).Render(quoteMark)+line)
				}
			}
		}
	}
	if count == 0 {
		return []string{"", "  " + Paint(Dim).Render(p.T("gates.empty", "no verification gates have run for this task"))}
	}
	return out
}
