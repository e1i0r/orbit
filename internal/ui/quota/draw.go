package quota

// Drawing the quota screen.

import (
	"math"
	"strings"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/roster"
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

// View draws the screen: a block per engine, a line per window.
func View(h, w int, e Env) []string {
	if h <= 0 {
		return nil
	}

	p := e.Words

	out := []string{
		"",
		"  " + theme.Paint(theme.Accent).Render(p.T("quota.title", "Quota")),
		"  " + theme.Paint(theme.Dim).Render(p.T("quota.subtitle",
			"what is left of each engine's windows, and when each comes back")),
	}

	for _, reading := range Readings(e) {
		out = append(out, "", "  "+theme.Paint(theme.Accent).Render(strings.ToUpper(reading.Engine)))
		for _, line := range engineLines(reading, w, e) {
			out = append(out, cells.Fit("    "+line, w))
		}
	}

	waysOut := p.T("quota.ways_out", "{back} back", about("back", e.Keys.Back.Help().Key))
	out = append(out, "", cells.Fit("  "+theme.Paint(theme.Dim).Render(waysOut), w))

	return cells.Fill(out, h)
}

// engineLines is one engine's block: a line per window it has, or the
// one sentence there is when it has none.
func engineLines(reading roster.Reading, w int, e Env) []string {
	if len(reading.Windows) == 0 {
		return []string{theme.Paint(theme.Dim).Render(roster.Quiet(e.Words, reading))}
	}

	out := make([]string, 0, len(reading.Windows))
	for _, win := range reading.Windows {
		line := roster.Says(e.Words, win)
		if w >= quotaBarFloor {
			line = bar(roster.Used(win), quotaBarCells) + "  " + line
		}

		out = append(out, line)
	}

	return out
}

// bar is one window's share drawn as cells, painted by how much of it
// is gone.
//
// The share is rounded up rather than down, so that a window a reader has
// started spending is never drawn as untouched: one cell of sixteen is 6%,
// and a bar that waits for the sixth percent to show the first mark reports
// nothing happening while something is.
func bar(pct float64, wide int) string {
	if wide <= 0 {
		return ""
	}

	spent := min(int(math.Ceil(pct/100*float64(wide))), wide)

	role := theme.OK
	if pct >= quotaFull {
		role = theme.Warn
	}

	return theme.Paint(role).Render(strings.Repeat(quotaSpent, spent)) +
		theme.Paint(theme.Dim).Render(strings.Repeat(quotaLeft, wide-spent))
}
