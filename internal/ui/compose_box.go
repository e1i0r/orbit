package ui

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

	"github.com/e1i0r/orbit/internal/words"
)

func (m Model) composeBox(field int, label, placeholder, hint string, in input, w int) []string {
	p := m.opts.Words
	active := m.compose.field == field

	boxW, innerW := m.composeBoxWidth(w), m.composeInnerWidth(w)
	lines := m.composeBoxLines(in, innerW, active, Paint(Dim).Render(fit(placeholder, innerW)))

	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#334155"))
	if active {
		borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#38BDF8"))
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
	out := []string{fit(top, w)}

	for _, l := range lines {
		padW := max(innerW-lipgloss.Width(l), 0)
		out = append(out, fit(indent+borderStyle.Render("│ ")+l+strings.Repeat(" ", padW)+borderStyle.Render(" │"), w))
	}

	// The hint goes under the box because the button took the end of the
	// top row: it is about typing rather than about the field, so it reads
	// as well from below and nothing has to give up its place for it.
	bottom := indent + borderStyle.Render("└"+strings.Repeat("─", boxW-2)+"┘")
	if active && hint != "" {
		bottom += " " + Paint(Dim).Render(hint)
	}

	return append(out, fit(bottom, w))
}

// composeBoxLines is what is drawn between the borders: the lines the value
// wraps into, the window of them the caret is inside, and the selection and
// the caret painted over the cells they are on.
func (m Model) composeBoxLines(in input, innerW int, active bool, placeholder string) []string {
	rs := in.runes()
	spans := wrapSpans(rs, innerW)
	caretRow := spanRow(spans, in.at)
	top := spanWindow(len(spans), composeTextRows, caretRow)
	from, to := in.selection()

	var out []string

	for i := top; i < len(spans) && i < top+composeTextRows; i++ {
		s := spans[i]

		line := spanText(rs, s)

		if active {
			caret := -1
			if i == caretRow {
				caret = in.at - s.from
			}

			line = paintCells(line, from-s.from, to-s.from, caret, unpainted)
		}

		out = append(out, line)
	}

	if len(rs) == 0 {
		out = []string{placeholder}
		if active {
			out = []string{paintCells("", 0, 0, 0, unpainted) + placeholder}
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
func (m Model) composeBoxRowCount() int {
	in := m.compose.text
	if m.compose.tab == composeTabURL {
		in = m.compose.url
	}

	return len(m.composeBoxLines(in, m.composeInnerWidth(m.frame.Body.W), false, ""))
}

// composePasteTab is the button that reads the clipboard into the field it
// is drawn beside, and composePasteRoom is the width it needs with the cell
// of space that separates it from that field.
func composePasteTab(p *words.Printer) string {
	return Pill(" 📋 "+p.T("compose.btn_paste", "Paste (^V)")+" ", "#FFFFFF", "#0369A1")
}

func composePasteRoom(p *words.Printer) int {
	return lipgloss.Width(composePasteTab(p)) + 1
}

// unpainted is the box: what is typed into it is drawn as it was typed.
func unpainted(s string) string { return s }
