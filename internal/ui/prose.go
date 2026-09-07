package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/markdown"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// Typography: how a block of text is set. theme.go says what a piece of text
// means; this file says how much room it gets, where it breaks, and what sits
// beside it.
//
// The detail panes painted almost every line Dim and set every paragraph at
// the pane's full width. Dim is the furniture colour, so a paragraph in it
// carries the same weight as a keyboard hint, and a paragraph set 140 cells
// wide loses the reader at each line break. Reading what the model wrote is
// the whole job of these panes.

const (

	// paneGutter is the left margin every pane's content starts at.
	paneGutter = "  "

	// proseRule stands to the left of a paragraph the model wrote, so its
	// prose reads as quoted speech rather than as one more row of data.
	proseRule = "│ "
)

// section heads a block: the arrow that folds it, the label in the accent,
// what it is holding while it is closed, and a rule out to the edge. The
// rule does the work a box would do — it says where one block ends —
// without spending two more lines and four more corners on saying it.
//
// The note is drawn only on a closed section. An open one has its detail
// under it, and a count above detail that shows the same thing is a line the
// reader has to check against another line.
func section(label, note string, width int, open bool) string {
	head := paneGutter + theme.Text(theme.Tertiary).Render(cells.Fold(open)) +
		theme.Paint(theme.Accent).Bold(true).Render(strings.ToUpper(label)) + " "

	if !open && note != "" {
		head += theme.Text(theme.Tertiary).Render(note) + " "
	}

	fill := max(0, width-lipgloss.Width(head)-2*len(paneGutter))
	if fill == 0 {
		return head
	}

	return head + theme.Text(theme.Tertiary).Render(strings.Repeat("─", fill))
}

// meta sets the facts that qualify something — a cost, a duration, a verdict
// — as one dim line with middots between them. Empty parts drop out, so a
// caller can pass a value it does not always have without asking first.
func meta(parts ...string) string {
	kept := make([]string, 0, len(parts))

	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, p)
		}
	}

	if len(kept) == 0 {
		return ""
	}

	return strings.Join(kept, theme.Text(theme.Tertiary).Render(" · "))
}

// prose sets what the model wrote: wrapped at the measure, ruled down the
// left, and painted in nothing at all — the terminal's own foreground is the
// brightest thing available and this is the text the reader came for.
func prose(text string, width int, indent string) []string {
	measure := max(20, min(markdown.Measure, width-lipgloss.Width(indent)-len(proseRule)-2))
	rule := theme.Text(theme.Tertiary).Render(proseRule)

	var out []string

	for _, para := range strings.Split(strings.TrimSpace(text), "\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		for _, l := range cells.Lines(para, measure) {
			out = append(out, indent+rule+theme.Text(theme.Primary).Render(l))
		}
	}

	return out
}

// stat is one cell of the strip: a quiet label over a loud value.
type stat struct {
	label string
	value string
	role  theme.Role
}

// statStrip draws the numbers that answer "how did this go" as a row of
// bordered cells, the one place in the window where a box earns its lines:
// four figures side by side are read by comparing them, and a border is what
// tells the eye where one figure stops.
//
// Below sixty cells there is no room for four cells of anything, so the
// caller gets the same figures as one dim line instead of four cramped ones.
func statStrip(cells []stat, width int) []string {
	if len(cells) == 0 {
		return nil
	}

	if width < 60 {
		flat := make([]string, 0, len(cells))
		for _, c := range cells {
			flat = append(flat, theme.Paint(c.role).Render(c.value)+" "+theme.Text(theme.Tertiary).Render(strings.ToLower(c.label)))
		}

		return []string{paneGutter + meta(flat...)}
	}

	each := max(cardFloor, (width-2*len(paneGutter))/len(cells))
	drawn := make([]string, 0, len(cells))

	for _, c := range cells {
		drawn = append(drawn, strings.Join(
			card(c.label, []string{theme.Paint(c.role).Bold(true).Render(c.value)}, each), "\n"))
	}

	out := strings.Split(lipgloss.JoinHorizontal(lipgloss.Top, drawn...), "\n")
	for i, l := range out {
		out[i] = paneGutter + l
	}

	return out
}
