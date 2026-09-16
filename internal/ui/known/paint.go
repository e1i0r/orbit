package known

// How one row of the form is painted: its mark, its label, its options, and
// the well the sentence is written in.

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// mark is the two cells a row opens with: the cursor, or nothing.
func (s State) mark(r aRow) string {
	if r.which == s.field {
		return theme.Paint(theme.Accent).Bold(true).Render("▸ ")
	}

	return "  "
}

// label is one row's name, in the column every row's name is in.
func label(text string, on bool) string {
	if on {
		return theme.Paint(theme.Accent).Bold(true).Render(cells.Pad(text, labelWidth, false)) + " "
	}

	return theme.Paint(theme.Dim).Render(cells.Pad(text, labelWidth, false)) + " "
}

// value is what a row is holding, drawn as its options with the one in force
// lit — or, for a row that offers none, as what has been typed into it.
func (s State) value(r aRow, cw int, e Env) string {
	if len(r.options) == 0 {
		return s.typedValue(r, cw-labelWidth-3)
	}

	held, found := s.held(r), false

	var out []string

	for _, o := range r.options {
		if o.value == held {
			found = true

			out = append(out, theme.Paint(theme.Sel).Render(" "+o.label+" "))

			continue
		}

		out = append(out, theme.Paint(theme.Dim).Render(o.label))
	}

	// What somebody typed that none of the options offers is a pill of its
	// own at the end, lit: a row whose value is on screen nowhere reads as
	// a row holding nothing.
	if !found && held != "" {
		out = append(out, theme.Paint(theme.Sel).Render(" "+held+" "))
	}

	if r.which == s.field {
		out = append(out, theme.Paint(theme.Dim).Render(
			e.Words.T("knowledge.or_type", "· or type one")))
	}

	return strings.Join(out, " ")
}

// typedValue is what a line holds, with the caret in it while it is the line
// being typed into, and the word for empty when it holds nothing.
//
// A field that drew nothing at all reads as broken. Somebody who has tabbed
// past the check without typing needs to see that it is empty on purpose.
func (s State) typedValue(r aRow, room int) string {
	in := s.in[r.typed]

	if r.which == s.field {
		if lipgloss.Width(in.Val) < room {
			return theme.Paint(theme.Accent).Render(withCaret(in.Val, in.At))
		}

		return theme.Paint(theme.Accent).Render(cells.Tail(in.Val, max(room-1, 1)) + "█")
	}

	if in.Val == "" {
		return theme.Paint(theme.Dim).Render("(" + "empty" + ")")
	}

	return theme.Text(theme.Primary).Render(cells.Fit(in.Val, room))
}

// well is the bordered box the sentence is written in.
func (s State) well(cw int, e Env) []string {
	inner := max(cw-8, 12)
	line := theme.Paint(theme.Dim)

	held := s.in[factPhrase]

	body := cells.Lines(held.Val, inner)
	if held.Val == "" {
		body = []string{theme.Paint(theme.Dim).Render(e.Words.T("knowledge.phrase_placeholder",
			"(write the rule here)"))}
	}

	if s.field == rowPhrase {
		body = cells.Lines(withCaret(held.Val, held.At), inner)
	}

	out := []string{"    " + line.Render("┌"+strings.Repeat("─", inner+2)+"┐")}

	for len(body) < boxRows {
		body = append(body, "")
	}

	for _, one := range body {
		out = append(out, "    "+line.Render("│ ")+
			cells.PadRight(theme.Text(theme.Primary).Render(one), inner)+line.Render(" │"))
	}

	return append(out, "    "+line.Render("└"+strings.Repeat("─", inner+2)+"┘"))
}

// button is a row that is done rather than filled in.
func button(text string, on bool) string {
	paper := theme.PillRest
	if strings.HasPrefix(text, "✔") {
		paper = theme.PillSave
	}

	if on {
		return theme.Pill(" "+text+" ", theme.PillInk, paper)
	}

	return theme.Pill(" "+text+" ", theme.PillInkRest, theme.PillRest)
}
