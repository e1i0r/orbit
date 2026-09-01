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

	"github.com/e1i0r/orbit/internal/words"
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
	if spent, ok := m.spentSegment(p); ok {
		segments = append(segments, spent)
	}

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
	if quota, ok := m.quotaSegment(p); ok {
		segments = append(segments, quota)
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

// spentSegment is what the board has cost, counting only the tasks whose
// engines charge by what they did.
//
// A run under a subscription is left out rather than added at whatever its
// engine reported. The money was spent once, in advance, and a share of it
// beside a growing total tells a reader they are being charged $0.42 that
// nobody is charging them. Which engines those are is not decided here —
// Options.Quota answers it, over internal/quota, so that this line and the
// budgets and the stats cannot come to different conclusions about the same
// engine.
//
// A window with no quota port counts everything, because a window opened
// without one has nobody to ask, and money is what every engine's own
// command line reports. When there are tasks and not one of them is charged
// for, the field is absent rather than zero — but a board with nothing on it
// keeps it, because $0.00 spent on nothing is true whoever is paying.
func (m Model) spentSegment(p *words.Printer) (statusSegment, bool) {
	var (
		total   float64
		charged bool
	)

	for _, t := range m.board.Tasks {
		if !m.spends(t.Engine) {
			continue
		}

		charged = true
		total += t.Cost
	}

	if len(m.board.Tasks) > 0 && !charged {
		return statusSegment{}, false
	}

	text := p.T("status.spent", "{cost} spent", about("cost", fmt.Sprintf("$%.2f", total)))

	return statusSegment{text: text, role: Accent}, true
}

// spends is whether this engine's use is spoken about in money.
func (m Model) spends(engine string) bool {
	if m.opts.Quota == nil {
		return true
	}

	return m.opts.Quota(m.dialEngine(engine)).Money
}

// quotaSegment is how much of the current engine's window is left.
//
// Three answers, and each is a different sentence. A window that was read is
// the percentage and the clock. An engine paid per token has no window to
// read and needs none — the spent field above is its whole story — so it
// draws nothing here. An engine paid by subscription with nowhere to read
// its window from says exactly that: it is the one case where the number
// that matters exists and cannot be seen, and silence there reads as a
// reader having plenty left.
func (m Model) quotaSegment(p *words.Printer) (statusSegment, bool) {
	if m.opts.Quota == nil {
		return statusSegment{}, false
	}

	reading := m.opts.Quota(m.dialEngine(""))

	if len(reading.Windows) > 0 {
		return statusSegment{text: windowUsed(p, reading.Windows[0]), role: Dim}, true
	}

	if reading.Money || reading.Sourced {
		return statusSegment{}, false
	}

	text := p.T("status.quota_unread", "no quota source for {engine}",
		about("engine", reading.Engine))

	return statusSegment{text: text, role: Dim}, true
}

// quotaChip is what the header carries about the engine's quota: the share
// used of each window it has, short enough to sit beside the engine chip the
// number is about.
//
// It is the same fact as the status line's sentence at a different length.
// The header is where a reader looks for what is standing — which repository,
// which engine, which language — and how much of the window is left is one of
// those: it does not change with the task under the cursor, and a reader
// deciding whether to start another run is asking about it before they look
// anywhere else. The countdown stays in the status line, because "resets in
// 1h15m" is a sentence and this is a chip.
//
// Nothing is drawn when there is nothing read. An engine with no source has
// no windows, and the sentence saying so is the status line's to print once
// rather than this line's to repeat.
func (m Model) quotaChip() string {
	if m.opts.Quota == nil {
		return ""
	}

	return m.windowsUsed(m.opts.Quota(m.dialEngine("")))
}

// windowsUsed is a reading's windows on one line: the share used of each,
// and the word said once at the end.
//
// Empty for a reading with nothing in it, which is what a caller draws when
// there is nothing to draw rather than a sentence about absence — the one
// place that says a source is missing is the status line, once.
func (m Model) windowsUsed(reading QuotaReading) string {
	var parts []string

	for _, w := range reading.Windows {
		parts = append(parts, fmt.Sprintf("%.0f%% %s", pctUsed(w), w.Label))
	}

	if len(parts) == 0 {
		return ""
	}

	// The word is said once, at the end, and it is the word the providers
	// own screens use. A percentage on a status bar is read as whichever of
	// the two numbers the reader expects, and the two are opposites: 1% of a
	// window is either almost nothing spent or almost nothing left. Saying
	// used keeps this chip and the /usage screen a reader has open in
	// another terminal reporting the same figure rather than its complement.
	return m.opts.Words.T("header.quota_used", "{windows} used",
		about("windows", strings.Join(parts, dot)))
}

// pctUsed is the share of a window already spent, capped at the whole of it:
// a proxy reporting more used than there was is reporting an overage, and
// 140% of a window drawn as a bar is a bar with nowhere to go.
func pctUsed(w QuotaWindow) float64 {
	if w.Pct > 100 {
		return 100
	}

	return w.Pct
}

// windowUsed is one quota window as a reader reads it.
func windowUsed(p *words.Printer, w QuotaWindow) string {
	return p.T("status.quota", "{pct} used in {label} · resets in {resets}",
		about("pct", fmt.Sprintf("%.0f%%", pctUsed(w))),
		about("label", w.Label),
		about("resets", fmtReset(w.ResetsIn)),
	)
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
