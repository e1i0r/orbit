package ui

// The status line: the decision engine when it is switched on and cannot
// work, spent, tasks, events, the heartbeat, quota remaining. It gives up
// fields from the right as the terminal narrows, and disappears entirely on
// a short terminal, so the order it builds them in is what survives a
// narrow one.

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/roster"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/words"
)

type statusSegment struct {
	text string
	role theme.Role
}

func (m Model) statusRows() []string {
	r := m.frame.Status
	if r.H <= 0 {
		return nil
	}

	return cells.Fill([]string{m.statusLine(r.W)}, r.H)
}

func (m Model) statusLine(w int) string {
	p := m.opts.Words

	var segments []statusSegment

	// 1. A decision engine that is switched on and cannot answer.
	//
	// First, because this line gives up fields from the right and keeps
	// the leftmost one whatever the width: the order here is priority,
	// not reading order. A warning appended after the readings is correct
	// at 200 columns and gone at 60, and the only thing on this line that
	// somebody has to do something about is the one that would go.
	//
	// It costs the other fields nothing, because it is absent every time
	// there is nothing wrong.
	if stalled, ok := m.decisionSegment(p); ok {
		segments = append(segments, stalled)
	}

	// 2. Spent (gastado)
	if spent, ok := m.spentSegment(p); ok {
		segments = append(segments, spent)
	}

	// 3. Tasks (tareas totales)
	tasksStr := p.P("status.total_tasks", len(m.board.Tasks), "{n} task", "{n} tasks")
	segments = append(segments, statusSegment{text: tasksStr, role: theme.Dim})

	// 4. Events (eventos)
	eventsStr := p.T("status.events", "{events} events", about("events", strconv.Itoa(m.board.Health.EventsRead)))
	segments = append(segments, statusSegment{text: eventsStr, role: theme.Dim})

	// 5. Heartbeat (latido)
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
		segments = append(segments, statusSegment{text: m.spin(), role: theme.Dim})
	}

	// 6. Quota remaining (quota restante)
	if quota, ok := m.quotaSegment(p); ok {
		segments = append(segments, quota)
	}

	sep := theme.Paint(theme.Dim).Render("  ·  ")

	// Try fitting all segments; drop from the right if too wide
	for len(segments) > 1 {
		rendered := renderSegments(segments, sep)
		if lipgloss.Width(rendered)+2 <= w {
			return cells.Fit("  "+rendered, w)
		}

		segments = segments[:len(segments)-1]
	}

	if len(segments) == 1 {
		rendered := theme.Paint(segments[0].role).Render(segments[0].text)
		return cells.Fit("  "+rendered, w)
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

	return statusSegment{text: text, role: theme.Accent}, true
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
		return statusSegment{text: roster.Says(p, reading.Windows[0]), role: theme.Dim}, true
	}

	if reading.Money || reading.Sourced {
		return statusSegment{}, false
	}

	text := p.T("status.quota_unread", "no quota source for {engine}",
		about("engine", reading.Engine))

	return statusSegment{text: text, role: theme.Dim}, true
}

// decisionSegment is the one thing the window says about the decision
// engine, and it says it only when the setting and the key disagree.
//
// Nothing when it is off, which is what it ships as: a reader who has not
// turned it on is not waiting for it. Nothing when it works, for the
// reason the quota field says nothing about an engine paid per token —
// there is no action behind the sentence. It is the third case this is
// for: `decisions` set, no key in the environment, and a run that walks to
// its gate and waits for a person while the window looks like a window
// with a decision engine in it. Both halves were readable and neither was
// read out.
//
// Warn and not Alert, because nothing is broken. The run is doing what a
// run did before any of this existed.
//
// It is here rather than in the key bar because a chip is glanced at on
// every frame and this is a sentence acted on once.
func (m Model) decisionSegment(p *words.Printer) (statusSegment, bool) {
	if m.opts.Settings == nil {
		return statusSegment{}, false
	}

	allowed, missing := m.opts.Settings.Deciding()
	if allowed == "" || missing == "" {
		return statusSegment{}, false
	}

	text := p.T("status.decisions_unkeyed", "decisions {state}, no {key} in the environment",
		words.Arg{Name: "state", Value: allowed},
		words.Arg{Name: "key", Value: missing})

	return statusSegment{text: text, role: theme.Warn}, true
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

	return roster.Spent(m.opts.Words, m.opts.Quota(m.dialEngine("")))
}

func renderSegments(segs []statusSegment, sep string) string {
	parts := make([]string, len(segs))
	for i, s := range segs {
		parts[i] = theme.Paint(s.role).Render(s.text)
	}

	return strings.Join(parts, sep)
}
