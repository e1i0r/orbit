package menu

// One row of the menu, and how far the list has scrolled.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// row lays one entry out: glyph, name, description — or, where it cannot be
// done here, the reason instead of the description. The pieces mirror the
// palette's, because a reader who learned to read one list should be able to
// read the other.
func row(e Entry, selected bool, w int) string {
	if e.Head {
		return headRow(e, w)
	}

	line := e.Title
	if e.Glyph != "" {
		line = theme.Paint(theme.Accent).Render(e.Glyph) + "  " + line
	} else {
		line = "   " + line
	}

	switch {
	case e.Reason != "":
		line += cells.Dot + theme.Paint(theme.Dim).Render(" "+e.Reason)
	case e.Detail != "":
		line += cells.Dot + theme.Paint(theme.Dim).Render(" "+e.Detail)
	}

	mark := strings.Repeat(" ", cells.Gutter)
	if selected {
		mark = cells.Mark + strings.Repeat(" ", cells.Gutter-1)
		return theme.Paint(theme.Sel).Render(cells.Fit(mark+line, w))
	}

	if e.Dim {
		return cells.Fit(mark+theme.Paint(theme.Dim).Render(line), w)
	}

	return cells.Fit(mark+line, w)
}

// headRow draws a heading: no gutter, no glyph, the accent the sections of
// the knobs screen are named in.
func headRow(e Entry, w int) string {
	if e.Title == "" {
		return ""
	}

	return cells.Fit("  "+theme.Paint(theme.Accent).Render(e.Title), w)
}

// view is how many entries fit under the title. The title and the blank
// under it do not scroll: a list that carried its own name off the top would
// leave the reader looking at rows with nothing saying what they are of.
func view(h int) int { return max(1, h-TitleRows) }

// offsetIn is the first entry drawn, clamped to a list that may have got
// shorter since it was set — the entries are recomputed on every frame, and
// a run that leaves the board takes its verbs with it.
func (s State) offsetIn(entries, rows int) int {
	return min(max(s.offset, 0), max(0, entries-rows))
}

// keepSeen moves the list as little as it can to keep the selection on
// screen. It is the palette's rule and the knobs': the reader is moving a
// cursor, and the scrolling is the screen's business rather than theirs.
func (s State) keepSeen(e Env) State {
	es := s.entries(e)

	rows := view(e.Frame.Body.H)
	off := s.offsetIn(len(es), rows)

	switch {
	case s.sel < off:
		// A heading is shown with the entry under it, for the reason the
		// heading exists: "pause" at the top of the screen does not say
		// whether it is a pane or a verb.
		s.offset = s.sel
		if s.sel > 0 && es[s.sel-1].Head {
			s.offset = s.sel - 1
		}
	case s.sel >= off+rows:
		s.offset = s.sel - rows + 1
	default:
		s.offset = off
	}

	if s.offset < 0 {
		s.offset = 0
	}

	return s
}
