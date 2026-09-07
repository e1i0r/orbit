package menu

// The menu drawn, and what a click lands on.

import (
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/words"
)

// TitleRows is how far down the first entry is drawn: the title and the
// blank under it. View writes them and Hit counts past them, and a click
// landing one row off is what happens when only one of the two knows.
const TitleRows = 2

// View draws the menu in the body: a line saying which menu this is, then
// one row per entry, the reason on the line for anything greyed, nothing
// hidden.
func (s State) View(h, w int, e Env) []string {
	if h <= 0 {
		return nil
	}

	es := s.entries(e)
	if len(es) == 0 {
		gone := e.Words.T("menu.gone", "the task this menu was opened on is no longer on the board")

		return cells.Fill([]string{"", cells.Fit("  "+theme.Paint(theme.Dim).Render(gone), w)}, h)
	}

	out := make([]string, 0, h)
	out = append(out, cells.Fit("  "+theme.Paint(theme.Dim).Render(s.Title(e)), w), "")

	off := s.offsetIn(len(es), view(h))
	for i, entry := range es[off:] {
		if len(out) >= h {
			break
		}

		out = append(out, row(entry, off+i == s.sel, w))
	}

	return cells.Fill(out, h)
}

// Title names the menu that is up: the two open on the same keystroke and
// answer different questions, and a reader who meant the task's and got the
// board's should not have to work that out from a missing verb.
func (s State) Title(e Env) string {
	p := e.Words

	switch {
	case e.Detail:
		return p.T("menu.title_detail", "{id} — where to look, and what can be done",
			about("id", s.task))
	case s.task == "":
		return p.T("menu.title_board", "commands — nothing here is about one task")
	}

	return p.T("menu.title_task", "{id} — what can be done to this task", about("id", s.task))
}

// about names a value and what the sentence calls it.
func about(name, value string) words.Arg { return words.Arg{Name: name, Value: value} }

// Hit answers the body while the menu is up: each entry is a row, and the
// rows past the list are nothing.
//
// The target carries what identifies the entry — its glyph for a verb, its
// name for a command — because the list is recomputed between press and
// release and an index could point at a different row by then.
func (s State) Hit(x, y int, e Env) point.Target {
	line, ok := e.Frame.BodyRow(y)
	if !ok {
		return point.Target{}
	}

	// Past the title, which is drawn above the entries and is not one.
	line -= TitleRows
	if line < 0 {
		return point.Target{}
	}

	// Down by whatever the list has scrolled: the drawing and the counting
	// read the same offset, or a click lands on the row the reader is not
	// looking at.
	es := s.entries(e)

	line += s.offsetIn(len(es), view(e.Frame.Body.H))
	if line >= len(es) {
		return point.Target{}
	}

	if id := ident(es[line]); id != "" {
		return point.Target{Kind: point.MenuEntry, Key: id}
	}

	return point.Target{}
}

// ident is what identifies an entry between a press and the release that
// chooses it: the glyph for a verb or a pane, the name for a command, and
// nothing at all for a heading.
func ident(e Entry) string {
	switch {
	case e.Head:
		return ""
	case e.Glyph != "":
		return e.Glyph
	}

	return e.Command
}
