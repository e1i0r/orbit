package palette

// What the line and its list are drawn as, and what the pointer reads back
// out of them. The two walk the same offset and the same filtered list,
// which is the whole deal the hit map makes: point and draw are two readings
// of one set of numbers, never two sets.

import (
	"strings"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// View is the body while the line is up: the candidates, oldest table order
// first, cut to the rows the region has and scrolled by whatever
// ensureVisible last had to move.
func (s State) View(h, w int, e Env) []string {
	if h <= 0 {
		return nil
	}

	all := s.candidates(e.Commands)
	if len(all) == 0 {
		line := theme.Paint(theme.Dim).Render(e.Words.T("palette.none",
			"no command starts with {typed}", about("typed", s.typed)))

		return cells.Fill([]string{"", cells.Fit("  "+line, w)}, h)
	}

	out := make([]string, 0, h)
	for i := s.offset; i < len(all) && len(out) < h; i++ {
		out = append(out, row(all[i], i == s.sel, w))
	}

	return cells.Fill(out, h)
}

// row draws one candidate: the name, the usage fragment when it has
// one, and either the description or — for a command the window refuses —
// the reason, on the same line, greyed.
//
// A refusal replaces the description rather than joining it, because the
// reason is the part a reader acts on and a line that carries both is a
// line that gets truncated to lose whichever mattered.
func row(c Command, selected bool, w int) string {
	tail := ""
	if c.Refused && c.Because != "" {
		tail = theme.Paint(theme.Dim).Render(cells.Dot + " " + c.Because)
	} else {
		if c.Args != "" {
			tail += theme.Paint(theme.Dim).Render(" " + c.Args)
		}

		if c.About != "" {
			tail += theme.Paint(theme.Dim).Render(cells.Dot + " " + c.About)
		}
	}

	line := "  " + c.Name + tail

	mark := strings.Repeat(" ", cells.Gutter)
	if selected {
		mark = cells.Mark + strings.Repeat(" ", cells.Gutter-1)
		line = cells.Fit(mark+line, w)

		return theme.Paint(theme.Sel).Render(line)
	}

	return cells.Fit(mark+c.Name+tail, w)
}

// Line is the line itself, where the key bar sits while the
// palette owns the keyboard. A block after the text is the caret: one cell,
// borrowed from the cursor's own paint, gone the moment the line goes down.
func (s State) Line(w int, e Env) string {
	if s.typed == "" {
		placeholder := theme.Paint(theme.Dim).Render(": " + e.Words.T("palette.placeholder", "type a command"))

		return cells.Fit(" "+placeholder, w)
	}

	return cells.Fit(" :"+s.typed+theme.Paint(theme.Sel).Render(" "), w)
}

// Hit answers the body's cells while the palette is up: each
// candidate is a row, and the rows past the list are nothing.
//
// It walks the same offset and the same filtered list the renderer drew
// from, which is the whole deal the hit map makes: point and draw are two
// readings of one set of numbers, never two sets.
func (s State) Hit(x, y int, e Env) point.Target {
	line, ok := e.Frame.BodyRow(y)
	if !ok {
		return point.Target{}
	}

	all := s.candidates(e.Commands)

	i := s.offset + line
	if line >= e.Frame.Body.H || i < 0 || i >= len(all) {
		return point.Target{}
	}

	return point.Target{Kind: point.Command, Key: all[i].Name}
}
