package ui

// The status line: five fields across the terminal — spent, tasks, events,
// read time, quota remaining — giving up fields from the right as the terminal
// narrows, and disappearing entirely on a short terminal.

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

type statusSegment struct {
	text string
	role Role
}

func (m Model) statusRows() []string {
	r := m.frame.Status
	if r.H <= 0 {
		return nil
	}
	return fill([]string{m.statusLine(r.W)}, r.H)
}

func (m Model) statusLine(w int) string {
	p := m.opts.Words
	var segments []statusSegment

	// 1. Spent (gastado)
	var totalCost float64
	for _, t := range m.board.Tasks {
		totalCost += t.Cost
	}
	spentStr := p.T("status.spent", "{cost} spent", about("cost", fmt.Sprintf("$%.2f", totalCost)))
	segments = append(segments, statusSegment{text: spentStr, role: Accent})

	// 2. Tasks (tareas totales)
	tasksStr := p.P("status.total_tasks", len(m.board.Tasks), "{n} task", "{n} tasks")
	segments = append(segments, statusSegment{text: tasksStr, role: Dim})

	// 3. Events (eventos)
	eventsStr := p.T("status.events", "{events} events", about("events", strconv.Itoa(m.board.Health.EventsRead)))
	segments = append(segments, statusSegment{text: eventsStr, role: Dim})

	// 4. Read time (ms de lectura)
	ms := m.board.Health.Duration.Milliseconds()
	readRole := Dim
	if ms > 100 {
		readRole = Bad
	}
	readStr := p.T("status.read_time", "{ms}ms read", about("ms", strconv.FormatInt(ms, 10)))
	segments = append(segments, statusSegment{text: readStr, role: readRole})

	// 5. Quota remaining (quota restante)
	if m.opts.Quota != nil {
		windows := m.opts.Quota()
		if len(windows) > 0 {
			qw := windows[0]
			pctLeft := 100.0 - qw.Pct
			if pctLeft < 0 {
				pctLeft = 0
			}
			resetsStr := fmtReset(qw.ResetsIn)
			quotaStr := p.T("status.quota", "{pct} left in {label} · resets in {resets}",
				about("pct", fmt.Sprintf("%.0f%%", pctLeft)),
				about("label", qw.Label),
				about("resets", resetsStr),
			)
			segments = append(segments, statusSegment{text: quotaStr, role: Dim})
		}
	}

	sep := Paint(Dim).Render("  ·  ")

	// Try fitting all segments; drop from the right if too wide
	for len(segments) > 1 {
		rendered := renderSegments(segments, sep)
		if lipgloss.Width(rendered)+2 <= w {
			return fit("  "+rendered, w)
		}
		segments = segments[:len(segments)-1]
	}

	if len(segments) == 1 {
		rendered := Paint(segments[0].role).Render(segments[0].text)
		return fit("  "+rendered, w)
	}
	return ""
}

func renderSegments(segs []statusSegment, sep string) string {
	parts := make([]string, len(segs))
	for i, s := range segs {
		parts[i] = Paint(s.role).Render(s.text)
	}
	return strings.Join(parts, sep)
}

func fmtReset(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
