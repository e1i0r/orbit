package ui

// The frame: which rows each region gets, what the body says when there is
// nothing in it, and the two primitives every line on this screen goes
// through on its way out.
//
// Nothing here decides what a region contains — header.go and cells.go do —
// and nothing here measures anything in bytes. fit and spread are the only
// two places a line is cut or padded, which is what makes "no row is wider
// than the terminal" one claim to check rather than thirty.

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// View draws the whole terminal, every time.
//
// It always returns exactly the number of rows the frame was fitted to, and
// every one of them at most as many cells wide. That is not a nicety: Bubble
// Tea diffs the frame it is given against the last one, and a frame one row
// short scrolls the whole screen by a row — which reads as the window
// flickering rather than as an off-by-one.
func (m Model) View() tea.View {
	if m.height <= 0 || m.width <= 0 {
		return tea.NewView("")
	}

	if m.tooNarrow {
		return tea.NewView(m.refusal())
	}

	lines := make([]string, 0, m.height)
	lines = append(lines, m.headerRows()...)
	lines = append(lines, m.statusRows()...)
	// The palette, a menu and a watched run sit above every screen: the
	// line and the list own the body while they are up, and the output of
	// the command the line raised keeps the body until esc takes it down.
	switch {
	case m.palette.open:
		lines = append(lines, m.paletteRows(m.frame.Body.H, m.frame.Body.W)...)
	case m.note.open:
		lines = append(lines, m.noteRows(m.frame.Body.H, m.frame.Body.W)...)
	case m.menu.open:
		lines = append(lines, m.menuRows(m.frame.Body.H, m.frame.Body.W)...)
	case m.watchUp:
		lines = append(lines, m.watchRows(m.frame.Body.H, m.frame.Body.W)...)
	default:
		switch m.screen {
		case screenDetail:
			lines = append(lines, m.detailRows(m.frame.Body.H, m.frame.Body.W)...)
		case screenStart:
			lines = append(lines, m.startRows(m.frame.Body.H, m.frame.Body.W)...)
		case screenCompose:
			lines = append(lines, m.composeRows(m.frame.Body.H, m.frame.Body.W)...)
		case screenSettings:
			lines = append(lines, m.settingsRows(m.frame.Body.H, m.frame.Body.W)...)
		case screenFlows:
			lines = append(lines, m.flowsRows(m.frame.Body.H, m.frame.Body.W)...)
		case screenRepos:
			lines = append(lines, m.repolistRows(m.frame.Body.H, m.frame.Body.W)...)
		case screenEngines:
			lines = append(lines, m.enginesRows(m.frame.Body.H, m.frame.Body.W)...)
		case screenQuota:
			lines = append(lines, m.quotaRows(m.frame.Body.H, m.frame.Body.W)...)
		case screenHelp:
			lines = append(lines, m.helpRows(m.frame.Body.H, m.frame.Body.W)...)
		case screenSupervisor:
			lines = append(lines, m.supervisorRows(m.frame.Body.H, m.frame.Body.W)...)
		default:
			lines = append(lines, m.bodyRows()...)
		}
	}

	lines = append(lines, m.bandRows()...)
	lines = append(lines, m.barRows()...)
	v := tea.NewView(strings.Join(lines, "\n"))
	// The window's own paper, so the frame reads as a program and not as
	// scrollback in whatever colours the terminal happens to be set to. Bubble
	// Tea puts the terminal back the way it found it on the way out.
	v.BackgroundColor = WindowBackground()
	v.ForegroundColor = WindowForeground()
	// AllMotion and not CellMotion: a message for every cell the pointer
	// crosses, whether or not a button is down. It costs that traffic and
	// it buys the hint under the pointer explaining itself, which is the
	// one thing on this window that is drawn for being pointed at. The
	// work per message is a row comparison — hitBar answers off its own
	// row before it lays anything out — and the model only changes when
	// the hint under the pointer does.
	v.MouseMode = tea.MouseModeAllMotion

	return v
}

