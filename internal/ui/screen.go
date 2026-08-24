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
		case screenHelp:
			lines = append(lines, m.helpRows(m.frame.Body.H, m.frame.Body.W)...)
		default:
			lines = append(lines, m.bodyRows()...)
		}
	}
	lines = append(lines, m.bandRows()...)
	lines = append(lines, m.barRows()...)
	v := tea.NewView(strings.Join(lines, "\n"))
	// CellMotion and not AllMotion: the window is told about a moved
	// pointer only while a button is down, which is what a drag needs and
	// all it needs. AllMotion is a message for every cell the pointer
	// crosses whether or not anything is being done with it, and it buys
	// nothing until a row is drawn differently for being hovered over.
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// headerRows is the header region: the line, then its rule.
func (m Model) headerRows() []string {
	r := m.frame.Header
	if r.H <= 0 {
		return nil
	}
	out := []string{m.headerLine(r.W)}
	if r.H > 1 {
		out = append(out, m.rule(r.W))
	}
	return fill(out, r.H)
}

// bandRows is the activity band: its rule, then the sentence. The rule goes
// first here and second in the header because both rules sit under the
// region that owns them.
func (m Model) bandRows() []string {
	r := m.frame.Band
	if r.H <= 0 {
		return nil
	}
	if r.H == 1 {
		return []string{m.bandLine(r.W)}
	}
	return fill([]string{m.rule(r.W), m.bandLine(r.W)}, r.H)
}

// barRows is the key bar, which is one row and has never wanted a second —
// unless the palette is up, when the row is the palette's line instead. The
// hints it replaces belong to a keyboard the palette is holding.
func (m Model) barRows() []string {
	r := m.frame.Bar
	if r.H <= 0 {
		return nil
	}
	if m.palette.open {
		return fill([]string{m.paletteInputLine(r.W)}, r.H)
	}
	return fill([]string{m.barLine(r.W)}, r.H)
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
	shown := page(h, len(all), m.offset)
	for i := m.offset; i < len(all) && len(out) < shown; i++ {
		out = append(out, m.bodyRow(all[i], i, w))
	}
	if hidden := len(all) - m.offset - len(out); hidden > 0 {
		more := m.opts.Words.P("body.more", hidden, "… and {n} more", "… and {n} more",
			about("n", strconv.Itoa(hidden)))
		out = append(out, fit(strings.Repeat(" ", gutter)+Paint(Dim).Render(more), w))
	}
	return fill(out, h)
}

// page is how many rows of the list the body can show at one time.
//
// It is one less than the region's height whenever there is more list below
// the fold, and the row it gives up is spent saying so. A list that simply
// stops at the bottom of the screen is the specific bug the plan calls out:
// there is nothing on screen to distinguish "these are all of them" from
// "there are nine more".
func page(h, rows, offset int) int {
	if h <= 0 {
		return 0
	}
	if offset+h >= rows {
		return h
	}
	return h - 1
}

// bodyRow draws one line of the list, whichever of the three kinds it is.
func (m Model) bodyRow(r row, i, w int) string {
	switch {
	case r.blank:
		return ""
	case r.head:
		return m.headRow(r, i == m.cursor, w)
	}
	return m.drawRow(r, w, i == m.cursor)
}

// emptyRows is the body with nothing in it, and it says which kind of
// nothing.
//
// There are three, and they want three different next moves: clear the
// filter, add a repository, write a task. The program this replaces printed
// one word for all three, and the reader's next move after reading it was to
// go looking for a task they were certain they had written.
func (m Model) emptyRows(h, w int) []string {
	p := m.opts.Words
	var lines []string
	switch typed := strings.TrimSpace(m.filter); {
	case typed != "":
		lines = []string{
			p.T("empty.filter", "Nothing matches {filter}.", about("filter", typed)), "",
			p.T("empty.clear_filter", "Press esc to clear the filter and see everything again."),
		}
	case m.board.Repos == 0:
		lines = []string{
			p.T("empty.no_repos", "No repositories under {root} yet.", about("root", m.opts.Root)), "",
			p.T("empty.add_repo", "Clone one into {root}, or start orbit in a folder that already has one.",
				about("root", m.opts.Root)),
		}
	default:
		lines = []string{
			p.T("empty.needs_you", "Nothing needs you."), "",
			p.P("empty.no_tasks", m.board.Repos,
				"{n} repository, and no tasks written against it yet.",
				"{n} repositories, and no tasks written against any of them yet.",
				about("n", strconv.Itoa(m.board.Repos))),
			p.T("empty.write_one", "Write one with `orbit new <id>`, then press n to start it."),
		}
	}
	out := []string{""}
	for i, line := range lines {
		if line == "" {
			out = append(out, "")
			continue
		}
		role := Dim
		if i == 0 {
			role = Accent
		}
		out = append(out, fit("  "+Paint(role).Render(line), w))
	}
	return fill(out, h)
}

// refusal is what a terminal narrower than the minimum gets instead of a
// crooked table: one sentence, both numbers, and the rest of the rows blank.
func (m Model) refusal() string {
	p := m.opts.Words
	w := max(m.width, 1)
	out := []string{
		fit(Paint(Warn).Render(p.T("narrow.refused", "orbit needs {need} columns.",
			about("need", strconv.Itoa(m.narrow.Need)))), w),
		fit(Paint(Dim).Render(p.T("narrow.got", "this one has {got}.",
			about("got", strconv.Itoa(m.narrow.Got)))), w),
	}
	return strings.Join(fill(out, max(m.height, 1)), "\n")
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
