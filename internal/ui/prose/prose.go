// Package prose is how a block of text is set: how much room it gets, where
// it breaks, and what sits beside it.
//
// theme says what a piece of text means; this says what it is assembled
// into. The screens that draw a task all ask for the same shapes here, which
// is the difference between a window with a look and a dozen tabs that each
// invented one.
package prose

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/markdown"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// The detail panes painted almost every line Dim and set every paragraph at
// the pane's full width. Dim is the furniture colour, so a paragraph in it
// carries the same weight as a keyboard hint, and a paragraph set 140 cells
// wide loses the reader at each line break. Reading what the model wrote is
// the whole job of those panes.

const (

	// Gutter is the left margin every pane's content starts at.
	Gutter = "  "

	// rule stands to the left of a paragraph the model wrote, so its
	// prose reads as quoted speech rather than as one more row of data.
	rule = "│ "
)

// Section heads a block: the arrow that folds it, the label in the accent,
// what it is holding while it is closed, and a rule out to the edge. The
// rule does the work a box would do — it says where one block ends —
// without spending two more lines and four more corners on saying it.
//
// The note is drawn only on a closed section. An open one has its detail
// under it, and a count above detail that shows the same thing is a line the
// reader has to check against another line.
func Section(label, note string, width int, open bool) string {
	head := Gutter + theme.Text(theme.Tertiary).Render(cells.Fold(open)) +
		theme.Paint(theme.Accent).Bold(true).Render(strings.ToUpper(label)) + " "

	if !open && note != "" {
		head += theme.Text(theme.Tertiary).Render(note) + " "
	}

	fill := max(0, width-lipgloss.Width(head)-2*len(Gutter))
	if fill == 0 {
		return head
	}

	return head + theme.Text(theme.Tertiary).Render(strings.Repeat("─", fill))
}

// Meta sets the facts that qualify something — a cost, a duration, a verdict
// — as one dim line with middots between them. Empty parts drop out, so a
// caller can pass a value it does not always have without asking first.
func Meta(parts ...string) string {
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

// Quote sets what the model wrote: wrapped at the measure, ruled down the
// left, and painted in nothing at all — the terminal's own foreground is the
// brightest thing available and this is the text the reader came for.
func Quote(text string, width int, indent string) []string {
	measure := max(20, min(markdown.Measure, width-lipgloss.Width(indent)-len(rule)-2))
	ruled := theme.Text(theme.Tertiary).Render(rule)

	var out []string

	for _, para := range strings.Split(strings.TrimSpace(text), "\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		for _, l := range cells.Lines(para, measure) {
			out = append(out, indent+ruled+theme.Text(theme.Primary).Render(l))
		}
	}

	return out
}

// Stat is one cell of the strip: a quiet label over a loud value.
type Stat struct {
	Label string
	Value string
	Role  theme.Role
}

// Strip draws the numbers that answer "how did this go" as a row of
// bordered cells, the one place in the window where a box earns its lines:
// four figures side by side are read by comparing them, and a border is what
// tells the eye where one figure stops.
//
// Below sixty cells there is no room for four cells of anything, so the
// caller gets the same figures as one dim line instead of four cramped ones.
func Strip(stats []Stat, width int) []string {
	if len(stats) == 0 {
		return nil
	}

	if width < 60 {
		flat := make([]string, 0, len(stats))
		for _, c := range stats {
			flat = append(flat, theme.Paint(c.Role).Render(c.Value)+" "+
				theme.Text(theme.Tertiary).Render(strings.ToLower(c.Label)))
		}

		return []string{Gutter + Meta(flat...)}
	}

	each := max(cardFloor, (width-2*len(Gutter))/len(stats))
	drawn := make([]string, 0, len(stats))

	for _, c := range stats {
		drawn = append(drawn, strings.Join(
			Card(c.Label, []string{theme.Paint(c.Role).Bold(true).Render(c.Value)}, each), "\n"))
	}

	out := strings.Split(lipgloss.JoinHorizontal(lipgloss.Top, drawn...), "\n")
	for i, l := range out {
		out[i] = Gutter + l
	}

	return out
}
