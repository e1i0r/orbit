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

	var thoughts []view.Entry
	for _, e := range m.entries {
		if e.What() == view.EntryThought {
			thoughts = append(thoughts, e)
		}
	}

	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(p.T("thinking.title", "Agent Reasoning & Thinking")),
		"  " + Paint(Dim).Render(p.T("thinking.subtitle", "its own words — what it saw, what it concluded and what it decided")),
		"",
	}

	if len(thoughts) == 0 {
		out = append(out, "  "+Paint(Dim).Render(p.T("thinking.empty", "no thinking blocks captured for this task")))
		return out
	}

	phaseName := "run"
	if len(thoughts) > 0 && thoughts[0].Phase != "" {
		phaseName = thoughts[0].Phase
	}
	out = append(out, fmt.Sprintf("  %s · %d %s",
		Paint(Accent).Render(phaseName),
		len(thoughts),
		p.T("thinking.entries", "entries"),
	))
	out = append(out, "")

	for _, e := range thoughts {
		timeStr := ""
		if !e.At.IsZero() {
			timeStr = e.At.Format("15:04:05")
		}
		if timeStr != "" {
			out = append(out, "    "+Paint(Dim).Render(timeStr))
		}
		if e.Text != "" {
			for _, line := range strings.Split(e.Text, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				out = append(out, "    "+line)
			}
		}
		out = append(out, "")
	}

	return out
}
