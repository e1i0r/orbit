package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
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
func mdMeasure(width int) int {
	return max(20, min(proseMeasure, width-lipgloss.Width(markdownIndent)-2))
}

// renderMarkdown renders a Markdown string into terminal-styled rows.
// If raw is true, it returns the source behind a rail, unchanged.
// If raw is false, it renders headings, code blocks, lists, quotes and rules.
func renderMarkdown(text string, width int, raw bool) []string {
	if text == "" {
		return nil
	}

	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")

	if raw {
		out := make([]string, 0, len(lines))
		for _, l := range lines {
			out = append(out, markdownIndent+Text(Tertiary).Render(quoteMark)+Text(Primary).Render(l))
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
				family = codeFamily(lang)
			}

			if coding && lang != "" {
				out = append(out, markdownIndent+Text(Tertiary).Render(strings.ToUpper(lang)))
			}

			continue
		}

		if coding {
			out = append(out, markdownIndent+Text(Tertiary).Render(quoteMark)+codeWell(l, family, w))

			continue
		}

		out = append(out, markdownLine(trimmed, l, w)...)
	}

	return out
}

// codeTab is what a tab is drawn as inside a well. Four columns: code in a
// pane has 84 of them, and eight spends a tenth of the line on one level.
const codeTab = "    "

// codeWell sets one line of a fenced block: the sunken paper, filled to the
// measure so the block reads as one shape rather than as lines of ragged
// length, and each run of the line in the colour syntax.go decided on.
//
// Every run carries the well's paper itself. A style rendered inside another
// closes with a reset, so a token painted on top of one background would take
// the rest of the row back to the window's own paper — which is the same
// reason the padding is a piece of the well and not a suffix on it.
func codeWell(line, family string, w int) string {
	// A tab is one character and four columns, so a well padded before the
	// terminal expands them comes out one step wider per level of indent.
	flat := pad(strings.ReplaceAll(line, "\t", codeTab), w, false)

	well := Surface(Sunken)

	var b strings.Builder

	for _, tok := range lexCode(flat, family) {
		ink := Text(Primary).GetForeground()
		if role, painted := codeRole(tok.part); painted {
			ink = Paint(role).GetForeground()
		}

		b.WriteString(well.Foreground(ink).Render(tok.text))
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
		for _, wl := range splitIntoLines(formatInlineMarkdown(quote), w-lipgloss.Width(proseRule)) {
			out = append(out, markdownIndent+Text(Tertiary).Render(proseRule)+wl)
		}

		return out
	}

	switch trimmed {
	case "---", "***", "___":
		return []string{markdownIndent + Text(Tertiary).Render(strings.Repeat("─", w))}
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
		{"# ", Paint(Accent).Bold(true)},
		{"## ", Text(Primary).Bold(true)},
		{"### ", Text(Secondary).Bold(true)},
	} {
		title, ok := strings.CutPrefix(trimmed, h.prefix)
		if !ok {
			continue
		}

		return []string{"", markdownIndent + h.style.Render(fit(title, w))}, true
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
			return bullet{"✔ ", Paint(OK), done}, true
		}

		if todo, box := strings.CutPrefix(text, "[ ] "); box {
			return bullet{"☐ ", Text(Tertiary), todo}, true
		}

		return bullet{"• ", Text(Tertiary), text}, true
	}

	if n, rest, found := strings.Cut(trimmed, ". "); found && isNumber(n) {
		return bullet{n + ". ", Text(Tertiary), rest}, true
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
	wrapped := splitIntoLines(formatInlineMarkdown(text), max(20, w-lead))

	out := make([]string, 0, len(wrapped))

	for i, wl := range wrapped {
		if i == 0 {
			out = append(out, markdownIndent+paint.Render(mark)+wl)
			continue
		}

		out = append(out, markdownIndent+strings.Repeat(" ", lead)+wl)
	}

	return out
}

// formatInlineMarkdown sets bold (**text**) and code (`text`) spans, and the
// words between them.
//
// The words between them are painted here rather than by wrapping the line
// in one style, because a span's own reset ends whatever style it was
// nested in and everything after it would fall back to the terminal's
// foreground.
func formatInlineMarkdown(s string) string {
	var b strings.Builder

	rest := s

	for {
		delim := firstDelim(rest)
		if delim == "" {
			break
		}

		before, tail, _ := strings.Cut(rest, delim)

		span, after, closed := strings.Cut(tail, delim)
		if !closed {
			break // an unpaired mark is a character somebody typed
		}

		if before != "" {
			b.WriteString(Text(Primary).Render(before))
		}

		if delim == "**" {
			b.WriteString(Text(Primary).Bold(true).Render(span))
		} else {
			b.WriteString(Paint(Accent).Render(span))
		}

		rest = after
	}

	if rest != "" {
		b.WriteString(Text(Primary).Render(rest))
	}

	return b.String()
}

// plainInline is a line with its inline marks taken out.
//
// A heading strip has one line for the title and no room to paint spans in
// it, so the marks that would have said "this is code" are read as the
// characters they are. Taking them out is what a label wants; painting them
// is what the panes do.
func plainInline(s string) string {
	return strings.NewReplacer("**", "", "`", "").Replace(s)
}

// firstDelim is whichever inline mark opens first, so that a backtick inside
// a bold run is set as part of it rather than closing something else.
func firstDelim(s string) string {
	bold, code := strings.Index(s, "**"), strings.Index(s, "`")

	switch {
	case bold < 0 && code < 0:
		return ""
	case bold < 0:
		return "`"
	case code < 0 || bold < code:
		return "**"
	default:
		return "`"
	}
}
