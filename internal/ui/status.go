package ui

// The status line: five fields across the terminal — spent, tasks, events,
// the heartbeat, quota remaining — giving up fields from the right as the
// terminal narrows, and disappearing entirely on a short terminal.

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

	// 4. Heartbeat (latido)
	//
	// This was "{ms}ms read": how long the last board read took, painted red
	// past 100ms. Three things were wrong with it. There is no screen that
	// puts the number in context, so it was a measurement with nothing to
	// measure against; a rescan that takes 182ms because the board is large
	// is not a fault, so the red said something had broken when nothing had;
	// and the number changed on every refresh, which made the one part of
	// the status line that moves the one part that means least.
	//
	// What that corner is for is whether anything is happening, and that is
	// one glyph. It is absent rather than frozen when nothing is running,
	// because the frame clock stops then and a spinner standing still is a
	// worse lie than no spinner at all.
	if m.moving() {
		segments = append(segments, statusSegment{text: m.spin(), role: Dim})
	}

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
