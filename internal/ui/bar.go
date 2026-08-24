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

// barLayout is the key bar, drawn, and where it put each hint.
func (m Model) barLayout(w int) (string, []placedHint) {
	tail := Paint(Dim).Render("[" + m.keys.Help.Help().Key + "] [" + m.keys.Quit.Help().Key + "]")
	hints := m.hints()
	for {
		line := " " + strings.Join(append(drawn(hints), tail), hintGap)
		if lipgloss.Width(line) <= w || len(hints) == 0 {
			return fit(line, w), place(hints)
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
