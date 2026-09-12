package menu

// One row of the menu, and how far the list has scrolled.

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// row lays one entry out: glyph, name padded to the column, description —
// or, where it cannot be done here, the reason instead of the description.
// The pieces mirror the palette's, because a reader who learned to read
// one list should be able to read the other.
//
// The selected row wears one paint and no inner ones: accents stay dark
// on the selection ground, which is how a hovered row read as a blank
// bar with the cursor on it.
func row(e Entry, selected bool, w, nameW int) string {
	if e.Head {
		return headRow(e, w)
	}

	var head, tail string

	if selected {
		head = cells.Pad(left(e), nameW, false)
		tail = plainTail(e)
	} else {
		if e.Glyph != "" {
			head = theme.Paint(theme.Accent).Render(e.Glyph) + "  " + e.Title
		} else {
			head = "   " + e.Title
		}

		head = cells.Pad(head, nameW, false)

		switch {
		case e.Reason != "":
			tail = cells.Dot + theme.Paint(theme.Dim).Render(" "+e.Reason)
		case e.Detail != "":
			tail = cells.Dot + theme.Paint(theme.Dim).Render(" "+e.Detail)
		}
	}

	line := head + tail

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

// left is the row before its description: glyph, name, nothing else.
func left(e Entry) string {
	if e.Glyph != "" {
		return e.Glyph + "  " + e.Title
	}

	return "   " + e.Title
}

// plainTail is the description with no paint on it, for the selected row.
func plainTail(e Entry) string {
	switch {
	case e.Reason != "":
		return cells.Dot + " " + e.Reason
	case e.Detail != "":
		return cells.Dot + " " + e.Detail
	}

	return ""
}

// nameWidth is the name column: every description starts under the same
// dot, no matter how long the verb before it is. Measured over the whole
// list rather than the window, so the column does not breathe while
// scrolling.
func nameWidth(es []Entry) int {
	w := 0

	for _, e := range es {
		if e.Head {
			continue
		}

		if n := lipgloss.Width(left(e)); n > w {
			w = n
		}
	}

	return w
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
