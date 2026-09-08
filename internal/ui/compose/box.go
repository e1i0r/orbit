package compose

// The box a field is written into: the label on its top border, the paste
// button at the end of that row, and what the field holds wrapped between
// the borders.
//
// Two fields are drawn this way. A task is written in sentences, and a
// tracker URL is longer than a row of the form has room for: both are read
// wrapped or not read at all.

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/ui/typing"
	"github.com/e1i0r/orbit/internal/words"
)

func (s State) composeBox(field int, label, placeholder, hint string, in typing.Field, w int, e Env) []string {
	p := e.Words
	active := s.field == field

	boxW, innerW := s.composeBoxWidth(w, e), s.composeInnerWidth(w, e)

	// A cell narrower while the caret is drawn, because the caret is a cell
	// of its own in front of the placeholder. Fitted to the whole inner
	// width, a focused empty box measured one cell more than the borders
	// above and below it, and the row stuck out at any terminal narrow
	// enough for the placeholder to reach the edge.
	ghostW := innerW
	if active {
		ghostW = max(0, innerW-1)
	}

	lines := s.composeBoxLines(in, innerW, active, theme.Paint(theme.Dim).Render(cells.Fit(placeholder, ghostW)))

	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.BoxLine))
	if active {
		borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.BoxLineActive))
	}
	// The label sits on the top border rather than on a line of its own, so
	// the box starts where every other value of the form starts and the two
	// read as one row. The paste button ends that row: its top edge is the
	// top edge of the box, so the two begin on the same line rather than
	// one of them hanging below the other.
	top := composeLabel(label, active) +
		borderStyle.Render("┌"+strings.Repeat("─", boxW-2)+"┐") +
		" " + composePasteTab(p)

	indent := strings.Repeat(" ", composeLabelStart)
	out := []string{cells.Fit(top, w)}

	for _, l := range lines {
		padW := max(innerW-lipgloss.Width(l), 0)
		out = append(out, cells.Fit(indent+borderStyle.Render("│ ")+l+strings.Repeat(" ", padW)+borderStyle.Render(" │"), w))
	}

	// The hint goes under the box because the button took the end of the
	// top row: it is about typing rather than about the field, so it reads
	// as well from below and nothing has to give up its place for it.
	bottom := indent + borderStyle.Render("└"+strings.Repeat("─", boxW-2)+"┘")
	if active && hint != "" {
		bottom += " " + theme.Paint(theme.Dim).Render(hint)
	}

	return append(out, cells.Fit(bottom, w))
}

// composeBoxLines is what is drawn between the borders: the lines the value
// wraps into, the window of them the caret is inside, and the selection and
// the caret painted over the cells they are on.
func (s State) composeBoxLines(in typing.Field, innerW int, active bool, placeholder string) []string {
	rs := in.Runes()
	spans := typing.Wrap(rs, innerW)
	caretRow := typing.SpanRow(spans, in.At)
	top := typing.SpanWindow(len(spans), composeTextRows, caretRow)
	from, to := in.Selection()

	var out []string

	for i := top; i < len(spans) && i < top+composeTextRows; i++ {
		s := spans[i]

		line := typing.SpanText(rs, s)

		if active {
			caret := -1
			if i == caretRow {
				caret = in.At - s.From
			}

			line = typing.PaintCells(line, from-s.From, to-s.From, caret, unpainted)
		}

		out = append(out, line)
	}

	if len(rs) == 0 {
		out = []string{placeholder}
		if active {
			out = []string{typing.PaintCells("", 0, 0, 0, unpainted) + placeholder}
		}
	}

	for len(out) < 3 {
		out = append(out, "")
	}

	return out
}

// composeBoxRowCount is how many lines the box of the open tab drew, asked
// of the same function that draws them, so a value long enough to wrap does
// not put every row below it one place from where it is.
func (s State) composeBoxRowCount(e Env) int {
	in := s.text
	if s.tab == composeTabURL {
		in = s.url
	}

	return len(s.composeBoxLines(in, s.composeInnerWidth(e.Frame.Body.W, e), false, ""))
}

// composePasteTab is the button that reads the clipboard into the field it
// is drawn beside, and composePasteRoom is the width it needs with the cell
// of space that separates it from that field.
func composePasteTab(p *words.Printer) string {
	return theme.Pill(" 📋 "+p.T("compose.btn_paste", "Paste (^V)")+" ", theme.PillInk, theme.PillPaste)
}

func composePasteRoom(p *words.Printer) int {
	return lipgloss.Width(composePasteTab(p)) + 1
}

// unpainted is the box: what is typed into it is drawn as it was typed.
func unpainted(s string) string { return s }
