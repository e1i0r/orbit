package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// barHint is one entry of the key bar: the hint as it is drawn, and the
// keystroke it stands for.
type barHint struct {
	key  string
	text string
}

// placedHint is one hint of the drawn bar and the cells it occupies, counted
// from the left edge of the terminal.
type placedHint struct {
	key  string
	x, w int
}

// barLine is what can be pressed right now.
func (m Model) barLine(w int) string {
	line, _ := m.barLayout(w)
	return line
}

// barFooterChips renders autopilot and interactive CLI on the footer right side
func (m Model) barFooterChips() string {
	p := m.opts.Words
	var chips []string

	// Autopilot chip
	pip, role := pipOff, Dim
	if m.autopilotOn() {
		pip, role = pipOn, Live
	}
	chips = append(chips, Paint(role).Render("⚡ "+p.T("header.autopilot", "autopilot"))+" "+Paint(role).Render(pip)+" "+Paint(role).Bold(true).Render("["+m.keys.Autopilot.Help().Key+"]"))

	// Interactive CLI chip
	if m.screen == screenList {
		chips = append(chips, Paint(Live).Render("💬 "+p.T("header.cli_chip", "cli"))+" "+Paint(Live).Bold(true).Render("[c]"))
	}

	return strings.Join(chips, "    ")
}

// barLayout is the key bar, drawn, and where it put each hint.
func (m Model) barLayout(w int) (string, []placedHint) {
	tail := Paint(Dim).Render("[" + m.keys.Help.Help().Key + "] [" + m.keys.Quit.Help().Key + "]")
	chips := m.barFooterChips()
	chipsW := lipgloss.Width(chips)
	hints := m.hints()

	for {
		leftStr := " " + strings.Join(append(drawn(hints), tail), hintGap)
		leftW := lipgloss.Width(leftStr)
		if leftW+chipsW+4 <= w && chips != "" {
			space := w - leftW - chipsW
			return leftStr + strings.Repeat(" ", space) + chips, place(hints)
		}
		if len(hints) == 0 {
			if leftW <= w {
				return fit(leftStr, w), place(hints)
			}
			return fit(leftStr, w), nil
		}
		hints = hints[:len(hints)-1]
	}
}

// drawn is the hints as barLine joins them.
func drawn(hints []barHint) []string {
	out := make([]string, 0, len(hints)+1)
	for _, h := range hints {
		out = append(out, h.text)
	}
	return out
}

// place walks the hints the way the line was joined and says where each one
// starts, measuring in cells.
func place(hints []barHint) []placedHint {
	out := make([]placedHint, 0, len(hints))
	x := 1
	for _, h := range hints {
		cells := lipgloss.Width(h.text)
		out = append(out, placedHint{key: h.key, x: x, w: cells})
		x += cells + lipgloss.Width(hintGap)
	}
	return out
}
