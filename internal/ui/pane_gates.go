package ui

import (
	"fmt"

	"github.com/e1i0r/orbit/internal/view"
)

type gateCheck struct {
	name     string
	duration string
	passed   bool
	command  string
	reason   string
	phase    string
}

// gatesLines renders Pane 3: Verification gates per phase and attempt.
func (m Model) gatesLines() []string {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{"  " + Paint(Bad).Render(m.logErr.Error())}
	}

	var checks []gateCheck
	for _, e := range m.entries {
		if e.What() == view.EntryGatePassed || e.What() == view.EntryGateFailed {
			gName := e.Gate
			if gName == "" {
				gName = "check"
			}
			cmd := e.Text
			if cmd == "" {
				cmd = e.Tool
			}
			checks = append(checks, gateCheck{
				name:     gName,
				passed:   e.What() == view.EntryGatePassed,
				command:  cmd,
				reason:   e.Cause,
				phase:    e.Phase,
				duration: "ok",
			})
		}
	}

	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(p.T("gates.title", "Verification Gates & Checks")),
		"  " + Paint(Dim).Render(p.T("gates.subtitle", "what needs to pass — by attempt, and why it stopped")),
		"",
	}

	if len(checks) == 0 {
		out = append(out, "  "+Paint(Dim).Render(p.T("gates.empty", "no verification gates have run for this task")))
		return out
	}

	passedCount := 0
	for _, c := range checks {
		if c.passed {
			passedCount++
		}
	}

	summaryRole := OK
	summaryWord := "pass"
	if passedCount < len(checks) {
		summaryRole = Bad
		summaryWord = fmt.Sprintf("falló %d", len(checks)-passedCount)
	}

	out = append(out, fmt.Sprintf("  %s %s   %d/%d %s   %s",
		Paint(Accent).Render("▼"),
		Paint(Accent).Bold(true).Render(p.T("gates.attempt", "intento 1")),
		passedCount, len(checks),
		p.T("gates.passed_word", "pasaron"),
		Paint(summaryRole).Bold(true).Render(summaryWord),
	))
	out = append(out, "")

	for _, c := range checks {
		icon := Paint(OK).Render("✅")
		statusStr := Paint(OK).Render("pass")
		if !c.passed {
			icon = Paint(Bad).Render("❌")
			statusStr = Paint(Bad).Render("fail")
		}
		cmdStr := c.command
		if cmdStr != "" {
			cmdStr = Paint(Dim).Render(cmdStr)
		}
		line := fmt.Sprintf("    %s %-16s %-8s %s",
			icon,
			Paint(Accent).Render(c.name),
			statusStr,
			cmdStr,
		)
		out = append(out, line)
		if !c.passed && c.reason != "" {
			out = append(out, fmt.Sprintf("       %s %s",
				Paint(Bad).Render("por qué falló ·"),
				Paint(Bad).Render(c.reason),
			))
		}
	}
	out = append(out, "")

	return out
}
