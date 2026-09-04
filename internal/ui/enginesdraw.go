package ui

import (
	"fmt"
	"strings"
)

const (
	// engineTop is the chrome above the list — the title, the line of
	// advice under it, and the blank rows that set them off — and engineFoot
	// the blank row and the ways out below it. Neither scrolls: what the
	// keys do has to stay on screen while the reader is going down a list
	// of sixty models looking for one.
	engineTop  = 4
	engineFoot = 2
)

func (m Model) enginesRows(h, w int) []string {
	if h <= 0 {
		return nil
	}

	p := m.opts.Words

	if m.engines.showingSetup {
		out := []string{
			"",
			"  " + Paint(Accent).Render(p.T("engines.setup_title", "Setup Steps for {engine}",
				about("engine", m.engines.setupEngine))),
			"",
		}

		rows := m.collectEngineRows()
		for _, r := range rows {
			if r.engine == m.engines.setupEngine && len(r.setup) > 0 {
				for _, step := range r.setup {
					out = append(out, "    "+step)
				}
			}
		}

		out = append(out,
			"",
			"  "+Paint(Dim).Render(p.T("engines.setup_notice", "Orbit verifies setup steps but executes nothing.")),
			"",
			"  "+Paint(Dim).Render(p.T("engines.setup_back", "{back} back", about("back", m.keys.Back.Help().Key))),
		)

		return fill(out, h)
	}

	out := []string{
		"",
		"  " + Paint(Accent).Render(p.T("engines.title", "Engine & Model Knobs")),
		"  " + Paint(Dim).Render(p.T("engines.subtitle", "choose model, effort and thinking for this run")),
		"",
	}

	lines, _ := m.engineLines(w)

	view := engineView(h)
	for i := m.engineOffset(len(lines), view); i < len(lines) && len(out) < engineTop+view; i++ {
		out = append(out, lines[i])
	}

	waysOut := p.T("engines.ways_out", "{open} select · {up_down} move · {fold} fold · {back} back",
		about("open", m.keys.Open.Help().Key),
		about("up_down", m.keys.Up.Help().Key+m.keys.Down.Help().Key),
		about("fold", m.keys.Sideways.Help().Key),
		about("back", m.keys.Back.Help().Key))
	out = append(out, "", fit("  "+Paint(Dim).Render(waysOut), w))

	return fill(out, h)
}

// engineLines is the whole list drawn, however much taller than the screen
// it is, and which line each selectable row landed on.
//
// Both answers come out of one pass because they are the same rule: where a
// row is drawn is where a click on it lands, and two passes would be two
// places for that to be decided.
func (m Model) engineLines(w int) ([]string, []int) {
	rows := m.collectEngineRows()
	idxs := m.selectableEngineIndices(rows)

	current := -1
	if m.engines.sel >= 0 && m.engines.sel < len(idxs) {
		current = idxs[m.engines.sel]
	}

	var (
		lines []string
		at    []int
	)

	for i, r := range rows {
		if r.kind == rowHeader {
			lines = append(lines, "", "  "+Paint(Accent).Render(r.title))
			continue
		}

		at = append(at, len(lines))
		lines = append(lines, fit(m.engineLine(r, i == current), w))
	}

	return lines, at
}

// engineLine is one row: the cursor's gutter, what the row is called, the
// dot on the one in force, and whatever the quota has to say about it.
func (m Model) engineLine(r engineRow, marked bool) string {
	mark := strings.Repeat(" ", gutter)
	if marked {
		mark = markGlyph + strings.Repeat(" ", gutter-1)
	}

	text := r.title
	if r.kind == rowEngine {
		text = Text(Tertiary).Render(foldMark(r.open)) + text
		if !r.open && r.models > 0 {
			text += " " + Paint(Dim).Render(m.opts.Words.T("engines.model_count",
				"{count} models", about("count", fmt.Sprint(r.models))))
		}
	}

	if r.selected {
		text += " " + Paint(OK).Render("●")
	}

	if r.disabled {
		text = Paint(Dim).Render(text)
	}

	if note := m.engineQuota(r); note != "" {
		text += "   " + Paint(Dim).Render(note)
	}

	return fmt.Sprintf("%s%s", mark, text)
}

// engineView is how many rows of the list the screen has room for, which is
// everything the chrome above and below it did not take.
func engineView(h int) int {
	return max(1, h-engineTop-engineFoot)
}

// engineOffset is the first line on show, held inside the list it is
// scrolling: a list that got shorter — an engine collapsed, a model chosen —
// cannot leave the view parked past its end.
func (m Model) engineOffset(lines, view int) int {
	return min(max(m.engines.offset, 0), max(0, lines-view))
}

// keepEngineRowSeen moves the list as little as it can to keep the selection
// on screen, which is the palette's rule and for the palette's reason: the
// reader is moving a cursor, and the scrolling is this screen's business
// rather than theirs.
func (m Model) keepEngineRowSeen() Model {
	lines, at := m.engineLines(m.frame.Body.W)

	view := engineView(m.frame.Body.H)

	if m.engines.sel < 0 || m.engines.sel >= len(at) {
		m.engines.offset = 0
		return m
	}

	line := at[m.engines.sel]
	off := m.engineOffset(len(lines), view)

	switch {
	case line < off:
		// A row under a section head is shown with it: the head says what
		// the row is one of, and "sonnet" alone at the top of the screen
		// does not say whether it is a model or an effort.
		off = max(0, line-2)
	case line >= off+view:
		off = line - view + 1
	}

	m.engines.offset = min(max(off, 0), max(0, len(lines)-view))

	return m
}

func (m Model) hitEngines(x, y int) Target {
	line, ok := m.frame.BodyRow(y)
	if !ok {
		return Target{}
	}

	lines, at := m.engineLines(m.frame.Body.W)

	want := line - engineTop + m.engineOffset(len(lines), engineView(m.frame.Body.H))
	for i, l := range at {
		if l == want {
			return Target{Kind: TargetEngineRow, Pane: i}
		}
	}

	return Target{}
}

// engineQuota is what one row of this screen says about the engine it names:
// how much of its windows is gone, or that nobody can tell.
//
// It is here and not only in the header because this is the screen where the
// choice is made. The header carries the engine already running; a reader
// standing on this list is deciding which one to hand the next task to, and
// "claude is at 77% of its week" is the fact that decides it — two lines up
// is far enough away to be worth repeating here.
//
// Only engine rows carry it. A window belongs to the engine and not to the
// model: the proxy reports what each model contributed to the window, which
// is not a limit that model has, and drawn beside opus it would read as a cap
// that does not exist. A row for an engine that is not installed carries
// nothing either — what that row is about is the setup it still needs.
func (m Model) engineQuota(r engineRow) string {
	if m.opts.Quota == nil || r.kind != rowEngine || r.disabled {
		return ""
	}

	reading := m.opts.Quota(r.engine)
	if used := m.windowsUsed(reading); used != "" {
		return used
	}

	// No window to report, which the quota screen says in the same words:
	// per token, silent, or nowhere to look at all.
	return m.quotaSilence(reading)
}
