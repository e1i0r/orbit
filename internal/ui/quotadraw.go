package ui

// Drawing the quota screen.

import (
	"math"
	"strings"

	"github.com/e1i0r/orbit/internal/ui/theme"
)

// quotaBarCells is how wide a window's bar is drawn, and quotaBarFloor is
// the body width under which it is not drawn at all: the sentence beside it
// is the fact, the bar is the same fact at a glance, and on a narrow
// terminal the one that survives is the one that says the number.
const (
	quotaBarCells = 16
	quotaBarFloor = 60
)

// quotaFull is the share of a window past which its bar stops reading as
// room and starts reading as a warning. Three quarters gone is where a
// reader deciding whether to start another run wants the colour to change —
// early enough to still choose a different engine.
const quotaFull = 75

// quotaBarGlyphs are the cell that has been spent and the cell that has not.
const (
	quotaSpent = "█"
	quotaLeft  = "░"
)

// quotaRows draws the screen: a block per engine, a line per window.
func (m Model) quotaRows(h, w int) []string {
	if h <= 0 {
		return nil
	}

	p := m.opts.Words

	out := []string{
		"",
		"  " + theme.Paint(theme.Accent).Render(p.T("quota.title", "Quota")),
		"  " + theme.Paint(theme.Dim).Render(p.T("quota.subtitle",
			"what is left of each engine's windows, and when each comes back")),
	}

	for _, reading := range m.quotaReadings() {
		out = append(out, "", "  "+theme.Paint(theme.Accent).Render(strings.ToUpper(reading.Engine)))
		for _, line := range m.quotaEngineLines(reading, w) {
			out = append(out, fit("    "+line, w))
		}
	}

	waysOut := p.T("quota.ways_out", "{back} back", about("back", m.keys.Back.Help().Key))
	out = append(out, "", fit("  "+theme.Paint(theme.Dim).Render(waysOut), w))

	return fill(out, h)
}

// quotaEngineLines is one engine's block: a line per window it has, or the
// one sentence there is when it has none.
func (m Model) quotaEngineLines(reading QuotaReading, w int) []string {
	if len(reading.Windows) == 0 {
		return []string{theme.Paint(theme.Dim).Render(m.quotaSilence(reading))}
	}

	out := make([]string, 0, len(reading.Windows))
	for _, win := range reading.Windows {
		line := windowUsed(m.opts.Words, win)
		if w >= quotaBarFloor {
			line = quotaBar(pctUsed(win), quotaBarCells) + "  " + line
		}

		out = append(out, line)
	}

	return out
}

// quotaBar is one window's share drawn as cells, painted by how much of it
// is gone.
//
// The share is rounded up rather than down, so that a window a reader has
// started spending is never drawn as untouched: one cell of sixteen is 6%,
// and a bar that waits for the sixth percent to show the first mark reports
// nothing happening while something is.
func quotaBar(pct float64, cells int) string {
	if cells <= 0 {
		return ""
	}

	spent := min(int(math.Ceil(pct/100*float64(cells))), cells)

	role := theme.OK
	if pct >= quotaFull {
		role = theme.Warn
	}

	return theme.Paint(role).Render(strings.Repeat(quotaSpent, spent)) +
		theme.Paint(theme.Dim).Render(strings.Repeat(quotaLeft, cells-spent))
}