// headerRows is the header region: optional top padding row, the line, then its rule.
func (m Model) headerRows() []string {
	r := m.frame.Header
	if r.H <= 0 {
		return nil
	}

	if r.H == 1 {
		return []string{m.headerLine(r.W)}
	}

	if r.H == 2 {
		return []string{m.headerLine(r.W), m.rule(r.W)}
	}

	out := []string{"", m.headerLine(r.W), m.rule(r.W)}

	return fill(out, r.H)
}

// bandRows is the activity band: a rule on top separating it from the body,
// the sentence itself, and a rule on the bottom separating it from the key bar.
func (m Model) bandRows() []string {
	r := m.frame.Band
	if r.H <= 0 {
		return nil
	}

	switch r.H {
	case 1:
		return []string{m.bandLine(r.W)}
	case 2:
		return []string{m.rule(r.W), m.bandLine(r.W)}
	default:
		return fill([]string{m.rule(r.W), m.bandLine(r.W), m.rule(r.W)}, r.H)
	}
}

// barRows is the key bar, which is one row and has never wanted a second —
// unless the palette is up, when the row is the palette's line instead. A
// second row provides bottom padding before the terminal bottom edge.
func (m Model) barRows() []string {
	r := m.frame.Bar
	if r.H <= 0 {
		return nil
	}

	line := m.barLine(r.W)
	if m.palette.open {
		line = m.paletteInputLine(r.W)
	}

	if r.H == 1 {
		return []string{line}
	}

	out := []string{line, ""}

	return fill(out, r.H)
}

// bodyRows is the list, the window it is seen through, and the sentence that
// replaces it when there is nothing to list.
func (m Model) bodyRows() []string {
	h, w := m.frame.Body.H, m.frame.Body.W
	if h <= 0 {
		return nil
	}

	all := m.rows()
	if len(all) == 0 {
		return m.emptyRows(h, w)
	}

	out := make([]string, 0, h)
	out = append(out, m.tableHeader(w))

	shown := page(h-1, len(all), m.offset)
	for i := m.offset; i < len(all) && len(out) < shown+1; i++ {
		out = append(out, m.bodyRow(all[i], i, w))
	}

	if hidden := len(all) - m.offset - (len(out) - 1); hidden > 0 {
		more := m.opts.Words.P("body.more", hidden, "… and {n} more", "… and {n} more",
			about("n", strconv.Itoa(hidden)))
		out = append(out, fit(strings.Repeat(" ", gutter)+Paint(Dim).Render(more), w))
	}

	return fill(out, h)
}

// fill pads a region out to the number of rows it was given, and cuts it to
// that number if a caller overshot. Every region goes through it, so the
// claim "the frame is exactly h rows" holds by construction rather than by
// four separate arguments.
func fill(lines []string, h int) []string {
	for len(lines) < h {
		lines = append(lines, "")
	}

	if len(lines) > h {
		return lines[:max(h, 0)]
	}

	return lines
}

// fit cuts one line to w cells, counting cells and never bytes.
//
// The tail is an ellipsis rather than nothing, because a line that was cut
// and a line that happened to end there are two different facts and a reader
// deciding whether to widen the terminal needs to tell them apart.
func fit(text string, w int) string {
	if w <= 0 {
		return ""
	}

	if lipgloss.Width(text) <= w {
		return text
	}

	return ansi.Truncate(text, w, "…")
}

// spread puts one thing at the left of a line and another at the right, with
// at least one space between them, and gives the right-hand one up entirely
// when they will not both fit.
//
// Dropping it whole is the point. A right-hand hint truncated to "unread cap
// reac…" costs the reader the number, which was the only part of it worth
// the cells.
func spread(left, right string, w int) string {
	if right == "" {
		return fit(left, w)
	}

	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return fit(left, w)
	}

	return left + strings.Repeat(" ", gap) + right
}
