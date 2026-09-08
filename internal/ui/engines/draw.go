package engines

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/roster"
	"github.com/e1i0r/orbit/internal/ui/theme"
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

func (s State) rows(h, w int, e Env) []string {
	if h <= 0 {
		return nil
	}

	p := e.Words

	if s.showingSetup {
		out := []string{
			"",
			"  " + theme.Paint(theme.Accent).Render(p.T("engines.setup_title", "Setup Steps for {engine}",
				about("engine", s.setupEngine))),
			"",
		}

		rows := s.collectEngineRows(e)
		for _, r := range rows {
			if r.engine == s.setupEngine && len(r.setup) > 0 {
				for _, step := range r.setup {
					out = append(out, "    "+step)
				}
			}
		}

		out = append(out,
			"",
			"  "+theme.Paint(theme.Dim).Render(p.T("engines.setup_notice", "Orbit verifies setup steps but executes nothing.")),
			"",
			"  "+theme.Paint(theme.Dim).Render(p.T("engines.setup_back", "{back} back", about("back", e.Keys.Back.Help().Key))),
		)

		return cells.Fill(out, h)
	}

	// The line under the title is the advice until something is typed, and
	// then it is what was typed: the filter has to be on screen, and a
	// fourth line of chrome would come out of the list.
	said := p.T("engines.subtitle", "choose model, effort and thinking for this run")
	if s.typing || s.filter != "" {
		said = s.knobFilterLine(s.shownModels(e), e)
	}

	out := []string{
		"",
		"  " + theme.Paint(theme.Accent).Render(p.T("engines.title", "Engine & Model Knobs")),
		"  " + theme.Paint(theme.Dim).Render(said),
		"",
	}

	lines, _ := s.engineLines(w, e)

	view := engineView(h)
	for i := s.engineOffset(len(lines), view); i < len(lines) && len(out) < engineTop+view; i++ {
		out = append(out, lines[i])
	}

	waysOut := p.T("engines.ways_out",
		"{open} select · {up_down} move · {fold} fold · {filter} filter · {back} back",
		about("open", e.Keys.Open.Help().Key),
		about("up_down", e.Keys.Up.Help().Key+e.Keys.Down.Help().Key),
		about("fold", e.Keys.Sideways.Help().Key),
		about("filter", e.Keys.Filter.Help().Key),
		about("back", e.Keys.Back.Help().Key))

	if s.typing {
		waysOut = p.T("engines.ways_typing", "{open} keep it · {back} clear it · {up_down} move",
			about("open", e.Keys.Open.Help().Key),
			about("back", e.Keys.Back.Help().Key),
			about("up_down", e.Keys.Up.Help().Key+e.Keys.Down.Help().Key))
	}

	out = append(out, "", cells.Fit("  "+theme.Paint(theme.Dim).Render(waysOut), w))

	return cells.Fill(out, h)
}

// engineLines is the whole list drawn, however much taller than the screen
// it is, and which line each selectable row landed on.
//
// Both answers come out of one pass because they are the same rule: where a
// row is drawn is where a click on it lands, and two passes would be two
// places for that to be decided.
func (s State) engineLines(w int, e Env) ([]string, []int) {
	rows := s.collectEngineRows(e)
	idxs := selectableEngineIndices(rows)

	current := -1
	if s.sel >= 0 && s.sel < len(idxs) {
		current = idxs[s.sel]
	}

	var (
		lines []string
		at    []int
	)

	for i, r := range rows {
		if r.kind == rowHeader {
			lines = append(lines, "", "  "+theme.Paint(theme.Accent).Render(r.title))
			continue
		}

		at = append(at, len(lines))
		lines = append(lines, s.engineLine(r, i == current, w, e))
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
func (s State) engineLine(r engineRow, marked bool, w int, e Env) string {
	mark := strings.Repeat(" ", cells.Gutter)
	if marked {
		mark = cells.Mark + strings.Repeat(" ", cells.Gutter-1)
	}

	ink := func(paint func(...string) string, s string) string {
		if marked {
			return s
		}

		return paint(s)
	}

	text := r.title
	if r.kind == rowEngine {
		text = ink(theme.Text(theme.Tertiary).Render, cells.Fold(r.open)) + text
	}

	if r.selected {
		text += " " + ink(theme.Paint(theme.OK).Render, "●")
	}

	if r.kind == rowEngine && !r.open {
		text += " " + ink(theme.Paint(theme.Dim).Render, s.shutEngineNote(r, e))
	}

	if r.disabled {
		text = ink(theme.Paint(theme.Dim).Render, text)
	}

	if note := s.engineQuota(r, e); note != "" {
		text += "   " + ink(theme.Paint(theme.Dim).Render, note)
	}

	line := cells.Fit(mark+text, w)
	if !marked {
		return line
	}

	// Out to the edge, so the row the cursor is on is a band across the
	// screen and not a word with a colour behind it.
	return theme.Paint(theme.Sel).Render(line + strings.Repeat(" ", max(0, w-lipgloss.Width(line))))
}

// shutEngineNote is what a folded engine says it is holding: the model it
// would run with, and how many there are to choose from.
func (s State) shutEngineNote(r engineRow, e Env) string {
	count := e.Words.T("engines.model_count", "{count} models",
		about("count", fmt.Sprint(r.models)))

	if r.chosen == "" {
		return count
	}

	return r.chosen + cells.Dot + count
}

// engineView is how many rows of the list the screen has room for, which is
// everything the chrome above and below it did not take.
func engineView(h int) int {
	return max(1, h-engineTop-engineFoot)
}

// engineOffset is the first line on show, held inside the list it is
// scrolling: a list that got shorter — an engine collapsed, a model chosen —
// cannot leave the view parked past its end.
func (s State) engineOffset(lines, view int) int {
	return min(max(s.offset, 0), max(0, lines-view))
}

// keepEngineRowSeen moves the list as little as it can to keep the selection
// on screen, which is the palette's rule and for the palette's reason: the
// reader is moving a cursor, and the scrolling is this screen's business
// rather than theirs.
func (s State) keepEngineRowSeen(e Env) State {
	lines, at := s.engineLines(e.Frame.Body.W, e)

	view := engineView(e.Frame.Body.H)

	if s.sel < 0 || s.sel >= len(at) {
		s.offset = 0
		return s
	}

	line := at[s.sel]
	off := s.engineOffset(len(lines), view)

	switch {
	case line < off:
		// A row under a section head is shown with it: the head says what
		// the row is one of, and "sonnet" alone at the top of the screen
		// does not say whether it is a model or an effort.
		off = max(0, line-2)
	case line >= off+view:
		off = line - view + 1
	}

	s.offset = min(max(off, 0), max(0, len(lines)-view))

	return s
}

func (s State) hit(x, y int, e Env) point.Target {
	line, ok := e.Frame.BodyRow(y)
	if !ok {
		return point.Target{}
	}

	lines, at := s.engineLines(e.Frame.Body.W, e)

	want := line - engineTop + s.engineOffset(len(lines), engineView(e.Frame.Body.H))
	for i, l := range at {
		if l == want {
			return point.Target{Kind: point.EngineRow, Pane: i}
		}
	}

	return point.Target{}
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
func (s State) engineQuota(r engineRow, e Env) string {
	if e.Quota == nil || r.kind != rowEngine || r.disabled {
		return ""
	}

	reading := e.Quota(r.engine)
	if used := roster.Spent(e.Words, reading); used != "" {
		return used
	}

	// No window to report, which the quota screen says in the same words:
	// per token, silent, or nowhere to look at all.
	return roster.Quiet(e.Words, reading)
}
