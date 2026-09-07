package markdown

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// Markdown, set the way the rest of the panes are set.
//
// What the engine writes is the only thing on these panes a person reads
// end to end, so it gets the typography the prose gets: a measure it is cut
// to, full contrast on the words, and the furniture — rails, bullets,
// numbers, rules — a rung quieter than the sentence they carry.

// mdMeasure is how wide a line of markdown is set: the pane's width less the
// indent, and never past the measure the eye tracks. A report set to a
// terminal's full 140 cells loses the reader at every line break.
// Indent is the margin every rendered row carries, and Measure the width a
// paragraph is set to: the eye tracks a line of about this many cells, and a
// report set to a terminal's full 140 loses the reader at every break.
const (
	Indent  = "    "
	Measure = 84
)

// quoteMark is what the engine's own words are set behind.
const quoteMark = "│ "

func mdMeasure(width int) int {
	return max(20, min(Measure, width-lipgloss.Width(Indent)-2))
}

// Render renders a Markdown string into terminal-styled rows.
// If raw is true, it returns the source behind a rail, unchanged.
// If raw is false, it renders headings, code blocks, lists, quotes and rules.
func Render(text string, width int, raw bool) []string {
	if text == "" {
		return nil
	}

	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")

	if raw {
		out := make([]string, 0, len(lines))
		for _, l := range lines {
			out = append(out, Indent+theme.Text(theme.Tertiary).Render(quoteMark)+theme.Text(theme.Primary).Render(l))
		}

		return out
	}

	w := mdMeasure(width)

	var (
		out    []string
		coding bool
		family string
	)

	for _, l := range lines {
		trimmed := strings.TrimSpace(l)

		if lang, fence := strings.CutPrefix(trimmed, "```"); fence {
			coding = !coding

			// The language is the opening fence's own, whether it named one
			// or not: a block that fell back to the last block's syntax
			// would colour plain output as somebody's Go.
			if coding {
				family = theme.CodeFamily(lang)
			}

			if coding && lang != "" {
				out = append(out, Indent+theme.Text(theme.Tertiary).Render(strings.ToUpper(lang)))
			}

			continue
		}

		if coding {
			out = append(out, Indent+theme.Text(theme.Tertiary).Render(quoteMark)+Well(l, family, w))

			continue
		}

		out = append(out, markdownLine(trimmed, l, w)...)
	}

	return out
}

// codeTab is what a tab is drawn as inside a well. Four columns: code in a
// pane has 84 of them, and eight spends a tenth of the line on one level.
const codeTab = "    "

// Well sets one line of a fenced block: the sunken paper, filled to the
// measure so the block reads as one shape rather than as lines of ragged
// length, and each run of the line in the colour syntax.go decided on.
//
// Every run carries the well's paper itself. A style rendered inside another
// closes with a reset, so a token painted on top of one background would take
// the rest of the row back to the window's own paper — which is the same
// reason the padding is a piece of the well and not a suffix on it.
func Well(line, family string, w int) string {
	// A tab is one character and four columns, so a well padded before the
	// terminal expands them comes out one step wider per level of indent.
	flat := cells.Pad(strings.ReplaceAll(line, "\t", codeTab), w, false)

	well := theme.Surface(theme.Sunken)

	var b strings.Builder

	for _, tok := range theme.LexCode(flat, family) {
		ink := theme.Text(theme.Primary).GetForeground()
		if role, painted := theme.CodeRole(tok.Part); painted {
			ink = theme.Paint(role).GetForeground()
		}

		b.WriteString(well.Foreground(ink).Render(tok.Text))
	}

	return b.String()
}

// markdownLine is everything outside a fence: one source line, and the rows
// it is drawn as.
//
// It takes the line twice because the block markers are found on the trimmed
// line while a paragraph keeps the indent it was written with.
func markdownLine(trimmed, raw string, w int) []string {
	if head, ok := headingRow(trimmed, w); ok {
		return head
	}

	if quote, ok := strings.CutPrefix(trimmed, "> "); ok {
		out := make([]string, 0, 2)
		for _, wl := range cells.Lines(Inline(quote), w-lipgloss.Width(quoteMark)) {
			out = append(out, Indent+theme.Text(theme.Tertiary).Render(quoteMark)+wl)
		}

		return out
	}

	switch trimmed {
	case "---", "***", "___":
		return []string{Indent + theme.Text(theme.Tertiary).Render(strings.Repeat("─", w))}
	case "":
		return []string{""}
	}

	if item, ok := listItem(trimmed); ok {
		return hung(item.mark, item.paint, item.text, w)
	}

	return hung("", lipgloss.NewStyle(), raw, w)
}

// headingRow sets the three heading levels as three rungs of the same
// ladder the rest of the pane uses: the accent for the title of a document,
// then full contrast, then the tone that qualifies. A heading also opens a
// blank line above it, which is what says a block ended.
func headingRow(trimmed string, w int) ([]string, bool) {
	for _, h := range []struct {
		prefix string
		style  lipgloss.Style
	}{
		{"# ", theme.Paint(theme.Accent).Bold(true)},
		{"## ", theme.Text(theme.Primary).Bold(true)},
		{"### ", theme.Text(theme.Secondary).Bold(true)},
	} {
		title, ok := strings.CutPrefix(trimmed, h.prefix)
		if !ok {
			continue
		}

		return []string{"", Indent + h.style.Render(cells.Fit(title, w))}, true
	}

	return nil, false
}

// bullet is one item of a list: what stands in front of it, in what, and
// what it says.
type bullet struct {
	mark  string
	paint lipgloss.Style
	text  string
}

// listItem reads the four kinds of item, in the order that tells them
// apart: a checklist is a bullet list whose item opens with a box, so it has
// to be read first.
func listItem(trimmed string) (bullet, bool) {
	for _, open := range []string{"- ", "* "} {
		text, ok := strings.CutPrefix(trimmed, open)
		if !ok {
			continue
		}

		if done, box := strings.CutPrefix(text, "[x] "); box {
			return bullet{"✔ ", theme.Paint(theme.OK), done}, true
		}

		if todo, box := strings.CutPrefix(text, "[ ] "); box {
			return bullet{"☐ ", theme.Text(theme.Tertiary), todo}, true
		}

		return bullet{"• ", theme.Text(theme.Tertiary), text}, true
	}

	if n, rest, found := strings.Cut(trimmed, ". "); found && isNumber(n) {
		return bullet{n + ". ", theme.Text(theme.Tertiary), rest}, true
	}

	return bullet{}, false
}

// isNumber reports whether s is one or more digits and nothing else, which
// is what separates a numbered item from a sentence with a full stop in it.
func isNumber(s string) bool {
	if s == "" {
		return false
	}

	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}

// hung wraps text under a marker, hanging: the marker stands on the first
// row and the rows under it start where its text did. Repeating the marker
// on every wrapped row is what turned one three-line item into three items.
func hung(mark string, paint lipgloss.Style, text string, w int) []string {
	lead := lipgloss.Width(mark)
	wrapped := cells.Lines(Inline(text), max(20, w-lead))

	out := make([]string, 0, len(wrapped))

	for i, wl := range wrapped {
		if i == 0 {
			out = append(out, Indent+paint.Render(mark)+wl)
			continue
		}

		out = append(out, Indent+strings.Repeat(" ", lead)+wl)
	}

	return out
}
