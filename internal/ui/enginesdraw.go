package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
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

	// The line under the title is the advice until something is typed, and
	// then it is what was typed: the filter has to be on screen, and a
	// fourth line of chrome would come out of the list.
	said := p.T("engines.subtitle", "choose model, effort and thinking for this run")
	if m.engines.typing || m.engines.filter != "" {
		said = m.knobFilterLine(m.shownModels())
	}

	out := []string{
		"",
		"  " + Paint(Accent).Render(p.T("engines.title", "Engine & Model Knobs")),
		"  " + Paint(Dim).Render(said),
		"",
	}

	lines, _ := m.engineLines(w)

	view := engineView(h)
	for i := m.engineOffset(len(lines), view); i < len(lines) && len(out) < engineTop+view; i++ {
		out = append(out, lines[i])
	}

	waysOut := p.T("engines.ways_out",
		"{open} select · {up_down} move · {fold} fold · {filter} filter · {back} back",
		about("open", m.keys.Open.Help().Key),
		about("up_down", m.keys.Up.Help().Key+m.keys.Down.Help().Key),
		about("fold", m.keys.Sideways.Help().Key),
		about("filter", m.keys.Filter.Help().Key),
		about("back", m.keys.Back.Help().Key))

	if m.engines.typing {
		waysOut = p.T("engines.ways_typing", "{open} keep it · {back} clear it · {up_down} move",
			about("open", m.keys.Open.Help().Key),
			about("back", m.keys.Back.Help().Key),
			about("up_down", m.keys.Up.Help().Key+m.keys.Down.Help().Key))
	}

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
		lines = append(lines, m.engineLine(r, i == current, w))
	}

	return lines, at
}

// engineLine is one row: the cursor's gutter, what the row is called, the
// dot on the one in force, and whatever the quota has to say about it.
//
// The chosen row is painted across its whole width rather than marked only
// in the gutter. This list is read by running an eye down it, and a cursor
// that is one glyph three columns to the left of a name is the thing the eye
// was not looking at.
//
// Nothing inside that row carries a colour of its own: a colour ends with
// its own reset, and a reset halfway along the line would end the highlight
// halfway along the name it is marking.
func (m Model) engineLine(r engineRow, marked bool, w int) string {
	mark := strings.Repeat(" ", gutter)
	if marked {
		mark = markGlyph + strings.Repeat(" ", gutter-1)
	}

	ink := func(paint func(...string) string, s string) string {
		if marked {
			return s
		}

		return paint(s)
	}

	text := r.title
	if r.kind == rowEngine {
		text = ink(Text(Tertiary).Render, foldMark(r.open)) + text
	}

	if r.selected {
		text += " " + ink(Paint(OK).Render, "●")
	}

	if r.kind == rowEngine && !r.open {
		text += " " + ink(Paint(Dim).Render, m.shutEngineNote(r))
	}

	if r.disabled {
		text = ink(Paint(Dim).Render, text)
	}

	if note := m.engineQuota(r); note != "" {
		text += "   " + ink(Paint(Dim).Render, note)
	}

	line := fit(mark+text, w)
	if !marked {
		return line
	}

	// Out to the edge, so the row the cursor is on is a band across the
	// screen and not a word with a colour behind it.
	return Paint(Sel).Render(line + strings.Repeat(" ", max(0, w-lipgloss.Width(line))))
}

// shutEngineNote is what a folded engine says it is holding: the model it
// would run with, and how many there are to choose from.
func (m Model) shutEngineNote(r engineRow) string {
	count := m.opts.Words.T("engines.model_count", "{count} models",
		about("count", fmt.Sprint(r.models)))

	if r.chosen == "" {
		return count
	}

	return r.chosen + dot + count
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
