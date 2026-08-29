package ui

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/view"
)

type thoughtBlock struct {
	at      string
	phase   string
	attempt int
	lines   []string
}

// formatThoughtLine formats a thought line into a clean, concise decision bullet.
func formatThoughtLine(l string) (string, Role) {
	lower := strings.ToLower(l)
	switch {
	case strings.Contains(lower, "decid") || strings.Contains(lower, "conclu") ||
		strings.Contains(lower, "opt") || strings.Contains(lower, "resolv") ||
		strings.Contains(lower, "implement"):
		return "🎯 " + l, OK
	case strings.Contains(lower, "descart") || strings.Contains(lower, "turn down") ||
		strings.Contains(lower, "reject") || strings.Contains(lower, "avoid") ||
		strings.Contains(lower, "evit"):
		return "🚫 " + l, Warn
	case strings.Contains(lower, "investig") || strings.Contains(lower, "check") ||
		strings.Contains(lower, "find") || strings.Contains(lower, "encontr") ||
		strings.Contains(lower, "analiz") || strings.Contains(lower, "evalu"):
		return "🔍 " + l, Live
	case strings.Contains(lower, "porqu") || strings.Contains(lower, "becaus") ||
		strings.Contains(lower, "razon") || strings.Contains(lower, "reason") ||
		strings.Contains(lower, "motivo"):
		return "💡 " + l, Accent
	default:
		return "• " + l, Dim
	}
}

// thinkingLines renders Pane 11: Concise decision reasoning and analysis captured from the model.
func (m Model) thinkingLines() []string {
	p := m.opts.Words
	if m.logErr != nil {
		return []string{"  " + Paint(Bad).Render(m.errSaid(m.logErr))}
	}

	var blocks []thoughtBlock

	for _, e := range m.entries {
		isThought := e.What() == view.EntryThought

		isPhaseSummary := (e.What() == view.EntryFinished || e.What() == view.EntryFailed) &&
			strings.TrimSpace(e.Text) != ""
		if isThought || isPhaseSummary {
			timeStr := ""
			if !e.At.IsZero() {
				timeStr = e.At.Format("15:04:05")
			}

			var lines []string

			for _, l := range strings.Split(e.Text, "\n") {
				l = strings.TrimSpace(l)
				if l != "" {
					lines = append(lines, l)
				}
			}

			if len(lines) > 0 {
				blocks = append(blocks, thoughtBlock{
					at:      timeStr,
					phase:   e.Phase,
					attempt: e.Attempt,
					lines:   lines,
				})
			}
		}
	}

	t, ok := m.task(m.detail)
	if ok && t.CurrentThought != "" {
		blocks = append(blocks, thoughtBlock{
			at:    p.T("thinking.live_now", "live / in flight"),
			phase: t.Phase,
			lines: []string{t.CurrentThought},
		})
	}

	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render(p.T("thinking.title", "Decision Analysis & Agent Thinking")),
		"  " + Paint(Dim).Render(p.T("thinking.subtitle", "why it made each decision, what it evaluated and what it turned down")),
		"",
	}

	if len(blocks) == 0 {
		out = append(out, "  "+Paint(Dim).Render(p.T("thinking.empty", "no thinking blocks or decision logs captured for this task")))
		return out
	}

	out = append(out, fmt.Sprintf("  %d %s",
		len(blocks),
		p.T("thinking.entries_count", "reasoning and decisions analysed"),
	))
	out = append(out, "")

	for _, b := range blocks {
		head := fmt.Sprintf("  %s %s", Paint(Accent).Render("●"), Paint(Dim).Render(b.at))
		if b.phase != "" {
			head += "  " + Paint(Accent).Render(b.phase)
		}

		if b.attempt > 0 {
			head += "  " + Paint(Dim).Render(p.P("thinking.attempt", b.attempt, "attempt {n}", "attempt {n}"))
		}

		out = append(out, head)

		for _, l := range b.lines {
			formatted, role := formatThoughtLine(l)
			if !m.expandedDetail {
				out = append(out, "      "+Paint(role).Render(formatted))
				continue
			}

			wrapped := splitIntoLines(formatted, max(20, m.frame.Body.W-10))
			for _, wl := range wrapped {
				out = append(out, "      "+Paint(role).Render(wl))
			}
		}

		out = append(out, "")
	}

	return out
}
